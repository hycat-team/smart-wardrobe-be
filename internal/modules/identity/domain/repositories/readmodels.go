package repositories

import (
	"time"

	"smart-wardrobe-be/internal/modules/identity/domain/entities"

	"github.com/google/uuid"
)

// Repository read models owned by the domain layer (feature 017);
// application/dto aliases them for handler compatibility.
type UserStyleProfileRes struct {
	UserID          uuid.UUID                 `json:"userId"`
	TasteEmbedding  entities.Vector           `json:"tasteEmbedding,omitempty"`
	PreferredColors *entities.PreferredColors `json:"preferredColors,omitempty"`
}

type UserAdminStatsDTO struct {
	SnapshotDate   time.Time `json:"snapshotDate"`
	TotalUsers     int       `json:"totalUsers"`
	NewUsers       int       `json:"newUsers"`
	ActiveUsersDau int       `json:"activeUsersDau"`
}
