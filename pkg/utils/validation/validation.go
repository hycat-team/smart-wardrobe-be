package validation

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"strings"

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
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

func TranslateValidationError(err error) *apperror.Error {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			var msg string
			fieldName := fe.Field()
			paramName := fe.Param()

			switch fe.Tag() {
			case "required":
				msg = fmt.Sprintf("Vui lòng nhập %s", fieldName)
			case "email":
				msg = "Định dạng email không hợp lệ"
			case "username":
				msg = "Username phải dài 4-20 kí tự, chỉ chứa chữ, số, dấu chấm (.), và dấu gạch dưới (_)"
			case "eqfield":
				msg = fmt.Sprintf("%s phải giống với %s", fieldName, paramName)
			case "nefield":
				msg = fmt.Sprintf("%s phải khác với %s", fieldName, paramName)
			case "len":
				msg = fmt.Sprintf("%s phải dài %s kí tự", fieldName, paramName)
			case "gt":
				msg = fmt.Sprintf("%s phải lớn hơn %s", fieldName, paramName)
			case "min":
				msg = fmt.Sprintf("%s phải dài ít nhất %s kí tự", fieldName, paramName)
			case "max":
				if fe.Kind().String() == "int32" || fe.Kind().String() == "int" {
					msg = fmt.Sprintf("%s không được lớn hơn %s", fieldName, paramName)
				} else {
					msg = fmt.Sprintf("%s không được vượt quá %s kí tự", fieldName, paramName)
				}
			case "datetime":
				msg = fmt.Sprintf("%s phải đúng định dạng ngày %s", fieldName, paramName)
			case "oneof":
				msg = fmt.Sprintf("%s phải là một trong các giá trị: %s", fieldName, paramName)
			case "numeric":
				msg = fmt.Sprintf("%s chỉ được chứa các ký tự số", fieldName)
			default:
				msg = fmt.Sprintf("%s: %s không hợp lệ", fieldName, fe.Tag())
			}

			msgs = append(msgs, msg)
		}

		return apperror.NewBadRequest(strings.Join(msgs, "; "))
	}

	if strings.Contains(err.Error(), "unmarshal") {
		return apperror.NewError(
			http.StatusBadRequest,
			"Dữ liệu không đúng định dạng",
			"Giá trị số quá lớn hoặc kiểu dữ liệu không hợp lệ",
		)
	}

	return apperror.NewBadRequest(err.Error())
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
