# AI Embedding & Provider Fallback Guidelines

This document reflects the current AI integration architecture implemented in the backend.

The codebase no longer follows a Gemini-only embedding design. Instead, it uses a provider-agnostic AI service with configurable primary and fallback providers for:

- vision analysis
- embeddings
- text generation
- streaming text generation

Current provider identifiers supported by the shared AI service:

- `google`
- `openai`

---

## 1. Current architecture

The shared AI entry point is:

- `internal/shared/application/ai/interfaces.go`
- `internal/shared/infrastructure/ai/core.go`

The service is configured from `config.AI` with separate primary/fallback settings for:

- `VisionPrimary` and `VisionFallback`
- `EmbeddingPrimary` and `EmbeddingFallback`
- `TextPrimary` and `TextFallback`

This means technical docs must not claim that embeddings are permanently tied to Gemini or to a single SDK.

---

## 2. Rate-limit protection strategy actually used

The current implementation protects external AI APIs through these mechanisms:

- a shared request-per-minute limiter using `golang.org/x/time/rate`
- batched embedding requests up to 100 inputs per outbound call
- provider fallback from primary to fallback on eligible failures
- worker-level retry with exponential backoff for background wardrobe digitization jobs

The limiter is global at AI service level and is applied before outbound calls for image analysis, embeddings, text generation, and streaming text.

---

## 3. Embedding batching rules

`GenerateEmbeddings(ctx, chunks)` already accepts multiple chunks and batches them internally.

Current implementation details:

- empty input returns early
- chunks are split into batches of at most 100
- Google uses `:batchEmbedContents`
- OpenAI uses a single embeddings request with `input: []string`
- vectors are normalized to length 768 before returning to callers

Feature code should pass chunk arrays to the shared AI service rather than firing one request per chunk manually.

---

## 4. Provider fallback rules

The shared helper `executeWithFallback(...)` is the current fallback mechanism.

Behavior:

- primary provider is attempted first
- fallback provider is used when configured and when the primary failure is not treated as a hard bad-request failure
- if both fail, the operation returns the fallback error

Streaming text follows a similar principle:

- primary stream starts first
- if the primary stream fails before yielding content and a fallback exists, the service retries with fallback streaming

---

## 5. Wardrobe digitization workflow implications

For wardrobe batch upload processing, the implemented flow is:

- worker consumes RabbitMQ job
- categories are loaded from DB
- image is analyzed by the shared AI service
- metadata is parsed
- rich text context is built
- embeddings are generated through the shared AI service
- HSL metadata is derived locally
- wardrobe item is updated and a follow-up event is published

Retry behavior for transient failures is implemented at worker level with delayed re-publish, not inside a Gemini-only client wrapper.

---

## 6. Documentation guardrails

When updating technical docs or code comments, keep these statements accurate:

- embeddings are provider-agnostic at architecture level
- batching is mandatory for bulk embedding calls
- fallback configuration is part of the official runtime design
- rate limiting exists at shared AI service level
- worker retries are part of the async processing design

Do not reintroduce documentation that says:

- Gemini is the only supported embedding provider
- the system uses only the official Gemini Go SDK
- embedding generation is implemented as one request per chunk
