package helper

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New()
	// Key error memakai nama field JSON, bukan nama field struct Go.
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name := strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	return v
}

// ValidateStruct mengembalikan nil bila valid, atau map nama field JSON ke pesan.
func ValidateStruct(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return map[string]string{"body": "data tidak valid"}
	}

	fields := make(map[string]string, len(verrs))
	for _, fe := range verrs {
		fields[fe.Field()] = validationMessage(fe)
	}
	return fields
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		if fe.Kind() == reflect.String {
			return "minimal " + fe.Param() + " karakter"
		}
		return "minimal " + fe.Param()
	case "max":
		if fe.Kind() == reflect.String {
			return "maksimal " + fe.Param() + " karakter"
		}
		return "maksimal " + fe.Param()
	case "oneof":
		return "harus salah satu dari: " + fe.Param()
	default:
		return "tidak valid"
	}
}
