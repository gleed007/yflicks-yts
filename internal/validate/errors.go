package validate

import (
	"fmt"
)

type StructValidationError struct {
	Filter   string
	Field    string
	Tag      string
	Value    interface{}
	Expected string
}

func (e *StructValidationError) Error() string {
	return fmt.Sprintf(
		`failed validation for "%s.%s" field's "%s:%s" tag, provided value %v`,
		e.Filter,
		e.Field,
		e.Tag,
		e.Expected,
		e.Value,
	)
}
