package mapper

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
)

// MapUserStyleProfileRes converts the stored style profile into the client
// response. A nil profile produces the defaults of a user who has never saved
// one (enabled, empty lists, no embedding) instead of an error. The embedding
// vector, its hash and the embedding model are deliberately not exposed.
func MapUserStyleProfileRes(
	profile *entities.UserStyleProfile,
	personalizationAvailable bool,
	maxStyles int,
) *dto.StyleProfileRes {
	res := &dto.StyleProfileRes{
		Styles:                   []string{},
		ExcludedStyles:           []string{},
		Enabled:                  true,
		PersonalizationAvailable: personalizationAvailable,
		MaxStyles:                maxStyles,
	}
	if profile == nil {
		return res
	}

	if len(profile.Styles) > 0 {
		res.Styles = append(res.Styles, profile.Styles...)
	}
	if len(profile.ExcludedStyles) > 0 {
		res.ExcludedStyles = append(res.ExcludedStyles, profile.ExcludedStyles...)
	}
	res.Enabled = profile.Enabled
	res.HasEmbedding = len(profile.StyleEmbedding) > 0
	if !profile.UpdatedAt.IsZero() {
		updatedAt := profile.UpdatedAt
		res.UpdatedAt = &updatedAt
	}
	return res
}
