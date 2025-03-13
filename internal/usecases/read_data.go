package usecases

import (
	"context"
	"errors"
	"fmt"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

type (
	ClientReadDataGateway interface {
		interfaces.SaveDataGateway
		interfaces.ReadDataGateway
		interfaces.ListDataGateway
	}

	// ServerReadDataUsecase usecase for handling data reading on server side
	ServerReadDataUsecase struct {
		repo        interfaces.Repository
		dataStorage interfaces.ServerStorage
	}
	// ClientReadDataUsecase usecase for handling data reading on client side
	ClientReadDataUsecase struct {
		convertService    interfaces.BytesToRawDataConverter
		encryptionService interfaces.Decryptor
		dataStorage       interfaces.ClientStorage
		gateway           ClientReadDataGateway
	}
)

// NewServerReadDataUsecase constructor for ServerReadDataUsecase
func NewServerReadDataUsecase(
	repo interfaces.Repository,
	dataStorage interfaces.ServerStorage,
) *ServerReadDataUsecase {
	return &ServerReadDataUsecase{
		repo:        repo,
		dataStorage: dataStorage,
	}
}

// Handle reads encrypted data from storage and metadata from DB. Returns metadata for data manipulation
// and encrypted data
func (u *ServerReadDataUsecase) Handle(
	ctx context.Context,
	userUUID string,
	dataUUID string,
) (*domainmodels.DataInfo, []byte, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerReadDataUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	meta, err := dbSess.ReadDataInfo(ctx, userUUID, dataUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerReadDataUsecase.repo.ReadDataInfo: %w", err)
	}

	if meta.IsDeleted {
		return meta, nil, domainmodels.ErrDataDeleted
	}

	encryptedData, err := u.dataStorage.ReadData(ctx, userUUID, dataUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerReadDataUsecase.dataStorage.ReadData: %w", err)
	}
	return meta, encryptedData, nil
}

// NewClientReadDataUsecase constructor for ClientReadDataUsecase
func NewClientReadDataUsecase(
	convertService interfaces.BytesToRawDataConverter,
	encryptionService interfaces.Decryptor,
	dataStorage interfaces.ClientStorage,
	gateway ClientReadDataGateway,
) *ClientReadDataUsecase {
	return &ClientReadDataUsecase{
		convertService:    convertService,
		encryptionService: encryptionService,
		dataStorage:       dataStorage,
		gateway:           gateway,
	}
}

// Handle reads encrypted data from local storage and decrypts it. Returns metadata for data manipulation
// and decrypted data
func (u *ClientReadDataUsecase) Handle(
	ctx context.Context,
	dek []byte,
	userUUID string,
	dataUUID string,
) (*domainmodels.DataInfo, *domainmodels.RawData, error) {
	localMeta, encryptedData, err := u.dataStorage.ReadDataWithMeta(ctx, userUUID, dataUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("ClientReadDataUsecase.dataStorage.ReadData: %w", err)
	}

	remoteMeta, err := u.checkRemote(ctx, localMeta)
	if err != nil {
		return nil, nil, fmt.Errorf("ClientReadDataUsecase.Handle.checkRemote: %w", err)
	}

	var decryptedBytes []byte
	if remoteMeta != localMeta {
		decryptedBytes, err = u.syncWithRemoteAndDecrypt(ctx, dek, remoteMeta, localMeta, encryptedData)
		if err != nil {
			return nil, nil, fmt.Errorf("ClientReadDataUsecase.syncWithRemoteAndDecrypt: %w", err)
		}
	} else {
		decryptedBytes, err = u.encryptionService.Decrypt(encryptedData, dek)
		if err != nil {
			return nil, nil, fmt.Errorf("ClientReadDataUsecase.encryptionService.Decrypt: %w", err)
		}
	}
	rawData, err := u.convertService.ConvertBytesToRawData(decryptedBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("ClientReadDataUsecase.convertService.ConvertBytesToRawData: %w", err)
	}

	return localMeta, rawData, nil
}

func (u *ClientReadDataUsecase) syncWithRemoteAndDecrypt(
	ctx context.Context,
	dek []byte,
	remote *domainmodels.DataInfo,
	local *domainmodels.DataInfo,
	localEncryptedData []byte,
) ([]byte, error) {
	if remote.LastUpdated.After(local.LastUpdated) {
		newEncryptedData, err := u.gateway.ReadData(ctx, remote)
		if err != nil {
			return nil, fmt.Errorf("ClientReadDataUsecase.syncWithRemoteAndDecrypt.gateway.ReadData: %w", err)
		}
		decryptedBytes, err := u.encryptionService.Decrypt(newEncryptedData, dek) // decrypt before saving to check DEK ok
		if err != nil {
			return nil, fmt.Errorf("ClientReadDataUsecase.syncWithRemoteAndDecrypt.encryptionService.Decrypt: %w", err)
		}

		err = u.dataStorage.SaveData(ctx, remote, newEncryptedData)
		if err != nil {
			return nil, fmt.Errorf("ClientReadDataUsecase.syncWithRemoteAndDecrypt.dataStorage.SaveData: %w", err)
		}
		return decryptedBytes, nil
	} else if local.IsDeleted {
		return nil, domainmodels.ErrDataDeleted
	}
	_, err := u.gateway.SaveData(ctx, local, localEncryptedData)
	if err != nil {
		return nil, fmt.Errorf("ClientReadDataUsecase.syncWithRemoteAndDecrypt.gateway.SaveData: %w", err)
	}
	decryptedBytes, err := u.encryptionService.Decrypt(localEncryptedData, dek)
	if err != nil {
		return nil, fmt.Errorf("ClientReadDataUsecase.encryptionService.Decrypt: %w", err)
	}
	return decryptedBytes, nil
}

func (u *ClientReadDataUsecase) checkRemote(
	ctx context.Context,
	local *domainmodels.DataInfo,
) (*domainmodels.DataInfo, error) {
	remotes, err := u.gateway.ListData(ctx)
	if errors.Is(err, domainmodels.ErrOffline) {
		return local, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ClientListDataUsecase.checkRemote.gateway.ListData: %w", err)
	}
	for _, remote := range remotes {
		if remote.UUID == local.UUID && local.LastUpdated.Before(remote.LastUpdated) {
			return remote, nil
		} else if remote.UUID == local.UUID {
			break
		}
	}
	return local, nil
}
