// Package transport maps AI wire types (proto) to and from the owner's
// contract DTOs, so both the gRPC server and remote clients reuse one mapping
// (design-pattern.md "Cross-service transport mapping (A')").
package transport

import (
	"encoding/json"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	aidto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/shared/application/constants/apperror"
)

// OptionsToProto converts the orchestrator options into its wire form. The
// response schema crosses as a JSON string.
func OptionsToProto(opts aidto.OrchestratorOptions) (*aiv1.OrchestratorOptions, error) {
	schema := ""
	if opts.ResponseSchema != nil {
		raw, err := json.Marshal(opts.ResponseSchema)
		if err != nil {
			wrapErr := apperror.NewInternalError(apperror.GenericInternalMessage)
			wrapErr.Cause = err
			return nil, wrapErr
		}
		schema = string(raw)
	}
	return &aiv1.OrchestratorOptions{
		MaxOutputTokens:    int32(opts.MaxOutputTokens),
		Temperature:        opts.Temperature,
		ResponseMimeType:   opts.ResponseMIMEType,
		ResponseSchemaJson: schema,
		UserId:             opts.UserID.String(),
		Operation:          opts.Operation,
	}, nil
}

// ChatTextRequestToProto builds the wire chat request from prompts and options.
func ChatTextRequestToProto(systemPrompt, userPrompt string, opts aidto.OrchestratorOptions) (*aiv1.GenerateChatTextRequest, error) {
	options, err := OptionsToProto(opts)
	if err != nil {
		return nil, err
	}
	return &aiv1.GenerateChatTextRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Options:      options,
	}, nil
}

// RecommendationTextRequestToProto builds the wire recommendation request.
func RecommendationTextRequestToProto(systemPrompt, userPrompt string, opts aidto.OrchestratorOptions) (*aiv1.GenerateRecommendationTextRequest, error) {
	options, err := OptionsToProto(opts)
	if err != nil {
		return nil, err
	}
	return &aiv1.GenerateRecommendationTextRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Options:      options,
	}, nil
}

// EmbeddingsFromProto converts wire embedding vectors into matrices.
func EmbeddingsFromProto(pb []*aiv1.EmbeddingVector) [][]float32 {
	out := make([][]float32, 0, len(pb))
	for _, item := range pb {
		out = append(out, item.GetValues())
	}
	return out
}

// AdminStatsToDTO converts wire AI usage statistics into the contract DTO.
func AdminStatsToDTO(pb *aiv1.AIUsageAdminStats) *aidto.AIUsageAdminStatsDTO {
	if pb == nil {
		return nil
	}
	out := &aidto.AIUsageAdminStatsDTO{
		AIRequestsCount:      int(pb.GetAiRequestsCount()),
		AIErrorCount:         int(pb.GetAiErrorCount()),
		AIPaidTokens:         pb.GetAiPaidTokens(),
		AIActualCostMicroVND: pb.GetAiActualCostMicroVnd(),
	}
	if ts := pb.GetSnapshotDate(); ts != nil {
		out.SnapshotDate = ts.AsTime()
	}
	return out
}

// EventsToDTOs converts the wire drill-down list into contract DTOs plus total.
func EventsToDTOs(pb *aiv1.AIEventDrillDownList) ([]aidto.AIEventDrillDownDTO, int64) {
	if pb == nil {
		return nil, 0
	}
	out := make([]aidto.AIEventDrillDownDTO, 0, len(pb.GetItems()))
	for _, item := range pb.GetItems() {
		entry := aidto.AIEventDrillDownDTO{
			RequestID:    item.GetRequestId(),
			Provider:     item.GetProvider(),
			Model:        item.GetModel(),
			StatusCode:   int(item.GetStatusCode()),
			ErrorMessage: item.GetErrorMessage(),
		}
		if ts := item.GetCreatedAt(); ts != nil {
			entry.CreatedAt = ts.AsTime()
		}
		out = append(out, entry)
	}
	return out, pb.GetTotal()
}

// UsageByOperationToDTOs converts the wire usage list into contract DTOs.
func UsageByOperationToDTOs(pb *aiv1.AIUsageByOperationList) []aidto.AIUsageByOperationDTO {
	if pb == nil {
		return nil
	}
	out := make([]aidto.AIUsageByOperationDTO, 0, len(pb.GetItems()))
	for _, item := range pb.GetItems() {
		out = append(out, aidto.AIUsageByOperationDTO{
			Operation:         item.GetOperation(),
			LogicalRoute:      item.GetLogicalRoute(),
			RequestCount:      item.GetRequestCount(),
			TotalCostMicroVND: item.GetTotalCostMicroVnd(),
			TotalPromptTokens: item.GetTotalPromptTokens(),
			TotalOutputTokens: item.GetTotalOutputTokens(),
		})
	}
	return out
}

// ErrorBreakdownToDTOs converts the wire error list into contract DTOs.
func ErrorBreakdownToDTOs(pb *aiv1.AIErrorBreakdownList) []aidto.AIErrorBreakdownDTO {
	if pb == nil {
		return nil
	}
	out := make([]aidto.AIErrorBreakdownDTO, 0, len(pb.GetItems()))
	for _, item := range pb.GetItems() {
		out = append(out, aidto.AIErrorBreakdownDTO{
			ErrorCode:  item.GetErrorCode(),
			Provider:   item.GetProvider(),
			Model:      item.GetModel(),
			ErrorCount: item.GetErrorCount(),
		})
	}
	return out
}
