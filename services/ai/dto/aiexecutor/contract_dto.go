package dto

import (
	"github.com/google/uuid"
)

// OrchestratorOptions is the input options type of IAIOrchestrator contract
// methods (XxxInput-style). It is the only structured payload the AI
// orchestrator contract exchanges, so it lives in contract_dto.go per the
// DTO Convention (design-pattern.md).
type OrchestratorOptions struct {
	MaxOutputTokens  int
	Temperature      float64
	ResponseMIMEType string
	ResponseSchema   any
	UserID           uuid.UUID
	Operation        string
}
