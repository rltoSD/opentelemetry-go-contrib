package cortex

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "replace character",
			input: "test/key-1",
			want:  "test_key_1",
		},
		{
			name:  "add prefix if starting with digit",
			input: "0123456789",
			want:  "key_0123456789",
		},
		{
			name:  "add prefix if starting with _",
			input: "_0123456789",
			want:  "key_0123456789",
		},
		{
			name:  "starts with _ after sanitization",
			input: "/0123456789",
			want:  "key_0123456789",
		},
		{
			name:  "valid input",
			input: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_0123456789",
			want:  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := sanitize(tt.input), tt.want; got != want {
				t.Errorf("Sanitize() = %q; want %q", got, want)
			}
		})
	}
}
