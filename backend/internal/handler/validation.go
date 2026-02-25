package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// formatValidationError converts Gin/validator errors into human-readable messages
// using JSON field names instead of Go struct field names.
func formatValidationError(err error) string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err.Error()
	}

	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		field := jsonFieldName(fe.Namespace())
		switch fe.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("%s is required", field))
		case "min":
			msgs = append(msgs, fmt.Sprintf("%s must be at least %s characters", field, fe.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("%s must be at most %s characters", field, fe.Param()))
		case "email":
			msgs = append(msgs, fmt.Sprintf("%s must be a valid email", field))
		case "uuid":
			msgs = append(msgs, fmt.Sprintf("%s must be a valid UUID", field))
		case "oneof":
			msgs = append(msgs, fmt.Sprintf("%s must be one of: %s", field, fe.Param()))
		default:
			msgs = append(msgs, fmt.Sprintf("%s failed validation (%s)", field, fe.Tag()))
		}
	}
	return strings.Join(msgs, "; ")
}

// jsonFieldName extracts the last segment of the namespace and converts to
// a readable form. e.g. "ReinstallRequest.OSProfileID" → "os_profile_id"
// falls back to the raw field name for unresolvable cases.
func jsonFieldName(namespace string) string {
	// Namespace format: "StructName.FieldName" or "StructName.Nested.FieldName"
	parts := strings.Split(namespace, ".")
	if len(parts) < 2 {
		return namespace
	}
	// Use the last part (Go field name) and convert CamelCase → snake_case
	return camelToSnake(parts[len(parts)-1])
}

func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + ('a' - 'A'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
