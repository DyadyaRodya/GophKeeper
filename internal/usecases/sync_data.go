package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"golang.org/x/sync/errgroup"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

const syncParallelLimit = 4

type (
	ClientSyncDataGateway interface {
		interfaces.SaveDataGateway
		interfaces.ReadDataGateway
		interfaces.ListDataGateway
	}

	// ClientSyncDataUsecase handles sync with server
	ClientSyncDataUsecase struct {
		dataStorage interfaces.ClientStorage
		gateway     ClientSyncDataGateway
	}
)

// NewClientSyncDataUsecase constructor for ClientSyncDataUsecase
func NewClientSyncDataUsecase(
	dataStorage interfaces.ClientStorage,
	gateway ClientSyncDataGateway,
) *ClientSyncDataUsecase {
	return &ClientSyncDataUsecase{
		dataStorage: dataStorage,
		gateway:     gateway,
	}
}

// Handle compares local and remote lists, saves newer data from server local, sends updates to server.
func (u *ClientSyncDataUsecase) Handle(ctx context.Context, userUUID string) error {
	remote, err := u.gateway.ListData(ctx)
	if err != nil {
		return fmt.Errorf("ClientListDataUsecase.mergeWithRemote.gateway.ListData: %w", err)
	}
	local, err := u.dataStorage.ListData(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("ClientListDataUsecase.Handle.dataStorage.ListData: %w", err)
	}
	remoteMap := make(map[string]*domainmodels.DataInfo, len(remote))
	for _, remoteData := range remote {
		remoteMap[remoteData.UUID] = remoteData
	}

	syncedUUIDs := make([]string, 0, len(local)+len(remote))

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(syncParallelLimit) // to prevent full memory utilization

	for _, localData := range local {
		loc := localData
		syncedUUIDs = append(syncedUUIDs, loc.UUID)
		remoteData, ok := remoteMap[loc.UUID]
		if !localData.IsDeleted && (!ok || remoteData.LastUpdated.Before(localData.LastUpdated)) { // local is newer
			eg.Go(func() error {
				return u.handleLocalNewer(ctx, localData)
			})
		} else if localData.IsDeleted && !ok { // clean local notes
			eg.Go(func() error {
				return u.cleanLocal(ctx, loc)
			})
		} else if ok && remoteData.LastUpdated.Before(localData.LastUpdated) && localData.IsDeleted {
			eg.Go(func() error {
				return u.notifyRemoteDeleted(ctx, loc)
			})
		} else if ok && localData.LastUpdated.Before(remoteData.LastUpdated) && !remoteData.IsDeleted {
			rem := remoteData
			eg.Go(func() error {
				return u.handleRemoteNewer(ctx, rem)
			})
		} else if ok && localData.LastUpdated.Before(remoteData.LastUpdated) { // remote deleted, need to clean local
			eg.Go(func() error {
				return u.cleanLocal(ctx, loc)
			})
		}
	}

	for remUUID, remoteData := range remoteMap {
		if !remoteData.IsDeleted && !slices.Contains(syncedUUIDs, remUUID) {
			rem := remoteData
			eg.Go(func() error {
				return u.handleRemoteNewer(ctx, rem)
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("ClientSyncDataUsecase.Handle.eg: %w", err)
	}
	return nil
}

func (u *ClientSyncDataUsecase) handleRemoteNewer(ctx context.Context, rem *domainmodels.DataInfo) error {
	data, err := u.gateway.ReadData(ctx, rem)
	if err != nil {
		return fmt.Errorf("ClientSyncDataUsecase.handleRemoteNewer.gateway.ReadData: %w", err)
	}
	err = u.dataStorage.SaveData(ctx, rem, data)
	if err != nil {
		return fmt.Errorf("ClientSyncDataUsecase.handleRemoteNewer.dataStorage.SaveData: %w", err)
	}
	return nil
}

func (u *ClientSyncDataUsecase) handleLocalNewer(ctx context.Context, local *domainmodels.DataInfo) error {
	local, data, err := u.dataStorage.ReadDataWithMeta(ctx, local.OwnerUUID, local.UUID)
	if err != nil {
		return fmt.Errorf("ClientSyncDataUsecase.handleLocalNewer.gateway.ReadDataWithMeta: %w", err)
	}
	_, err = u.gateway.SaveData(ctx, local, data)
	if err != nil {
		return fmt.Errorf("ClientSyncDataUsecase.handleLocalNewer.gateway.SaveData: %w", err)
	}
	return nil
}

func (u *ClientSyncDataUsecase) cleanLocal(ctx context.Context, meta *domainmodels.DataInfo) error {
	err := u.dataStorage.DeleteData(ctx, meta)
	if err != nil {
		return fmt.Errorf("ClientSyncDataUsecase.cleanLocal.gateway.DeleteData: %w", err)
	}
	return nil
}

func (u *ClientSyncDataUsecase) notifyRemoteDeleted(ctx context.Context, local *domainmodels.DataInfo) error {
	_, err := u.gateway.SaveData(ctx, local, nil) // try notify server to delete
	if err != nil &&
		!errors.Is(err, domainmodels.ErrDataDeleted) {
		return fmt.Errorf("ClientSyncDataUsecase.notifyRemoteDeleted.gateway.SaveData: %w", err)
	}
	return u.cleanLocal(ctx, local)
}
