package interfaces

import (
	"context"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

type (
	RepositorySession interface {

		//users

		AddUser(ctx context.Context, user *domainmodels.User) error
		UpdateUser(ctx context.Context, user *domainmodels.User) error
		GetUserByUsername(ctx context.Context, username string) (*domainmodels.User, error)
		GetUserByUUID(ctx context.Context, uuid string) (*domainmodels.User, error)

		// keys

		SaveKeysInfo(ctx context.Context, userUUID string, keysInfo *domainmodels.KeysInfo) error
		GetKeysInfo(ctx context.Context, userUUID string) (*domainmodels.KeysInfo, error)

		// data

		SaveDataInfo(ctx context.Context, dataInfo *domainmodels.DataInfo) error
		ReadDataInfo(ctx context.Context, userUUID string, dataUUID string) (*domainmodels.DataInfo, error)
		ReadDataInfoByUserUUID(ctx context.Context, userUUID string) ([]*domainmodels.DataInfo, error)

		// common

		Commit(ctx context.Context) error
		Close(ctx context.Context) error
	}
	Repository interface {
		NewSession(ctx context.Context) (RepositorySession, error)
	}
)
