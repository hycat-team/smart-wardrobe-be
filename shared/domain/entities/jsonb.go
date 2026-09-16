package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type bodyProfile struct {
	HeightCM       float64              `json:"height_cm"`
	WeightKG       float64              `json:"weight_kg"`
	BodyShape      string               `json:"body_shape"`
	Measurements   *bodyMeasurements    `json:"measurements,omitempty"`
	InferredByAI   *inferredBodyProfile `json:"inferred_by_ai,omitempty"`
	VerifiedByUser bool                 `json:"verified_by_user"`
	LastUpdatedAt  *time.Time           `json:"last_updated_at,omitempty"`
}

func (b bodyProfile) Value() (driver.Value, error) {
	return json.Marshal(b)
}

func (b *bodyProfile) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, b)
}

func (b *bodyProfile) UnmarshalJSON(data []byte) error {
	type rawBodyProfile struct {
		HeightCM            *float64             `json:"height_cm"`
		WeightKG            *float64             `json:"weight_kg"`
		BodyShape           *string              `json:"body_shape"`
		Measurements        *bodyMeasurements    `json:"measurements"`
		InferredByAI        *inferredBodyProfile `json:"inferred_by_ai"`
		VerifiedByUser      *bool                `json:"verified_by_user"`
		LastUpdatedAt       *time.Time           `json:"last_updated_at"`
		LegacyHeight        *float64             `json:"height"`
		LegacyWeight        *float64             `json:"weight"`
		LegacyBodyType      *string              `json:"bodyType"`
		LegacyEstimated     *string              `json:"estimatedBodyShape"`
		LegacyFitPreference *string              `json:"fitPreference"`
		LegacySkinTone      *string              `json:"skinTone"`
		LegacyRecommended   *string              `json:"recommendedSize"`
		LegacyStylingNotes  *string              `json:"stylingNotes"`
	}

	var raw rawBodyProfile
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if raw.HeightCM != nil {
		b.HeightCM = *raw.HeightCM
	} else if raw.LegacyHeight != nil {
		b.HeightCM = *raw.LegacyHeight
	}

	if raw.WeightKG != nil {
		b.WeightKG = *raw.WeightKG
	} else if raw.LegacyWeight != nil {
		b.WeightKG = *raw.LegacyWeight
	}

	if raw.BodyShape != nil {
		b.BodyShape = strings.TrimSpace(*raw.BodyShape)
	} else if raw.LegacyBodyType != nil {
		b.BodyShape = strings.TrimSpace(*raw.LegacyBodyType)
	}

	b.Measurements = raw.Measurements
	b.InferredByAI = raw.InferredByAI

	if b.InferredByAI == nil && raw.LegacyEstimated != nil && strings.TrimSpace(*raw.LegacyEstimated) != "" {
		b.InferredByAI = &inferredBodyProfile{
			BodyShape: strings.TrimSpace(*raw.LegacyEstimated),
		}
	}

	if raw.VerifiedByUser != nil {
		b.VerifiedByUser = *raw.VerifiedByUser
	}

	if raw.LastUpdatedAt != nil {
		b.LastUpdatedAt = raw.LastUpdatedAt
	}

	return nil
}

type bodyMeasurements struct {
	ChestCM float64 `json:"chest_cm,omitempty"`
	WaistCM float64 `json:"waist_cm,omitempty"`
	HipCM   float64 `json:"hip_cm,omitempty"`
}

type inferredBodyProfile struct {
	BodyShape       string   `json:"body_shape"`
	ConfidenceScore *float64 `json:"confidence_score,omitempty"`
}

type BodyProfile = bodyProfile
type BodyMeasurements = bodyMeasurements
type InferredBodyProfile = inferredBodyProfile

type JSONDocument json.RawMessage

func (j JSONDocument) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONDocument) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = append((*j)[:0], v...)
	default:
		return fmt.Errorf("unsupported JSON document value %T", value)
	}
	return nil
}

type preferredColors []string

type PreferredColors = preferredColors

func (p preferredColors) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func (p *preferredColors) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, p)
}
