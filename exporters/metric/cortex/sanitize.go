package cortex

import (
	"strings"
	"unicode"
)

// This is a copy of opentelemetry-go/sdk/internal/sanitize.go

// sanitize replaces non-alphanumeric characters with underscores
func sanitize(s string) string {
	if len(s) == 0 {
		return s
	}

	s = strings.Map(sanitizeRune, s)
	if unicode.IsDigit(rune(s[0])) {
		s = "key_" + s
	}
	if s[0] == '_' {
		s = "key" + s
	}
	return s
}

// converts anything that is not a letter or digit to an underscore
func sanitizeRune(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return r
	}
	// Everything else turns into an underscore
	return '_'
}
