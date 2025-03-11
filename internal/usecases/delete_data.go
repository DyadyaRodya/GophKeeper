package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

type (
	// ClientDeleteDataUsecase usecase for handling data deletion of client side
	ClientDeleteDataUsecase struct {
		gateway     interfaces.SaveDataGateway
		dataStorage interfaces.ClientStorage
	}
)

// NewClientDeleteDataUsecase constructor for ClientDeleteDataUsecase
func NewClientDeleteDataUsecase(
	gateway interfaces.SaveDataGateway,
	dataStorage interfaces.ClientStorage,
) *ClientDeleteDataUsecase {
	return &ClientDeleteDataUsecase{
		gateway:     gateway,
		dataStorage: dataStorage,
	}
}

// Handle deletes data from local storage and if online sends update command to the server. Returns metadata
func (u *ClientDeleteDataUsecase) Handle(
	ctx context.Context,
	userUUID string,
	dataUUID string,
) error {
	meta := &domainmodels.DataInfo{
		UUID:        dataUUID,
		OwnerUUID:   userUUID,
		LastUpdated: time.Now().UTC(),
		IsDeleted:   true,
	}
	meta, err := u.gateway.SaveData(ctx, meta, nil) // try notify server to delete
	offline := err != nil && errors.Is(err, domainmodels.ErrOffline)
	if err != nil &&
		!offline &&
		!errors.Is(err, domainmodels.ErrDataDeleted) &&
		!errors.Is(err, domainmodels.ErrDataInfoNotFound) {
		return fmt.Errorf("ClientDeleteDataUsecase.gateway.SaveData: %w", err)
	}

	if offline {
		err = u.dataStorage.SaveData(ctx, meta, []byte{}) // save meta to notify server later
		if err != nil {
			return fmt.Errorf("ClientDeleteDataUsecase.dataStorage.SaveData: %w", err)
		}
	} else {
		err = u.dataStorage.DeleteData(ctx, meta)
		if err != nil {
			return fmt.Errorf("ClientDeleteDataUsecase.dataStorage.DeleteData: %w", err)
		}
	}
	return nil
}
