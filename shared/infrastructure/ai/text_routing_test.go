package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	app_ai "smart-wardrobe-be/shared/application/ai"
	"smart-wardrobe-be/shared/config"

	"go.uber.org/zap"
)

// Routing regression tests for feature 032 user story 1: the three text
// capabilities go to the Ollama proxy (provider "openai"), vision/embedding stay
// on Gemini, summary/rewriter follow their business route, and a text request
// makes at most two attempts with no third free failover.

const (
	openAITrialPath = "/v1/chat/completions"
	trialModelName  = "qwen3:4b"
	geminiTextReply = `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"from-gemini"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2,"totalTokenCount":5}}`
)

// stubLogger satisfies logger.Interface without writing to stdout.
type stubLogger struct{}

func (s *stubLogger) Debug(string, ...zap.Field) {}
func (s *stubLogger) Info(string, ...zap.Field)  {}
func (s *stubLogger) Warn(string, ...zap.Field)  {}
func (s *stubLogger) Error(string, ...zap.Field) {}
func (s *stubLogger) Fatal(string, ...zap.Field) {}

// openAIEndpoint builds the full endpoint the trial env uses.
func openAIEndpoint(server *httptest.Server) string { return server.URL + openAITrialPath }

// openAIProvider is the trial primary: Ollama behind the TLS/Bearer proxy.
func openAIProvider(server *httptest.Server) config.APIProviderConfig {
	return config.APIProviderConfig{
		Provider:        ProviderOpenAI,
		ApiKey:          "trial-bearer",
		Endpoint:        openAIEndpoint(server),
		Model:           trialModelName,
		ReasoningEffort: "none",
	}
}

// geminiProvider is the single fallback (and the vision/embedding primary).
func geminiProvider(server *httptest.Server, model string) config.APIProviderConfig {
	return config.APIProviderConfig{
		Provider: ProviderGemini,
		ApiKey:   "gemini-key",
		Endpoint: server.URL,
		Model:    model,
	}
}

// routingTestService builds an AIService wired like the trial: openai text
// primary, google fallback, google vision/embedding.
func routingTestService(t *testing.T, primary, fallback, vision, embedding config.APIProviderConfig) *AIService {
	t.Helper()

	cfg := &config.Config{}
	cfg.AI.ChatTextPrimary = primary
	cfg.AI.ChatTextFallback = fallback
	cfg.AI.RecommendationTextPrimary = primary
	cfg.AI.RecommendationTextFallback = fallback
	cfg.AI.FreeTextPrimary = primary
	cfg.AI.FreeTextFallback = fallback
	cfg.AI.VisionPrimary = vision
	cfg.AI.EmbeddingPrimary = embedding
	cfg.AI.ChatTextTimeoutSeconds = 5
	cfg.AI.RecommendationTextTimeoutSeconds = 5
	cfg.AI.FreeTextTimeoutSeconds = 5
	cfg.AI.VisionTimeoutSeconds = 5
	cfg.AI.EmbeddingTimeoutSeconds = 5
	cfg.AI.TextPrimaryTimeoutSeconds = 5
	cfg.AI.TextFallbackTimeoutSeconds = 5
	cfg.AI.Billing.UnbilledCapabilities = []string{config.CapabilityFreeText}

	return &AIService{
		cfg:                       cfg,
		chatTextClient:            &http.Client{},
		recommendationTextClient:  &http.Client{},
		freeTextClient:            &http.Client{},
		visionClient:              &http.Client{},
		embeddingClient:           &http.Client{},
		chatTextLimiter:           newRPMLimiter(6000, 1),
		recommendationTextLimiter: newRPMLimiter(6000, 1),
		freeTextLimiter:           newRPMLimiter(6000, 1),
		visionLimiter:             newRPMLimiter(6000, 1),
		embeddingLimiter:          newRPMLimiter(6000, 1),
		textSemaphore:             make(chan struct{}, 2),
		logger:                    &stubLogger{},
	}
}

