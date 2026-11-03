package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const (
	brandSlugMinLen = 2
	brandSlugMaxLen = 100
)

var brandSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func IsBrandSlugFormatValid(value string) bool {
	if len(value) < brandSlugMinLen || len(value) > brandSlugMaxLen {
		return false
	}
	if !brandSlugPattern.MatchString(value) {
		return false
	}
	if _, err := uuid.Parse(value); err == nil {
		return false
	}
	return true
}

func IsBrandSlug(fl validator.FieldLevel) bool {
	return IsBrandSlugFormatValid(fl.Field().String())
}
