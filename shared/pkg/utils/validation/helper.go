package validation

import (
	"reflect"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	"golang.org/x/text/unicode/norm"
)

func IsUserName(fl validator.FieldLevel) bool {
	s := fl.Field().String()

	reAllowed := regexp.MustCompile(`^[a-zA-Z0-9._]{4,20}$`)
	if !reAllowed.MatchString(s) {
		return false
	}

	reRepeat := regexp.MustCompile(`[_.]{2}`)
	if reRepeat.MatchString(s) {
		return false
	}

	if s[0] == '.' || s[0] == '_' || s[len(s)-1] == '.' || s[len(s)-1] == '_' {
		return false
	}

	return true
}

func NotEqualField(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	other := fl.Parent().FieldByName(fl.Param()).String()
	return field != other
}

func IsPasswordComplex(fl validator.FieldLevel) bool {
	s := fl.Field().String()
	if len(s) < 8 {
		return false
	}
	var (
		hasUpper   = false
		hasLower   = false
		hasDigit   = false
		hasSpecial = false
	)
	for _, char := range s {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func NFCMax(fl validator.FieldLevel) bool {
	limit, err := strconv.Atoi(fl.Param())
	if err != nil || limit < 0 {
		return false
	}
	return utf8.RuneCountInString(norm.NFC.String(fl.Field().String())) <= limit
}

func normalizeNFC(value reflect.Value) {
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		normalizeNFC(value.Elem())
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			normalizeNFC(value.Field(i))
		}
	case reflect.String:
		if value.CanSet() {
			value.SetString(norm.NFC.String(value.String()))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			normalizeNFC(value.Index(i))
		}
	}
}