// openAIChatServer answers every chat completion with a fixed content.
func openAIChatServer(t *testing.T, hits *int32, status int, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if status != http.StatusOK {
			http.Error(w, "upstream failure", status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"choices":[{"message":{"content":%q},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}}`, content)
	}))
}

// geminiChatServer answers a Gemini generateContent call with a fixed text.
func geminiChatServer(t *testing.T, hits *int32, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if status != http.StatusOK {
			http.Error(w, "gemini failure", status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, geminiTextReply)
	}))
}

func TestTextCapabilitiesRouteToOpenAI(t *testing.T) {
	cfg := &config.Config{}
	cfg.AI.ChatTextPrimary = config.APIProviderConfig{Provider: ProviderOpenAI, ApiKey: "chat-key", Endpoint: "https://ollama.test" + openAITrialPath, Model: trialModelName}
	cfg.AI.ChatTextFallback = config.APIProviderConfig{Provider: ProviderGemini, Model: "gemini-text"}
	cfg.AI.RecommendationTextPrimary = config.APIProviderConfig{Provider: ProviderOpenAI, ApiKey: "rec-key", Endpoint: "https://ollama.test" + openAITrialPath, Model: trialModelName}
	cfg.AI.RecommendationTextFallback = config.APIProviderConfig{Provider: ProviderGemini, Model: "gemini-text"}
	cfg.AI.FreeTextPrimary = config.APIProviderConfig{Provider: ProviderOpenAI, ApiKey: "free-key", Endpoint: "https://ollama.test" + openAITrialPath, Model: trialModelName}
	cfg.AI.FreeTextFallback = config.APIProviderConfig{Provider: ProviderGemini, Model: "gemini-text"}

	s := &AIService{cfg: cfg}

	chat := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}, app_ai.AIOperationChat)
	if chat.primary.provider.Provider != ProviderOpenAI {
		t.Errorf("chat_text primary = %q, want openai", chat.primary.provider.Provider)
	}
	if chat.fallback.provider.Provider != ProviderGemini {
		t.Errorf("chat_text fallback = %q, want google", chat.fallback.provider.Provider)
	}

	rec := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}, app_ai.AIOperationOutfit)
	if rec.primary.provider.Provider != ProviderOpenAI || rec.primary.capability != config.CapabilityRecommendationText {
		t.Errorf("recommendation_text primary = %+v, want openai/recommendation_text", rec.primary)
	}

	free := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierFree}, app_ai.AIOperationChat)
	if free.primary.provider.Provider != ProviderOpenAI || free.primary.capability != config.CapabilityFreeText {
		t.Errorf("free_text primary = %+v, want openai/free_text", free.primary)
	}
}

// Summary and rewriter are not their own capability: the orchestrator routes them
// through chat_text (paid) or free_text (free). In the trial both point at the
// Ollama proxy, so the resolved attempt is openai either way.
func TestSummaryAndRewriterFollowBusinessRoute(t *testing.T) {
	cfg := &config.Config{}
	cfg.AI.ChatTextPrimary = config.APIProviderConfig{Provider: ProviderOpenAI, Endpoint: "https://ollama.test" + openAITrialPath, Model: trialModelName}
	cfg.AI.ChatTextFallback = config.APIProviderConfig{Provider: ProviderGemini, Model: "gemini-text"}
	cfg.AI.FreeTextPrimary = config.APIProviderConfig{Provider: ProviderOpenAI, Endpoint: "https://ollama.test" + openAITrialPath, Model: trialModelName}
	cfg.AI.FreeTextFallback = config.APIProviderConfig{Provider: ProviderGemini, Model: "gemini-text"}

	s := &AIService{cfg: cfg}

	for _, operation := range []string{app_ai.AIOperationSummary, app_ai.AIOperationRewriter} {
		// The orchestrator sends summary/rewriter as non-outfit text, so the
		// operation resolves to the CHAT_TEXT / FREE_TEXT groups.
		paid := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid, Operation: operation}, app_ai.AIOperationChat)
		if paid.primary.capability != config.CapabilityChatText {
			t.Errorf("%s paid capability = %q, want chat_text", operation, paid.primary.capability)
		}
		if paid.primary.provider.Provider != ProviderOpenAI {
			t.Errorf("%s paid provider = %q, want openai in the trial", operation, paid.primary.provider.Provider)
		}

		free := s.resolveTextCallPlan(app_ai.TextGenerationOptions{Tier: app_ai.TextTierFree, Operation: operation}, app_ai.AIOperationChat)
		if free.primary.capability != config.CapabilityFreeText {
			t.Errorf("%s free capability = %q, want free_text", operation, free.primary.capability)
		}
		if free.primary.provider.Provider != ProviderOpenAI {
			t.Errorf("%s free provider = %q, want openai in the trial", operation, free.primary.provider.Provider)
		}
	}
}

func TestVisionAndEmbeddingStayOnGoogle(t *testing.T) {
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("downloadable-image-bytes"))
	}))
	defer imageServer.Close()

	var (
		visionHits    int32
		embeddingHits int32
	)
	visionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&visionHits, 1)
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Errorf("vision path = %q, want a Gemini generateContent path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"{\"styles\":[]}"}]}}],"usageMetadata":{"promptTokenCount":7,"candidatesTokenCount":4,"totalTokenCount":11}}`)
	}))
	defer visionServer.Close()

	embeddingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&embeddingHits, 1)
		if !strings.Contains(r.URL.Path, ":batchEmbedContents") {
			t.Errorf("embedding path = %q, want a Gemini batchEmbedContents path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"embeddings":[{"values":[0.1,0.2,0.3]}]}`)
	}))
	defer embeddingServer.Close()

	s := routingTestService(t,
		openAIProvider(visionServer), // text primary must not be used by vision/embedding
		geminiProvider(visionServer, "gemini-text"),
		geminiProvider(visionServer, "gemini-vision"),
		geminiProvider(embeddingServer, "gemini-embedding"),
	)

	text, usage, err := s.AnalyzeImageWithUsage(context.Background(), imageServer.URL+"/photo.png", "describe")
	if err != nil {
		t.Fatalf("AnalyzeImageWithUsage returned error: %v", err)
	}
	if !strings.Contains(text, "styles") {
		t.Errorf("vision text = %q, want the Gemini answer", text)
	}
	if usage.Provider != ProviderGemini || usage.Model != "gemini-vision" {
		t.Errorf("vision usage = %+v, want google/gemini-vision", usage)
	}
	if atomic.LoadInt32(&visionHits) != 1 {
		t.Errorf("vision hits = %d, want 1 (no Ollama attempt)", visionHits)
	}

	vectors, err := s.GenerateEmbeddings(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("GenerateEmbeddings returned error: %v", err)
	}
	if len(vectors) != 1 || len(vectors[0]) != 768 {
		t.Fatalf("embeddings = %d vectors, want 1 padded to 768", len(vectors))
	}
	if atomic.LoadInt32(&embeddingHits) != 1 {
		t.Errorf("embedding hits = %d, want 1 (no Ollama attempt)", embeddingHits)
	}
}

