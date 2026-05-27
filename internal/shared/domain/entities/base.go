package entities

import (
	"time"

	"github.com/google/uuid"
)

type BaseEntity struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type AuditableEntity struct {
	BaseEntity
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type SoftDeleteEntity struct {
	IsDeleted bool `gorm:"type:boolean;not null;default:false"`
}

func (e *SoftDeleteEntity) SoftDelete() {
	e.IsDeleted = true
}

func (e *SoftDeleteEntity) Undo() {
	e.IsDeleted = false
}
