package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

const (
	errInvalidFormatTitle  = "Dữ liệu không đúng định dạng"
	errInvalidFormatDetail = "Giá trị số quá lớn hoặc kiểu dữ liệu không hợp lệ"

	errMsgRequired           = "Vui lòng nhập %s"
	errMsgEmail              = "Định dạng email không hợp lệ"
	errMsgUsername           = "Username phải dài 4-20 kí tự, chỉ chứa chữ, số, dấu chấm (.), và dấu gạch dưới (_)"
	errMsgPasswordComplexity = "%s phải dài ít nhất 8 ký tự, bao gồm ít nhất 1 chữ hoa, 1 chữ thường, 1 số và 1 ký tự đặc biệt"
	errMsgEqField            = "%s phải giống với %s"
	errMsgNeField            = "%s phải khác với %s"
	errMsgLen                = "%s phải dài %s kí tự"
	errMsgGt                 = "%s phải lớn hơn %s"
	errMsgMin                = "%s phải dài ít nhất %s kí tự"
	errMsgMaxInt             = "%s không được lớn hơn %s"
	errMsgMaxStr             = "%s không được vượt quá %s kí tự"
	errMsgDateTime           = "%s phải đúng định dạng ngày %s"
	errMsgOneOf              = "%s phải là một trong các giá trị: %s"
	errMsgNumeric            = "%s chỉ được chứa các ký tự số"
	errMsgDefault            = "%s: %s không hợp lệ"
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

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("username", IsUserName)
		v.RegisterValidation("neqfield", NotEqualField)
		v.RegisterValidation("password_complexity", IsPasswordComplex)
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "" {
				return lowerCamelCase(fld.Name)
			}
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

func humanizeFieldName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}

	name = strings.ReplaceAll(name, "_", " ")
	runes := []rune(name)
	var builder strings.Builder
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				builder.WriteRune(' ')
			}
		}
		builder.WriteRune(unicode.ToLower(r))
	}

	return strings.TrimSpace(builder.String())
}

func fieldJSONName(fe validator.FieldError) string {
	if field := strings.TrimSpace(fe.Field()); field != "" {
		return field
	}
	return lowerCamelCase(fe.StructField())
}

func fieldLabel(fe validator.FieldError) string {
	if field := strings.TrimSpace(fe.Field()); field != "" {
		return humanizeFieldName(field)
	}
	return humanizeFieldName(fe.StructField())
}

func lowerCamelCase(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	if strings.Contains(name, "_") {
		parts := strings.Split(name, "_")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
			if parts[i] == "" {
				continue
			}
			if i == 0 {
				parts[i] = strings.ToLower(parts[i][:1]) + parts[i][1:]
				continue
			}
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
		return strings.Join(parts, "")
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func TranslateValidationError(err error) *apperror.Error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		items := make([]apperror.ValidationErrorItem, 0, len(ve))
		for _, fe := range ve {
			fieldName := fieldLabel(fe)
			paramName := fe.Param()
			msg := validationMessage(fe, fieldName, paramName)
			items = append(items, apperror.ValidationErrorItem{
				Field:   fieldJSONName(fe),
				Message: msg,
			})
		}

		appErr := apperror.NewBadRequest("")
		if len(items) > 0 {
			appErr.Message = items[0].Message
			appErr.Errors = items
		}
		return appErr
	}

	var unmarshalTypeErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeErr) {
		appErr := apperror.NewError(
			http.StatusBadRequest,
			errInvalidFormatTitle,
			errInvalidFormatDetail,
		)
		if unmarshalTypeErr.Field != "" {
			appErr.Errors = []apperror.ValidationErrorItem{{
				Field:   lowerCamelCase(unmarshalTypeErr.Field),
				Message: appErr.Message,
			}}
		}
		return appErr
	}

	if strings.Contains(err.Error(), "unmarshal") {
		return apperror.NewError(
			http.StatusBadRequest,
			errInvalidFormatTitle,
			errInvalidFormatDetail,
		)
	}

	return apperror.NewBadRequest(err.Error())
}

func validationMessage(fe validator.FieldError, fieldName, paramName string) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf(errMsgRequired, fieldName)
	case "email":
		return errMsgEmail
	case "username":
		return errMsgUsername
	case "password_complexity":
		return fmt.Sprintf(errMsgPasswordComplexity, fieldName)
	case "eqfield":
		return fmt.Sprintf(errMsgEqField, fieldName, paramName)
	case "nefield":
		return fmt.Sprintf(errMsgNeField, fieldName, paramName)
	case "len":
		return fmt.Sprintf(errMsgLen, fieldName, paramName)
	case "gt":
		return fmt.Sprintf(errMsgGt, fieldName, paramName)
	case "min":
		return fmt.Sprintf(errMsgMin, fieldName, paramName)
	case "max":
		if fe.Kind().String() == "int32" || fe.Kind().String() == "int" {
			return fmt.Sprintf(errMsgMaxInt, fieldName, paramName)
		}
		return fmt.Sprintf(errMsgMaxStr, fieldName, paramName)
	case "datetime":
		return fmt.Sprintf(errMsgDateTime, fieldName, paramName)
	case "oneof":
		return fmt.Sprintf(errMsgOneOf, fieldName, paramName)
	case "numeric":
		return fmt.Sprintf(errMsgNumeric, fieldName)
	default:
		return fmt.Sprintf(errMsgDefault, fieldName, fe.Tag())
	}
}

func BindJSON(c *gin.Context, obj any) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return TranslateValidationError(err)
	}
	return nil
}

func BindQuery(c *gin.Context, obj any) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return TranslateValidationError(err)
	}
	return nil
}