func TestTextRequestFallsBackOnceOn429(t *testing.T) {
	var primaryHits, fallbackHits, freeHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusTooManyRequests, "")
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusOK)
	defer fallback.Close()
	free := geminiChatServer(t, &freeHits, http.StatusOK)
	defer free.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)
	// A free key is configured; it must never be used as a third attempt.
	s.cfg.AI.FreeTextPrimary = geminiProvider(free, "free-text")

	text, err := s.GenerateChatText(context.Background(), "system", "user", app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid})
	if err != nil {
		t.Fatalf("GenerateChatText returned error: %v", err)
	}
	if text != "from-gemini" {
		t.Errorf("text = %q, want the fallback answer", text)
	}
	if atomic.LoadInt32(&primaryHits) != 1 || atomic.LoadInt32(&fallbackHits) != 1 {
		t.Errorf("attempts = primary %d / fallback %d, want exactly 1 each", primaryHits, fallbackHits)
	}
	if atomic.LoadInt32(&freeHits) != 0 {
		t.Errorf("free key hits = %d, want 0 (no third attempt)", freeHits)
	}
}

func TestTextRequestMakesAtMostTwoAttempts(t *testing.T) {
	var primaryHits, fallbackHits, freeHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusInternalServerError, "")
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusInternalServerError)
	defer fallback.Close()
	free := geminiChatServer(t, &freeHits, http.StatusOK)
	defer free.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)
	s.cfg.AI.FreeTextPrimary = geminiProvider(free, "free-text")

	if _, err := s.GenerateChatText(context.Background(), "system", "user", app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}); err == nil {
		t.Fatal("want an error when both providers fail")
	}
	if atomic.LoadInt32(&primaryHits) != 1 || atomic.LoadInt32(&fallbackHits) != 1 {
		t.Errorf("attempts = primary %d / fallback %d, want exactly 2 calls total", primaryHits, fallbackHits)
	}
	if atomic.LoadInt32(&freeHits) != 0 {
		t.Errorf("free key hits = %d, want 0", freeHits)
	}
}

