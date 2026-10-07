// Package shared holds Value Objects used by more than one bounded context.
package shared

import (
	"errors"
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]{2,}$`)

// ErrInvalidEmail is returned when a string does not match a valid email format.
var ErrInvalidEmail = errors.New("invalid email format")

// Email is an immutable Value Object — see library-docs/02-domain/entities-and-rules.md.
type Email struct {
	value string
}

// NewEmail validates and normalizes (lowercase, trimmed) a raw email string.
func NewEmail(raw string) (Email, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if !emailPattern.MatchString(normalized) {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: normalized}, nil
}

func (e Email) String() string { return e.value }

func (e Email) Equals(other Email) bool { return e.value == other.value }
