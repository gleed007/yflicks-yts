package yts

import (
	"fmt"
)

type FilterValidationError struct {
	filter   string
	field    string
	tag      string
	value    interface{}
	expected string
}

func (e *FilterValidationError) Error() string {
	return fmt.Sprintf(
		`failed validation for "%s.%s" field's "%s:%s" tag, provided value %v`,
		e.filter,
		e.field,
		e.tag,
		e.expected,
		e.value,
	)
}
