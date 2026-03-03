package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	v *validator.Validate
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func New() *Validator {
	v := validator.New(validator.WithRequiredStructFields())

	// Register tag name function to use json tags in error messages
	v.RegisterTagNameFunc(func(fld interface{ Tag(string) string }) string {
		name := strings.SplitN(fld.Tag("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &Validator{v: v}
}

func (val *Validator) Validate(s any) []FieldError {
	err := val.v.Struct(s)
	if err == nil {
		return nil
	}

	var errs []FieldError
	for _, e := range err.(validator.ValidationErrors) {
		errs = append(errs, FieldError{
			Field:   e.Field(),
			Message: fieldMessage(e),
		})
	}
	return errs
}

func fieldMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", e.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", e.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", e.Field(), e.Param())
	case "eqfield":
		return fmt.Sprintf("%s must match %s", e.Field(), e.Param())
	default:
		return fmt.Sprintf("%s is invalid", e.Field())
	}
}
