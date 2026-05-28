# 📑 Gemini Embedding Implementation Guidelines & Rate Limit Protection

This document outlines the strict technical constraints and architectural requirements for implementing text embedding generation using the Google Gemini API (specifically the `text-embedding-004` model) within the SmartWardrobe Backend system.

All AI Agents must read and strictly adhere to these implementation rules to prevent hitting Google's severe Rate Limits (RPD, RPM, TPM) and causing application downtime.

---

## 🛑 The Rate Limit Challenge

When processing a large volume of text blocks or splitting long documents into numerous segments (e.g., over 50 or 100 chunks), executing a single HTTP request per chunk within a sequential or concurrent loop will instantly deplete the Gemini API quotas:

- **RPM (Requests Per Minute):** Flooding the API with dozens of individual requests triggers an immediate `429 Too Many Requests` status code.
- **RPD (Requests Per Day):** Processing chunks individually causes the daily request allowance to be consumed entirely within minutes.
- **TPM (Tokens Per Minute):** Unregulated concurrent spikes exceed the allowed token throughput before the system has time to replenish the window.

---

## 🛠️ Strict Implementation Rules

### Rule 1: Mandatory Batch Embedding

- Never invoke individual embedding generation methods inside a raw loop for bulk data processing.
- You must utilize the official Gemini batching endpoint (`BatchEmbedContents`) provided by the `github.com/google/generative-ai-go/genai` package.
- Group chunk arrays into a single batch request, packing up to a maximum of 100 chunks per single HTTP request. This architectural choice compresses 100 logical requests into 1 physical request against the Gemini server.

### Rule 2: Controlled Concurrency via Worker Pool

- If the total number of chunks exceeds the single batch capacity (e.g., hundreds of chunks), you must implement a structured Worker Pool or use a rate-limited channel mechanism.
- Control the maximum number of concurrent batch requests flying out at the same time.
- Introduce an intentional pacing or micro-cooldown delay between consecutive batch dispatches if the system detects an exceptionally heavy payload workload.

### Rule 3: Graceful Error Recovery and Exponential Backoff

- All code interacting with the Gemini API must handle `429 Too Many Requests` errors gracefully.
- Implement a robust retry mechanism utilizing an Exponential Backoff strategy to allow the server-side API rate-limiting windows to recover before attempting a redelivery.

### Rule 4: Clean Code Commenting Constraints

- When writing or updating Golang source code, scripts, or comments for the embedding infrastructure, you must absolutely NOT include any sequential numbering (such as 1., 2., 3., 01., 02.) within the comment blocks. Use plain text descriptive headings to explain logic, ensuring future flexibility when reordering blocks.
- You must absolutely NOT use any emojis, icons, or visual symbols inside any source code comments.

---

## 📝 Reference Code Blueprint (Golang)

The following implementation serves as the architectural standard for the Identity/Outfit module when communicating with the Gemini embedding service:

```go
package ai

import (
	"context"
	"fmt"

	"[github.com/google/generative-ai-go/genai](https://github.com/google/generative-ai-go/genai)"
)

type GeminiEmbeddingService struct {
	client *genai.Client
}

func NewGeminiEmbeddingService(client *genai.Client) *GeminiEmbeddingService {
	return &GeminiEmbeddingService{client: client}
}

func (s *GeminiEmbeddingService) GenerateEmbeddingsInBatches(ctx context.Context, chunks []string) ([][]float32, error) {
	model := s.client.EmbeddingModel("text-embedding-004")
	var finalEmbeddings [][]float32

	maxBatchSize := 100
	chunksLength := len(chunks)

	// Process chunks in safe slices to respect the batch capacity limits
	for i := 0; i < chunksLength; i += maxBatchSize {
		end := i + maxBatchSize
		if end > chunksLength {
			end = chunksLength
		}

		batch := model.NewBatch()
		for _, chunk := range chunks[i:end] {
			batch.AddContent(genai.Text(chunk))
		}

		// Execute a single batched HTTP request to protect API quotas
		res, err := model.BatchEmbedContents(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("gemini batch embedding failed at slice offset: %w", err)
		}

		// Extract the resulting vector coordinates sequentially
		for _, emb := range res.Embeddings {
			finalEmbeddings = append(finalEmbeddings, emb.Values)
		}
	}

	return finalEmbeddings, nil
}
```
