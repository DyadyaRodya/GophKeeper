package usecases

import (
	"context"
	"errors"
	"fmt"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/dto"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

type (
	ClientLoginEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.Decryptor
	}

	// ServerLoginUsecase usecase for handling login on sever side
	ServerLoginUsecase struct {
		repo            interfaces.Repository
		passwordService interfaces.PasswordComparator
	}
	// ClientLoginUsecase usecase for handling login on client side
	ClientLoginUsecase struct {
		encryptionService ClientLoginEncryptionService
		localKeyStorage   interfaces.KeyStorage
		gateway           interfaces.LoginServerGateway
	}
)

// NewServerLoginUsecase constructor for ServerLoginUsecase
func NewServerLoginUsecase(
	repo interfaces.Repository,
	ps interfaces.PasswordComparator,
) *ServerLoginUsecase {
	return &ServerLoginUsecase{
		repo:            repo,
		passwordService: ps,
	}
}

// Handle checks username and password, reads user, encrypted dek and kek salt
func (u *ServerLoginUsecase) Handle(ctx context.Context, username string, password string) (*domainmodels.User, *domainmodels.ShortKeyInfo, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerLoginUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	userInfo, err := dbSess.GetUserByUsername(ctx, username)
	if err != nil && !errors.Is(err, domainmodels.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("ServerLoginUsecase.dbSess.GetUserByUsername: %w", err)
	}

	if userInfo == nil { // same as errors.Is(err, domainmodels.ErrUserNotFound)
		return nil, nil, domainmodels.ErrWrongCredentials
	}

	valid, err := u.passwordService.Compare(password, userInfo.PasswordHash, userInfo.PasswordSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerLoginUsecase.passwordService.Compare: %w", err)
	}

	if !valid {
		return nil, nil, domainmodels.ErrWrongCredentials
	}

	keysInfo, err := dbSess.GetKeysInfo(ctx, userInfo.UUID)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerLoginUsecase.dbSess.GetKeysInfo: %w", err)
	}

	return userInfo, keysInfo.ToShort(userInfo.UUID), nil
}

// NewClientLoginUsecase constructor for ClientLoginUsecase
func NewClientLoginUsecase(
	es ClientLoginEncryptionService,
	lks interfaces.KeyStorage,
	gw interfaces.LoginServerGateway,
) *ClientLoginUsecase {
	return &ClientLoginUsecase{
		encryptionService: es,
		localKeyStorage:   lks,
		gateway:           gw,
	}
}

// HandleOnline makes call to server,
// stores encrypted dek and kek salt locally for offline work.
// If succeeded returns user uuid, current dek for current session and user UUID.
func (u *ClientLoginUsecase) HandleOnline(ctx context.Context, username string, password string) (*dto.ClientLoginResult, error) {
	keysInfo, err := u.gateway.Login(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("ClientLoginUsecase.gateway.Login: %w", err)
	}
	err = u.localKeyStorage.SaveKeys(ctx, keysInfo)
	if err != nil {
		return nil, fmt.Errorf("ClientLoginUsecase.localKeyStorage.SaveKeys: %w", err)
	}

	kek := u.encryptionService.GenerateKEK(password, keysInfo.KEKSalt)

	dek, err := u.encryptionService.Decrypt(keysInfo.DEKCiphertext, kek)
	if err != nil {
		return nil, fmt.Errorf("ClientLoginUsecase.encryptionService.Decrypt: %w", err)
	}

	return &dto.ClientLoginResult{UserUUID: keysInfo.UserUUID, DEK: dek}, nil
}

// HandleOffline reads encrypted dek and kek salt form local storage and decrypts dek with password.
// If succeeded returns user uuid, dek for offline work and user UUID.
func (u *ClientLoginUsecase) HandleOffline(ctx context.Context, password string) (*dto.ClientLoginResult, error) {
	localKeysInfo, err := u.localKeyStorage.ReadKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("ClientLoginUsecase.localKeyStorage.SaveKeys: %w", err)
	}

	kek := u.encryptionService.GenerateKEK(password, localKeysInfo.KEKSalt)

	dek, err := u.encryptionService.Decrypt(localKeysInfo.DEKCiphertext, kek)
	if err != nil {
		return nil, fmt.Errorf("ClientLoginUsecase.encryptionService.Decrypt: %w", err)
	}

	return &dto.ClientLoginResult{UserUUID: localKeysInfo.UserUUID, DEK: dek}, nil
}
