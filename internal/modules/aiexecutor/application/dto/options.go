package dto

import (
	"github.com/google/uuid"
)

type OrchestratorOptions struct {
	MaxOutputTokens  int
	Temperature      float64
	ResponseMIMEType string
	ResponseSchema   any
	UserID           uuid.UUID
	Operation        string
}
