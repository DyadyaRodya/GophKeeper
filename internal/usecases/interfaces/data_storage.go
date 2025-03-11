package interfaces

import (
	"context"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

type (
	ClientStorage interface {
		SaveData(ctx context.Context, meta *domainmodels.DataInfo, data []byte) error
		ReadDataWithMeta(ctx context.Context, userUUID, dataUUID string) (*domainmodels.DataInfo, []byte, error)
		ListData(ctx context.Context, userUUID string) ([]*domainmodels.DataInfo, error)
		DeleteData(ctx context.Context, meta *domainmodels.DataInfo) error
	}
	ServerStorage interface {
		SaveData(ctx context.Context, meta *domainmodels.DataInfo, data []byte) error
		ReadData(ctx context.Context, userUUID, dataUUID string) ([]byte, error)
		DeleteData(ctx context.Context, meta *domainmodels.DataInfo) error
	}
)
