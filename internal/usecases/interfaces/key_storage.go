package interfaces

import (
	"context"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

type (
	KeyStorage interface {
		SaveKeys(ctx context.Context, keysInfo *domainmodels.ShortKeyInfo) error
		ReadKeys(ctx context.Context) (*domainmodels.ShortKeyInfo, error)
	}
)
