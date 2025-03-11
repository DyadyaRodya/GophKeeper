package usecases

import (
	"context"
	"errors"
	"fmt"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

type (
	// ServerListDataUsecase usecase for handling data reading on server side
	ServerListDataUsecase struct {
		repo interfaces.Repository
	}
	// ClientListDataUsecase usecase for handling data reading on client side
	ClientListDataUsecase struct {
		dataStorage interfaces.ClientStorage
		gateway     interfaces.ListDataGateway
	}
)

// NewListDataUsecase constructor for ServerListDataUsecase
func NewListDataUsecase(
	repo interfaces.Repository,
) *ServerListDataUsecase {
	return &ServerListDataUsecase{
		repo: repo,
	}
}

// Handle reads from DB about data stored
func (u *ServerListDataUsecase) Handle(
	ctx context.Context,
	userUUID string,
) ([]*domainmodels.DataInfo, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("ServerListDataUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	metas, err := dbSess.ReadDataInfoByUserUUID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("ServerListDataUsecase.repo.ListDataInfo: %w", err)
	}

	return metas, nil
}

// NewClientListDataUsecase constructor for ClientListDataUsecase
func NewClientListDataUsecase(
	dataStorage interfaces.ClientStorage,
	gateway interfaces.ListDataGateway,
) *ClientListDataUsecase {
	return &ClientListDataUsecase{
		dataStorage: dataStorage,
		gateway:     gateway,
	}
}

// Handle reads list of metadata from local storage and tries to merge with remote
// and decrypted data
func (u *ClientListDataUsecase) Handle(
	ctx context.Context,
	userUUID string,
) ([]*domainmodels.DataInfo, error) {
	metas, err := u.dataStorage.ListData(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("ClientListDataUsecase.Handle.dataStorage.ListData: %w", err)
	}

	return u.mergeWithRemote(ctx, metas, userUUID)
}

func (u *ClientListDataUsecase) mergeWithRemote(
	ctx context.Context,
	local []*domainmodels.DataInfo,
	userUUID string,
) ([]*domainmodels.DataInfo, error) {
	remote, err := u.gateway.ListData(ctx)
	if errors.Is(err, domainmodels.ErrOffline) {
		return local, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ClientListDataUsecase.mergeWithRemote.gateway.ListData: %w", err)
	}

	merged := make(map[string]*domainmodels.DataInfo, len(local)+len(remote))
	for _, remoteData := range remote {
		merged[remoteData.UUID] = remoteData
	}

	for _, localData := range local {
		if remoteData, ok := merged[localData.UUID]; !ok || remoteData.LastUpdated.Before(localData.LastUpdated) {
			merged[localData.UUID] = localData
		}
	}

	result := make([]*domainmodels.DataInfo, 0, len(merged))
	for _, mergedData := range merged {
		result = append(result, mergedData)
	}
	return result, nil
}
