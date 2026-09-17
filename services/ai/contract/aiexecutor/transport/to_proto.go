package transport

import (
	"encoding/json"

	aiv1 "smart-wardrobe-be/contracts/ai/v1"
	aidto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// OptionsFromProto converts wire orchestrator options into the contract input.
func OptionsFromProto(pb *aiv1.OrchestratorOptions) (aidto.OrchestratorOptions, error) {
	var out aidto.OrchestratorOptions
	if pb == nil {
		return out, nil
	}
	out.MaxOutputTokens = int(pb.GetMaxOutputTokens())
	out.Temperature = pb.GetTemperature()
	out.ResponseMIMEType = pb.GetResponseMimeType()

	if raw := pb.GetResponseSchemaJson(); raw != "" {
		var schema any
		if err := json.Unmarshal([]byte(raw), &schema); err != nil {
			return out, apperror.NewBadRequest("Response schema (JSON) không hợp lệ.")
		}
		out.ResponseSchema = schema
	}
	if raw := pb.GetUserId(); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return out, apperror.From(err)
		}
		out.UserID = id
	}
	out.Operation = pb.GetOperation()
	return out, nil
}

// StatsFromDTO converts the contract DTO into its wire form.
func StatsFromDTO(d *aidto.AIUsageAdminStatsDTO) *aiv1.AIUsageAdminStats {
	if d == nil {
		return nil
	}
	return &aiv1.AIUsageAdminStats{
		SnapshotDate:         timestamppb.New(d.SnapshotDate),
		AiRequestsCount:      int32(d.AIRequestsCount),
		AiErrorCount:         int32(d.AIErrorCount),
		AiPaidTokens:         d.AIPaidTokens,
		AiActualCostMicroVnd: d.AIActualCostMicroVND,
	}
}

// EventsFromDTOs converts drill-down rows into wire messages.
func EventsFromDTOs(ds []aidto.AIEventDrillDownDTO) []*aiv1.AIEventDrillDown {
	out := make([]*aiv1.AIEventDrillDown, 0, len(ds))
	for _, d := range ds {
		out = append(out, &aiv1.AIEventDrillDown{
			RequestId:    d.RequestID,
			Provider:     d.Provider,
			Model:        d.Model,
			StatusCode:   int32(d.StatusCode),
			ErrorMessage: d.ErrorMessage,
			CreatedAt:    timestamppb.New(d.CreatedAt),
		})
	}
	return out
}

// UsageFromDTOs converts usage rows into wire messages.
func UsageFromDTOs(ds []aidto.AIUsageByOperationDTO) []*aiv1.AIUsageByOperation {
	out := make([]*aiv1.AIUsageByOperation, 0, len(ds))
	for _, d := range ds {
		out = append(out, &aiv1.AIUsageByOperation{
			Operation:         d.Operation,
			LogicalRoute:      d.LogicalRoute,
			RequestCount:      d.RequestCount,
			TotalCostMicroVnd: d.TotalCostMicroVND,
			TotalPromptTokens: d.TotalPromptTokens,
			TotalOutputTokens: d.TotalOutputTokens,
		})
	}
	return out
}

// ErrorBreakdownFromDTOs converts error rows into wire messages.
func ErrorBreakdownFromDTOs(ds []aidto.AIErrorBreakdownDTO) []*aiv1.AIErrorBreakdown {
	out := make([]*aiv1.AIErrorBreakdown, 0, len(ds))
	for _, d := range ds {
		out = append(out, &aiv1.AIErrorBreakdown{
			ErrorCode:  d.ErrorCode,
			Provider:   d.Provider,
			Model:      d.Model,
			ErrorCount: d.ErrorCount,
		})
	}
	return out
}
