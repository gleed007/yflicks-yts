package validate

import (
	"errors"
	"testing"
)

type TestEmployee struct {
	Name   string `validate:"required"`
	Salary int    `validate:"min=1,max=5000"`
}

func TestStruct(t *testing.T) {
	t.Run("returns nil if all struct fields pass stipulated validations", func(t *testing.T) {
		employee := TestEmployee{"ytswatcher", 5000}
		received := Struct("TestEmployee", &employee)
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
		received := Struct("TestEmployee", &TestEmployee{})
		if received == nil || received.Error() != expected.Error() {
			t.Errorf(`received %s, but expected "%s"`, received, expected)
		}
	})
}
