package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"
)

// openAIChunk is the subset of one OpenAI-compatible chat completion chunk the
// adapter consumes. reasoning_content / thinking deltas are deliberately absent:
// the trial requires thinking to be off, and the adapter must never forward
// reasoning text to the caller even if the provider emits it.
type openAIChunk struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Content string `json:"content"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
}

// hasUsage reports whether the chunk carried any token counter. Ollama may omit
// usage entirely, so "no counters" must stay distinguishable from "zero
// counters" for the caller's usage_reported flag.
func (c openAIChunk) hasUsage() bool {
	return c.Usage.PromptTokens > 0 || c.Usage.CompletionTokens > 0 || c.Usage.TotalTokens > 0
}

// finishReason returns the chunk's finish reason when it carries one.
func (c openAIChunk) finishReason() string {
	if len(c.Choices) == 0 {
		return ""
	}
	return c.Choices[0].FinishReason
}

// CallTextStream generates OpenAI text response streaming.
func CallTextStream(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
) (<-chan string, <-chan error) {
	textChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(textChan)
		defer close(errChan)

		payload := buildChatPayload(provider, systemPrompt, userPrompt, options, true)

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			errChan <- err
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			errChan <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

		resp, err := client.Do(req)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			errChan <- apperror.NewUpstreamUnavailable(NewHTTPError(resp.StatusCode, fmt.Errorf("OpenAI Text Stream API error (HTTP %d): %s", resp.StatusCode, string(respBytes))))
			return
		}

		var (
			usage      app_ai.TokenUsage
			usageSeen  bool
			sawContent bool
			sawDone    bool
		)

		err = parseSSEStream(resp.Body, func(data string) error {
			if data == "[DONE]" {
				sawDone = true
				return io.EOF
			}

			var chunk openAIChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return NewMalformedError(fmt.Errorf("malformed SSE chunk: %w", err))
			}

			if chunk.hasUsage() {
				usage = app_ai.TokenUsage{
					PromptTokens:   chunk.Usage.PromptTokens,
					OutputTokens:   chunk.Usage.CompletionTokens,
					ThinkingTokens: 0,
				}
				usageSeen = true
			}
			if reason := chunk.finishReason(); reason != "" {
				usage.FinishReason = reason
			}

			if len(chunk.Choices) > 0 {
				if text := chunk.Choices[0].Delta.Content; text != "" {
					sawContent = true
					select {
					case <-ctx.Done():
						return ctx.Err()
					case textChan <- text:
					}
				}
			}
			if chunk.finishReason() == "length" {
				return NewMaxTokensError(fmt.Errorf("OpenAI text generation reached MAX_TOKENS"))
			}
			return nil
		})

		canceled := errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)
		// [DONE] aborts the parser with io.EOF; that is a clean end, not a failure.
		if err == io.EOF {
			err = nil
		}
		// EOF without [DONE] is a truncated stream, not a success: the provider
		// may have closed mid-answer. A canceled attempt is left alone so the
		// caller still classifies it as cancellation rather than malformed.
		if err == nil && !sawDone && ctx.Err() == nil {
			err = NewMalformedError(fmt.Errorf("SSE stream ended without [DONE]"))
		}
		// The winning provider must be reported even when Ollama sends no usage
		// payload; the caller stamps provider/model from this callback. A canceled
		// stream reports nothing because the attempt is not a success.
		if !canceled && options.OnUsage != nil {
			options.OnUsage(usage)
		}

		if err == nil && !sawContent && !usageSeen {
			err = NewMalformedError(fmt.Errorf("empty stream: no content and no usage reported"))
		}
		if err != nil && !canceled {
			errChan <- err
		}
	}()

	return textChan, errChan
}

// CallText generates OpenAI text response.
func CallText(
	ctx context.Context,
	client *http.Client,
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
) (string, error) {
	payload := buildChatPayload(provider, systemPrompt, userPrompt, options, false)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", apperror.NewUpstreamUnavailable(NewHTTPError(resp.StatusCode, fmt.Errorf("OpenAI Text API error (HTTP %d): %s", resp.StatusCode, string(respBytes))))
	}

	var openAIResp openAIChunk
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", apperror.NewUpstreamUnavailable(NewMalformedError(fmt.Errorf("malformed chat completion body: %w", err)))
	}

	// Usage is reported before the content checks so a truncated or malformed
	// answer still carries the winning provider's token counters.
	if options.OnUsage != nil {
		options.OnUsage(app_ai.TokenUsage{
			PromptTokens:   openAIResp.Usage.PromptTokens,
			OutputTokens:   openAIResp.Usage.CompletionTokens,
			ThinkingTokens: 0,
			FinishReason:   openAIResp.finishReason(),
		})
	}

	if len(openAIResp.Choices) == 0 {
		return "", apperror.NewUpstreamUnavailable(NewMalformedError(fmt.Errorf("chat completion carries no choices")))
	}
	if openAIResp.finishReason() == "length" {
		// A truncated answer must not be returned as content: it is an output
		// failure and the single Gemini fallback may run instead.
		return "", apperror.NewUpstreamUnavailable(NewMaxTokensError(fmt.Errorf("OpenAI text generation reached MAX_TOKENS")))
	}

	content := openAIResp.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return "", apperror.NewUpstreamUnavailable(NewMalformedError(fmt.Errorf("chat completion content is empty")))
	}
	if err := validateStructuredContent(content, options); err != nil {
		return "", apperror.NewUpstreamUnavailable(err)
	}

	return content, nil
}

// buildChatPayload assembles the OpenAI-compatible chat request shared by the
// streaming and non-streaming calls. The contract in
// specs/032-ollama-lightsail-trial/contracts/ollama-openai-subset.md section 2
// requires `temperature` to be sent even when it is 0, and `reasoning_effort`
// only when it is configured ("none" disables qwen3 thinking during the trial).
func buildChatPayload(
	provider config.APIProviderConfig,
	systemPrompt string,
	userPrompt string,
	options app_ai.TextGenerationOptions,
	stream bool,
) map[string]any {
	payload := map[string]any{
		"model": provider.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"stream":      stream,
		"temperature": options.Temperature,
	}
	if stream {
		payload["stream_options"] = map[string]any{"include_usage": true}
	}
	if options.MaxOutputTokens > 0 {
		payload["max_tokens"] = options.MaxOutputTokens
	}
	if effort := strings.TrimSpace(provider.ReasoningEffort); effort != "" {
		payload["reasoning_effort"] = effort
	}
	applyOpenAIStructuredOutput(payload, options)
	return payload
}

// validateStructuredContent checks a structured answer before it is returned.
// It deliberately avoids a schema library: the trial only needs "is this valid
// JSON" plus the top-level keys the caller marked as required. A failure is a
// schema_invalid outcome, which is eligible for the one fallback attempt.
func validateStructuredContent(content string, options app_ai.TextGenerationOptions) error {
	if options.ResponseMIMEType != "application/json" {
		return nil
	}
	if !json.Valid([]byte(content)) {
		return NewSchemaInvalidError(fmt.Errorf("structured response is not valid JSON"))
	}

	schema, ok := options.ResponseSchema.(map[string]any)
	if !ok {
		return nil
	}
	required, ok := stringList(schema["required"])
	if !ok || len(required) == 0 {
		return nil
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		return NewSchemaInvalidError(fmt.Errorf("structured response is not a JSON object: %w", err))
	}
	for _, key := range required {
		if _, present := decoded[key]; !present {
			return NewSchemaInvalidError(fmt.Errorf("structured response is missing required key %q", key))
		}
	}
	return nil
}

// stringList accepts the shapes a JSON-schema "required" array can take once it
// has travelled through YAML configuration: []string or []any of strings.
func stringList(value any) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return typed, true
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, text)
		}
		return out, true
	default:
		return nil, false
	}
}

// CallVision analyzes image content.
func CallVision(ctx context.Context, client *http.Client, provider config.APIProviderConfig, imageUrl string, prompt string) (string, error) {
	text, _, err := CallVisionWithUsage(ctx, client, provider, imageUrl, prompt)
	return text, err
}

// CallVisionWithUsage analyzes image content and reports the provider usage.
// The OpenAI chat vision response carries no token usage payload, so the
// returned usage is always zero (with the finish reason when present). Callers
// that need billable tokens must treat this as unreported usage.
func CallVisionWithUsage(ctx context.Context, client *http.Client, provider config.APIProviderConfig, imageUrl string, prompt string) (string, app_ai.TokenUsage, error) {
	payload := map[string]any{
		"model": provider.Model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": imageUrl,
						},
					},
				},
			},
		},
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", app_ai.TokenUsage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", app_ai.TokenUsage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", app_ai.TokenUsage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", app_ai.TokenUsage{}, apperror.NewUpstreamUnavailable(NewHTTPError(resp.StatusCode, fmt.Errorf("OpenAI Vision API error (HTTP %d): %s", resp.StatusCode, string(respBytes))))
	}

	var openAIResp struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", app_ai.TokenUsage{}, err
	}

	if len(openAIResp.Choices) == 0 {
		return "", app_ai.TokenUsage{}, apperror.NewInternalError("Không thể nhận kết quả phân tích hình ảnh từ hệ thống trí tuệ nhân tạo.")
	}

	usage := app_ai.TokenUsage{}
	if openAIResp.Choices[0].FinishReason != "" {
		usage.FinishReason = openAIResp.Choices[0].FinishReason
	}

	return openAIResp.Choices[0].Message.Content, usage, nil
}

// CallEmbeddingBatch generates batch text embeddings.
func CallEmbeddingBatch(ctx context.Context, client *http.Client, provider config.APIProviderConfig, chunks []string) ([][]float32, error) {
	var allEmbeddings [][]float32
	maxBatchSize := 100

	for i := 0; i < len(chunks); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		subSlice := chunks[i:end]

		payload := map[string]any{
			"input":      subSlice,
			"model":      provider.Model,
			"dimensions": 768,
		}

		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST", provider.Endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+provider.ApiKey)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			return nil, apperror.NewUpstreamUnavailable(NewHTTPError(resp.StatusCode, fmt.Errorf("OpenAI Embedding Batch API error (HTTP %d): %s", resp.StatusCode, string(respBytes))))
		}

		var openAIResp struct {
			Data []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
			return nil, err
		}

		if len(openAIResp.Data) == 0 {
			return nil, apperror.NewInternalError("Không thể xử lý dữ liệu đặc trưng của nội dung lúc này.")
		}

		batchResults := make([][]float32, len(subSlice))
		for _, d := range openAIResp.Data {
			if d.Index >= 0 && d.Index < len(subSlice) {
				batchResults[d.Index] = sliceutils.AdjustVectorLength(d.Embedding, 768)
			}
		}

		allEmbeddings = append(allEmbeddings, batchResults...)
	}

	return allEmbeddings, nil
}

func applyOpenAIStructuredOutput(payload map[string]any, options app_ai.TextGenerationOptions) {
	if options.ResponseMIMEType != "application/json" {
		return
	}
	if options.ResponseSchema != nil {
		schema := normalizeOpenAISchemaTypes(options.ResponseSchema)
		payload["response_format"] = map[string]any{"type": "json_schema", "json_schema": map[string]any{"name": "structured_response", "strict": true, "schema": schema}}
		return
	}
	payload["response_format"] = map[string]any{"type": "json_object"}
}

// normalizeOpenAISchemaTypes lowercases every JSON-schema "type" value because
// OpenAI-compatible servers such as Ollama reject the uppercase names Gemini
// uses. It copies maps and slices so the caller's schema stays untouched for
// providers that expect the uppercase form.
func normalizeOpenAISchemaTypes(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			if key == "type" {
				if name, ok := item.(string); ok {
					normalized[key] = strings.ToLower(name)
					continue
				}
			}
			normalized[key] = normalizeOpenAISchemaTypes(item)
		}
		return normalized
	case []any:
		normalized := make([]any, len(typed))
		for index, item := range typed {
			normalized[index] = normalizeOpenAISchemaTypes(item)
		}
		return normalized
	default:
		return value
	}
}

func parseSSEStream(r io.Reader, callback func(data string) error) error {
	var buffer []byte
	temp := make([]byte, 4096)

	for {
		n, err := r.Read(temp)
		if n > 0 {
			buffer = append(buffer, temp[:n]...)
			for {
				lineEnd := bytes.IndexByte(buffer, '\n')
				if lineEnd < 0 {
					break
				}
				line := string(buffer[:lineEnd])
				buffer = buffer[lineEnd+1:]

				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					if err := callback(data); err != nil {
						return err
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return nil
}
