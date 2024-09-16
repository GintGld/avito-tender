package valid

import (
	"fmt"
)

// Validate validates string for boundary conditions.
func Validate(text, name string, limit int) error {
	if text == "" {
		return fmt.Errorf("%s must not be empty", name)
	}
	if len(text) > limit {
		return fmt.Errorf("%s must not be longer than %d", name, limit)
	}

	return nil
}