func TestTextRequestDoesNotFallBackOnClientError(t *testing.T) {
	var primaryHits, fallbackHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusBadRequest, "")
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusOK)
	defer fallback.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)

	if _, err := s.GenerateChatText(context.Background(), "system", "user", app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid}); err == nil {
		t.Fatal("want the 400 to surface as an error")
	}
	if atomic.LoadInt32(&fallbackHits) != 0 {
		t.Errorf("fallback hits = %d, want 0: a non-429 4xx is not fallback-eligible", fallbackHits)
	}
}

func TestTextRequestDoesNotFallBackWhenCanceled(t *testing.T) {
	var primaryHits, fallbackHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusOK, "unused")
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusOK)
	defer fallback.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.GenerateChatText(ctx, "system", "user", app_ai.TextGenerationOptions{Tier: app_ai.TextTierPaid})
	if err == nil {
		t.Fatal("a canceled request must fail")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if atomic.LoadInt32(&fallbackHits) != 0 {
		t.Errorf("fallback hits = %d, want 0 on cancellation", fallbackHits)
	}
}

func TestStreamDoesNotFallBackAfterFirstChunk(t *testing.T) {
	var primaryHits, fallbackHits int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&primaryHits, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Xin\"}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		// Malformed chunk after content: the attempt fails, but a fallback would
		// splice two answers together, so it must not run.
		fmt.Fprint(w, "data: not-json\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusOK)
	defer fallback.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)

	text, errc := s.GenerateChatTextStream(context.Background(), "system", "user", app_ai.TextGenerationOptions{})

	var streamed strings.Builder
	var streamErr error
	deadline := time.After(5 * time.Second)
	for text != nil || errc != nil {
		select {
		case chunk, ok := <-text:
			if !ok {
				text = nil
				continue
			}
			streamed.WriteString(chunk)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			if err != nil {
				streamErr = err
			}
		case <-deadline:
			t.Fatal("timed out draining the stream")
		}
	}

	if streamed.String() != "Xin" {
		t.Errorf("streamed text = %q, want the first chunk only", streamed.String())
	}
	if streamErr == nil {
		t.Error("want an error surfaced on the stream")
	}
	if atomic.LoadInt32(&fallbackHits) != 0 {
		t.Errorf("fallback hits = %d, want 0 after the first chunk", fallbackHits)
	}
}

