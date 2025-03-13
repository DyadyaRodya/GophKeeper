package interfaces

import (
	"context"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/dto"
)

type (
	RegisterServerGateway interface {
		Register(ctx context.Context, username, password string) (*dto.Keys, error)
	}
	LoginServerGateway interface {
		Login(ctx context.Context, username, password string) (*domainmodels.ShortKeyInfo, error)
	}
	RecoverServerGateway interface {
		Recover(ctx context.Context, username string, recoveryKey []byte, newPassword string) (*dto.Keys, error)
	}
	UpdatePasswordServerGateway interface {
		UpdatePassword(ctx context.Context, oldPassword, newPassword string) (*dto.Keys, error)
	}

	SaveDataGateway interface {
		SaveData(ctx context.Context, meta *domainmodels.DataInfo, encryptedData []byte) (*domainmodels.DataInfo, error)
	}
	ReadDataGateway interface {
		ReadData(ctx context.Context, meta *domainmodels.DataInfo) ([]byte, error)
	}
	ListDataGateway interface {
		ListData(ctx context.Context) ([]*domainmodels.DataInfo, error)
	}
)
