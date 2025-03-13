package services

import (
	"errors"
	"testing"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

func TestUsernameService(t *testing.T) {
	cfg := &UsernameConfig{
		MinLen: 5,
		MaxLen: 20,
		AllowedChars: []rune{
			'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
			'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
			'0', '1', '2', '3', '4', '5', '7', '8', '9', '_',
		},
	}
	service := NewUsernameDomainService(cfg)

	type testCase struct {
		name     string
		username string
		expected error
	}
	tests := []testCase{
		{
			name:     "ok",
			username: "johndoe",
			expected: nil,
		},
		{
			name:     "long",
			username: "123456789012345678901234567890",
			expected: domainmodels.ErrLoginTooLong,
		},
		{
			name:     "short",
			username: "123",
			expected: domainmodels.ErrLoginTooShort,
		},
		{
			name:     "bad_chars",
			username: "#$!test'",
			expected: domainmodels.ErrLoginChars,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := service.Validate(tc.username)
			if !errors.Is(err, tc.expected) {
				t.Errorf("expected error %v, got %v", tc.expected, err)
			}
		})
	}
}
