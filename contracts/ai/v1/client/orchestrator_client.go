// Package client provides the outbound gRPC adapter for the AI orchestrator
// contract, implementing the same port as the in-process implementation so
// callers do not change when the service boundary is crossed (FR-019).
package client

import (
	"context"
	"io"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	grpcx "smart-wardrobe-be/contracts/grpcx"
	aicontract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	aitransport "smart-wardrobe-be/services/ai/contract/aiexecutor/transport"
	aidto "smart-wardrobe-be/services/ai/dto/aiexecutor"

	"google.golang.org/grpc"
)

// AIOrchestratorClient is the remote adapter for IAIOrchestrator.
type AIOrchestratorClient struct {
	svc aiv1.AIOrchestratorServiceClient
}

var _ aicontract.IAIOrchestrator = (*AIOrchestratorClient)(nil)

// NewAIOrchestratorClient builds the AI orchestrator contract over gRPC.
func NewAIOrchestratorClient(cc grpc.ClientConnInterface) *AIOrchestratorClient {
	return &AIOrchestratorClient{svc: aiv1.NewAIOrchestratorServiceClient(cc)}
}

func (c *AIOrchestratorClient) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	res, err := c.svc.AnalyzeImage(ctx, &aiv1.AnalyzeImageRequest{ImageUrl: imageUrl, Prompt: prompt})
	if err != nil {
		return "", grpcx.ToAppError(err)
	}
	return res.GetResult(), nil
}

func (c *AIOrchestratorClient) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	res, err := c.svc.GenerateEmbeddings(ctx, &aiv1.GenerateEmbeddingsRequest{Chunks: chunks})
	if err != nil {
		return nil, grpcx.ToAppError(err)
	}
	return aitransport.EmbeddingsFromProto(res.GetVectors()), nil
}

func (c *AIOrchestratorClient) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string, options aidto.OrchestratorOptions) (string, error) {
	req, err := aitransport.ChatTextRequestToProto(systemPrompt, userPrompt, options)
	if err != nil {
		return "", err
	}
	res, err := c.svc.GenerateChatText(ctx, req)
	if err != nil {
		return "", grpcx.ToAppError(err)
	}
	return res.GetResult(), nil
}

func (c *AIOrchestratorClient) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string, options aidto.OrchestratorOptions) (string, error) {
	req, err := aitransport.RecommendationTextRequestToProto(systemPrompt, userPrompt, options)
	if err != nil {
		return "", err
	}
	res, err := c.svc.GenerateRecommendationText(ctx, req)
	if err != nil {
		return "", grpcx.ToAppError(err)
	}
	return res.GetResult(), nil
}

// GenerateChatTextStream bridges the server-streaming RPC to the contract's
// channel pair. The goroutine exits when the stream ends, fails, or the
// context is cancelled; errc is buffered so a departed caller never blocks it.
func (c *AIOrchestratorClient) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string, options aidto.OrchestratorOptions) (<-chan string, <-chan error) {
	deltas := make(chan string)
	errc := make(chan error, 1)
	go func() {
		defer close(deltas)
		defer close(errc)

		req, err := aitransport.ChatTextRequestToProto(systemPrompt, userPrompt, options)
		if err != nil {
			errc <- err
			return
		}
		stream, err := c.svc.GenerateChatTextStream(ctx, req)
		if err != nil {
			errc <- grpcx.ToAppError(err)
			return
		}
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				errc <- grpcx.ToAppError(err)
				return
			}
			if delta := chunk.GetDelta(); delta != "" {
				select {
				case deltas <- delta:
				case <-ctx.Done():
					return
				}
			}
			if chunk.GetDone() {
				return
			}
		}
	}()
	return deltas, errc
}
