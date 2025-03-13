package services

import (
	"slices"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

type (
	// UsernameConfig config for UsernameDomainService checks
	UsernameConfig struct {
		MinLen       int
		MaxLen       int
		AllowedChars []rune
	}
	// UsernameDomainService service for username validation
	UsernameDomainService struct {
		config *UsernameConfig
	}
)

// NewUsernameDomainService constructor for UsernameDomainService
func NewUsernameDomainService(config *UsernameConfig) *UsernameDomainService {
	return &UsernameDomainService{
		config: config,
	}
}

// Validate validates username using config limits
func (l *UsernameDomainService) Validate(username string) error {
	if len(username) < l.config.MinLen {
		return domainmodels.ErrLoginTooShort
	}
	if len(username) > l.config.MaxLen {
		return domainmodels.ErrLoginTooLong
	}

	for _, char := range username {
		if !slices.Contains(l.config.AllowedChars, char) {
			return domainmodels.ErrLoginChars
		}
	}

	return nil
}
