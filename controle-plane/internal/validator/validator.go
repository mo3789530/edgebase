package validator

import (
	"fmt"
	"regexp"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
}

type Validator struct {
	errors []ValidationError
}

func New() *Validator {
	return &Validator{errors: []ValidationError{}}
}

func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "is required",
		})
	}
	return v
}

func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters", min),
		})
	}
	return v
}

func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters", max),
		})
	}
	return v
}

func (v *Validator) Pattern(field, value, pattern string) *Validator {
	if value != "" {
		matched, _ := regexp.MatchString(pattern, value)
		if !matched {
			v.errors = append(v.errors, ValidationError{
				Field:   field,
				Message: "has invalid format",
			})
		}
	}
	return v
}

func (v *Validator) IsValid() bool {
	return len(v.errors) == 0
}

func (v *Validator) Errors() []ValidationError {
	return v.errors
}

func (v *Validator) ErrorMap() map[string]string {
	errMap := make(map[string]string)
	for _, err := range v.errors {
		errMap[err.Field] = err.Message
	}
	return errMap
}