func TestAttemptHooksRunBeforeAndAfterEveryAttempt(t *testing.T) {
	var primaryHits, fallbackHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusServiceUnavailable, "")
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusOK)
	defer fallback.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)

	var (
		before []app_ai.AttemptMeta
		after  []app_ai.AttemptResult
	)
	hooks := &app_ai.AttemptHooks{
		BeforeAttempt: func(_ context.Context, meta app_ai.AttemptMeta) error {
			before = append(before, meta)
			return nil
		},
		AfterAttempt: func(_ context.Context, _ app_ai.AttemptMeta, result app_ai.AttemptResult) {
			after = append(after, result)
		},
	}

	if _, err := s.GenerateChatText(context.Background(), "system", "user", app_ai.TextGenerationOptions{
		Tier:  app_ai.TextTierPaid,
		Hooks: hooks,
	}); err != nil {
		t.Fatalf("GenerateChatText returned error: %v", err)
	}

	if len(before) != 2 || len(after) != 2 {
		t.Fatalf("hook calls = before %d / after %d, want 2/2 (one per attempt)", len(before), len(after))
	}
	if before[0].AttemptIndex != 1 || before[1].AttemptIndex != 2 {
		t.Errorf("attempt indexes = %d,%d, want 1,2", before[0].AttemptIndex, before[1].AttemptIndex)
	}
	if before[0].Provider != ProviderOpenAI || before[1].Provider != ProviderGemini {
		t.Errorf("hook providers = %q,%q, want openai then google", before[0].Provider, before[1].Provider)
	}
	if before[0].Timeout <= 0 {
		t.Error("the attempt timeout must be reported to the hook")
	}
	if after[0].Outcome == "success" || !after[0].FallbackEligible {
		t.Errorf("first result = %+v, want a fallback-eligible failure", after[0])
	}
	if after[1].Outcome != "success" {
		t.Errorf("second result outcome = %q, want success", after[1].Outcome)
	}
}

func TestBeforeAttemptErrorSkipsProviderCall(t *testing.T) {
	var primaryHits, fallbackHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusOK, "unused")
	defer primary.Close()
	fallback := geminiChatServer(t, &fallbackHits, http.StatusOK)
	defer fallback.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		config.APIProviderConfig{},
		config.APIProviderConfig{},
	)

	blocked := errors.New("admission rejected")
	hooks := &app_ai.AttemptHooks{
		BeforeAttempt: func(context.Context, app_ai.AttemptMeta) error { return blocked },
	}

	_, err := s.GenerateChatText(context.Background(), "system", "user", app_ai.TextGenerationOptions{Hooks: hooks})
	if !errors.Is(err, blocked) {
		t.Fatalf("error = %v, want the BeforeAttempt error", err)
	}
	if atomic.LoadInt32(&primaryHits) != 0 {
		t.Errorf("provider hits = %d, want 0 when admission fails", primaryHits)
	}
	// An admission rejection is not a provider failure: the fallback must not be
	// attempted, or it would bypass the decision that just refused the request.
	if atomic.LoadInt32(&fallbackHits) != 0 {
		t.Errorf("fallback hits = %d, want 0 after an admission rejection", fallbackHits)
	}
}

func TestStreamFallsBackBeforeFirstChunk(t *testing.T) {
	var primaryHits, fallbackHits int32
	primary := openAIChatServer(t, &primaryHits, http.StatusServiceUnavailable, "")
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&fallbackHits, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, line := range []string{
			`{"candidates":[{"content":{"parts":[{"text":"from-"}]}}]}`,
			`{"candidates":[{"content":{"parts":[{"text":"gemini"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2,"totalTokenCount":5}}`,
		} {
			fmt.Fprintf(w, "data: %s\n\n", line)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer fallback.Close()

	s := routingTestService(t,
		openAIProvider(primary),
		geminiProvider(fallback, "gemini-text"),
		geminiProvider(fallback, "gemini-vision"),
		geminiProvider(fallback, "gemini-embedding"),
	)

	text, errc := s.GenerateChatTextStream(context.Background(), "system", "user", app_ai.TextGenerationOptions{})

	var streamed strings.Builder
	var streamErr error
	deadline := time.After(5 * time.Second)
	for text != nil || errc != nil {
		select {
		case chunk, ok := <-text:
			if !ok {
				text = nil
				continue
			}
			streamed.WriteString(chunk)
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			if err != nil {
				streamErr = err
			}
		case <-deadline:
			t.Fatal("timed out draining the stream")
		}
	}

	if streamErr != nil {
		t.Fatalf("stream returned error: %v", streamErr)
	}
	if streamed.String() != "from-gemini" {
		t.Errorf("streamed text = %q, want the fallback answer", streamed.String())
	}
	if atomic.LoadInt32(&primaryHits) != 1 || atomic.LoadInt32(&fallbackHits) != 1 {
		t.Errorf("attempts = primary %d / fallback %d, want exactly 1 each", primaryHits, fallbackHits)
	}
}
