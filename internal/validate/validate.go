package validate

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type StructValidationError struct {
	Struct   string
	Field    string
	Tag      string
	Value    interface{}
	Expected string
}

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

func (e *StructValidationError) Error() string {
	return fmt.Sprintf(
		`failed validation for "%s.%s" field's "%s:%s" tag, provided value %v`,
		e.Struct,
		e.Field,
		e.Tag,
		e.Expected,
		e.Value,
	)
}

func Struct(name string, value interface{}) error {
	err := validate.Struct(value)
	if err == nil {
		return nil
	}

	valErrors := make([]error, 0)
	for _, err := range err.(validator.ValidationErrors) {
		valError := &StructValidationError{
			Struct:   name,
			Field:    err.Field(),
			Tag:      err.ActualTag(),
			Value:    err.Value(),
			Expected: err.Param(),
		}
		valErrors = append(valErrors, valError)
	}

	return errors.Join(valErrors...)
}
