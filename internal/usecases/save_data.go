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
	// ServerSaveDataUsecase usecase for handling data save of server side
	ServerSaveDataUsecase struct {
		repo          interfaces.Repository
		uuidGenerator interfaces.UUIDGenerator
		dataStorage   interfaces.ServerStorage
	}
	// ClientSaveDataUsecase usecase for handling data save of client side
	ClientSaveDataUsecase struct {
		gateway           interfaces.SaveDataGateway
		convertService    interfaces.RawDataToBytesConverter
		encryptionService interfaces.Encryptor
		uuidGenerator     interfaces.UUIDGenerator
		dataStorage       interfaces.ClientStorage
	}
)

// ServerSaveDataUsecase constructor for ServerSaveDataUsecase
func NewServerSaveDataUsecase(
	repo interfaces.Repository,
	uuidGenerator interfaces.UUIDGenerator,
	dataStorage interfaces.ServerStorage,
) *ServerSaveDataUsecase {
	return &ServerSaveDataUsecase{
		repo:          repo,
		uuidGenerator: uuidGenerator,
		dataStorage:   dataStorage,
	}
}

// Handle stores encrypted data in storage, returns metadata
func (u *ServerSaveDataUsecase) Handle(
	ctx context.Context,
	userUUID string,
	meta *domainmodels.DataInfo,
	encryptedData []byte,
) (*domainmodels.DataInfo, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("ServerSaveDataUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	if meta == nil {
		var uuid string
		uuid, err = u.uuidGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("ServerSaveDataUsecase.uuidGenerator.Generate: %w", err)
		}

		meta = &domainmodels.DataInfo{
			UUID:        uuid,
			OwnerUUID:   userUUID,
			LastUpdated: time.Now().UTC(),
			IsDeleted:   false,
		}
	} else {
		var existingMeta *domainmodels.DataInfo
		existingMeta, err = dbSess.ReadDataInfo(ctx, userUUID, meta.UUID)
		if err != nil && !errors.Is(err, domainmodels.ErrDataInfoNotFound) {
			return nil, fmt.Errorf("ServerSaveDataUsecase.repo.ReadDataInfo: %w", err)
		}
		if existingMeta != nil && existingMeta.IsDeleted && meta.LastUpdated.Before(existingMeta.LastUpdated) {
			return existingMeta, domainmodels.ErrDataDeleted
		}
		if existingMeta != nil && meta.LastUpdated.Before(existingMeta.LastUpdated) {
			return existingMeta, domainmodels.ErrDataConflict
		}
	}
	err = dbSess.SaveDataInfo(ctx, meta)
	if err != nil {
		return nil, fmt.Errorf("ServerSaveDataUsecase.dbSess.SaveDataInfo: %w", err)
	}

	err = u.dataStorage.SaveData(ctx, meta, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("ServerSaveDataUsecase.dataStorage.SaveData: %w", err)
	}

	err = dbSess.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("ServerSaveDataUsecase.dbSess.Commit: %w", err)
	}
	return meta, nil
}

// NewClientSaveDataUsecase constructor for ClientSaveDataUsecase
func NewClientSaveDataUsecase(
	gateway interfaces.SaveDataGateway,
	convertService interfaces.RawDataToBytesConverter,
	encryptionService interfaces.Encryptor,
	uuidGenerator interfaces.UUIDGenerator,
	dataStorage interfaces.ClientStorage,
) *ClientSaveDataUsecase {
	return &ClientSaveDataUsecase{
		gateway:           gateway,
		convertService:    convertService,
		encryptionService: encryptionService,
		uuidGenerator:     uuidGenerator,
		dataStorage:       dataStorage,
	}
}

// HandleAdd stores new encrypted data in local storage and sends it to the server. Returns metadata
func (u *ClientSaveDataUsecase) HandleAdd(
	ctx context.Context,
	dek []byte,
	userUUID string,
	rawData *domainmodels.RawData,
) (*domainmodels.DataInfo, error) {
	dataBytes, err := u.convertService.ConvertRawDataToBytes(rawData)
	if err != nil {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleAdd.convertService.ConvertRawDataToBytes: %w", err)
	}

	encryptedData, err := u.encryptionService.Encrypt(dataBytes, dek)
	if err != nil {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleAdd.encryptionService.Encrypt: %w", err)
	}
	meta, err := u.gateway.SaveData(ctx, nil, encryptedData)
	offline := err != nil && errors.Is(err, domainmodels.ErrOffline)
	if err != nil && !offline {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleAdd.gateway.SaveData: %w", err)
	}

	if meta == nil { // same as errors.Is(err, domainmodels.ErrOffline)
		var uuid string
		uuid, err = u.uuidGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("ClientSaveDataUsecase.HandleAdd.uuidGenerator.Generate: %w", err)
		}

		meta = &domainmodels.DataInfo{
			UUID:        uuid,
			OwnerUUID:   userUUID,
			LastUpdated: time.Now().UTC(),
			IsDeleted:   false,
		}
	}
	err = u.dataStorage.SaveData(ctx, meta, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleAdd.dataStorage.SaveData: %w", err)
	}

	if offline {
		err = domainmodels.ErrOffline
	}
	return meta, err
}

// HandleUpdate stores updated encrypted data in local storage and sends it to the server. Returns metadata
func (u *ClientSaveDataUsecase) HandleUpdate(
	ctx context.Context,
	dek []byte,
	userUUID string,
	dataUUID string,
	rawData *domainmodels.RawData,
) (*domainmodels.DataInfo, error) {
	dataBytes, err := u.convertService.ConvertRawDataToBytes(rawData)
	if err != nil {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleUpdate.convertService.ConvertRawDataToBytes: %w", err)
	}

	encryptedData, err := u.encryptionService.Encrypt(dataBytes, dek)
	if err != nil {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleUpdate.encryptionService.Encrypt: %w", err)
	}
	meta := &domainmodels.DataInfo{
		UUID:        dataUUID,
		OwnerUUID:   userUUID,
		LastUpdated: time.Now().UTC(),
		IsDeleted:   false,
	}
	meta, err = u.gateway.SaveData(ctx, meta, encryptedData)
	offline := err != nil && errors.Is(err, domainmodels.ErrOffline)
	if err != nil && !offline {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleUpdate.gateway.SaveData: %w", err)
	}

	if meta == nil { // same as errors.Is(err, domainmodels.ErrOffline)
		var uuid string
		uuid, err = u.uuidGenerator.Generate()
		if err != nil {
			return nil, fmt.Errorf("ClientSaveDataUsecase.HandleUpdate.uuidGenerator.Generate: %w", err)
		}

		meta = &domainmodels.DataInfo{
			UUID:        uuid,
			OwnerUUID:   userUUID,
			LastUpdated: time.Now().UTC(),
			IsDeleted:   false,
		}
	}
	err = u.dataStorage.SaveData(ctx, meta, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("ClientSaveDataUsecase.HandleUpdate.dataStorage.SaveData: %w", err)
	}

	if offline {
		err = domainmodels.ErrOffline
	}
	return meta, err
}
