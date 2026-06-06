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

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("username", IsUserName)
		v.RegisterValidation("neqfield", NotEqualField)
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
			"Du lieu khong dung dinh dang",
			"Gia tri so qua lon hoac kieu du lieu khong hop le",
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
			"Du lieu khong dung dinh dang",
			"Gia tri so qua lon hoac kieu du lieu khong hop le",
		)
	}

	return apperror.NewBadRequest(err.Error())
}

func validationMessage(fe validator.FieldError, fieldName, paramName string) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("Vui lòng nhập %s", fieldName)
	case "email":
		return "Định dạng email không hợp lệ"
	case "username":
		return "Username phải dài 4-20 kí tự, chỉ chứa chữ, số, dấu chấm (.), và dấu gạch dưới (_)"
	case "eqfield":
		return fmt.Sprintf("%s phải giống với %s", fieldName, paramName)
	case "nefield":
		return fmt.Sprintf("%s phải khác với %s", fieldName, paramName)
	case "len":
		return fmt.Sprintf("%s phải dài %s kí tự", fieldName, paramName)
	case "gt":
		return fmt.Sprintf("%s phải lớn hơn %s", fieldName, paramName)
	case "min":
		return fmt.Sprintf("%s phải dài ít nhất %s kí tự", fieldName, paramName)
	case "max":
		if fe.Kind().String() == "int32" || fe.Kind().String() == "int" {
			return fmt.Sprintf("%s không được lớn hơn %s", fieldName, paramName)
		}
		return fmt.Sprintf("%s không được vượt quá %s kí tự", fieldName, paramName)
	case "datetime":
		return fmt.Sprintf("%s phải đúng định dạng ngày %s", fieldName, paramName)
	case "oneof":
		return fmt.Sprintf("%s phải là một trong các giá trị: %s", fieldName, paramName)
	case "numeric":
		return fmt.Sprintf("%s chỉ được chứa các ký tự số", fieldName)
	default:
		return fmt.Sprintf("%s: %s không hợp lệ", fieldName, fe.Tag())
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
