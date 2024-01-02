package validate

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New(validator.WithRequiredStructEnabled())
}

func Struct(name string, value interface{}) error {
	err := validate.Struct(value)
	if err == nil {
		return nil
	}

	filterErrors := make([]error, 0)
	for _, err := range err.(validator.ValidationErrors) {
		filterError := &StructValidationError{
			Filter:   name,
			Field:    err.Field(),
			Tag:      err.ActualTag(),
			Value:    err.Value(),
			Expected: err.Param(),
		}
		filterErrors = append(filterErrors, filterError)
	}

	return errors.Join(filterErrors...)
}
