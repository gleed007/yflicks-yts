package yts

import (
	"errors"
	"testing"
)

type TestEmployee struct {
	Name   string `json:"name"   validate:"required"`
	Salary int    `json:"salary" validate:"min=1,max=5000"`
}

func TestValidateStruct(t *testing.T) {
	t.Run("returns nil if all struct fields pass stipulated validations", func(t *testing.T) {
		employee := TestEmployee{"ytswatcher", 5000}
		received := validateStruct("TestEmployee", &employee)
		if received != nil {
			t.Errorf(`received %v, but expected "%s"`, nil, received)
		}
	})

	t.Run("returns joined StructValidation errors if field validations fail", func(t *testing.T) {
		valErrors := []error{
			&StructValidationError{
				Struct:   "TestEmployee",
				Field:    "Name",
				Tag:      "required",
				Value:    "",
				Expected: "",
			},
			&StructValidationError{
				Struct:   "TestEmployee",
				Field:    "Salary",
				Tag:      "min",
				Value:    0,
				Expected: "1",
			},
		}

		expected := errors.Join(valErrors...)
		received := validateStruct("TestEmployee", &TestEmployee{})
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})
}
