package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StyleList is a string slice stored as a JSON array in a jsonb column.
// A nil list is persisted as an empty array because both columns that use this
// type are NOT NULL DEFAULT '[]'::jsonb.
type StyleList []string

func (s StyleList) Value() (driver.Value, error) {
	// Return string, not []byte — a []byte driver value is encoded as bytea
	// under the simple protocol and a jsonb column rejects it (see
	// shared/domain/entities/jsonb.go for the same reasoning).
	if s == nil {
		return "[]", nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (s *StyleList) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB style list:", value))
	}
	return json.Unmarshal(data, s)
}

// StyleWeightMap is a style -> weight map stored as a JSON object in a jsonb
// column. A nil map is persisted as an empty object because the column is
// NOT NULL DEFAULT '{}'::jsonb.
type StyleWeightMap map[string]float64

func (m StyleWeightMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

func (m *StyleWeightMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB style weight map:", value))
	}
	return json.Unmarshal(data, m)
}

// UserStyleProfile is the persisted style profile of one user (feature 033).
// Declared preferences (styles/excluded_styles) are never overwritten by the
// learning pipeline; learned_* holds the derived representation instead.
type UserStyleProfile struct {
	UserID         uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	Styles         StyleList `gorm:"type:jsonb;not null;default:'[]'"`
	ExcludedStyles StyleList `gorm:"column:excluded_styles;type:jsonb;not null;default:'[]'"`
	// Enabled deliberately carries NO gorm `default` tag: GORM skips a
	// zero-valued field that declares a default, which would drop
	// enabled=false from the INSERT and let ON CONFLICT fall back to the DDL
	// DEFAULT TRUE. Without the tag the value is always written explicitly;
	// the DDL still keeps DEFAULT TRUE for rows created outside GORM.
	Enabled          bool           `gorm:"column:enabled;type:boolean;not null"`
	StyleEmbedding   Vector         `gorm:"column:style_embedding;type:vector(768)"`
	EmbeddingHash    *string        `gorm:"column:embedding_hash;type:text"`
	EmbeddingModel   *string        `gorm:"column:embedding_model;type:varchar(100)"`
	LearnedStyles    StyleWeightMap `gorm:"column:learned_styles;type:jsonb;not null;default:'{}'"`
	LearnedEmbedding Vector         `gorm:"column:learned_embedding;type:vector(768)"`
	LearnedAt        *time.Time     `gorm:"column:learned_at;type:timestamp with time zone"`
	CreatedAt        time.Time      `gorm:"column:created_at;type:timestamp with time zone;not null;default:now()"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;type:timestamp with time zone;not null;default:now()"`
}
