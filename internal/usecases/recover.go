package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/dto"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

type (
	RecoverPasswordService interface {
		interfaces.PasswordHashGenerator
		interfaces.PasswordValidator
		interfaces.PasswordSaltGenerator
	}
	ServerRecoverEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.SaltGenerator
		interfaces.RecoveryKeyGenerator
		interfaces.Decryptor
		interfaces.Encryptor
	}
	ClientRecoverEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.Decryptor
	}
	// ServerRecoverUsecase usecase for handling recover on sever side
	ServerRecoverUsecase struct {
		repo              interfaces.Repository
		passwordService   RecoverPasswordService
		saltSize          int
		encryptionService ServerRecoverEncryptionService
	}
	// ClientRecoverUsecase usecase for handling recover on client side
	ClientRecoverUsecase struct {
		passwordService   interfaces.PasswordValidator
		encryptionService ClientRecoverEncryptionService
		localKeyStorage   interfaces.KeyStorage
		gateway           interfaces.RecoverServerGateway
	}
)

// NewServerRecoverUsecase constructor for
func NewServerRecoverUsecase(
	repo interfaces.Repository,
	ps RecoverPasswordService,
	saltSize int,
	es ServerRecoverEncryptionService,
) *ServerRecoverUsecase {
	return &ServerRecoverUsecase{
		repo:              repo,
		passwordService:   ps,
		saltSize:          saltSize,
		encryptionService: es,
	}
}

// Handle validates new password, decrypts dek using recovery key, generates new kek, recovery key
// and encrypted dek ciphertext. DEK stays the same, so no need to recalculate encryption for data
func (u *ServerRecoverUsecase) Handle(
	ctx context.Context,
	username string,
	recoveryKey []byte,
	newPassword string,
) (*domainmodels.User, *dto.Keys, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	userInfo, err := dbSess.GetUserByUsername(ctx, username)
	if err != nil && !errors.Is(err, domainmodels.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.dbSess.GetUserByUsername: %w", err)
	}

	if userInfo == nil { // same as errors.Is(err, domainmodels.ErrUserNotFound)
		return nil, nil, domainmodels.ErrWrongCredentials
	}

	keysInfo, err := dbSess.GetKeysInfo(ctx, userInfo.UUID)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.dbSess.GetKeysInfo: %w", err)
	}

	dek, err := u.encryptionService.Decrypt(keysInfo.RecoveryDEKCiphertext, recoveryKey)
	if err != nil {
		return nil, nil, domainmodels.ErrWrongCredentials
	}

	if !u.passwordService.Validate(newPassword) {
		return nil, nil, domainmodels.ErrPasswordComplexity
	}

	passwordSalt, err := u.passwordService.GenerateRandomSalt(u.saltSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.passwordService.GenerateRandomSalt: %w", err)
	}

	passwordHash, err := u.passwordService.Hash(newPassword, passwordSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.passwordService.Hash: %w", err)
	}

	newKEKSalt, err := u.encryptionService.GenerateSalt(u.saltSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.passwordService.GenerateRandomSalt: %w", err)
	}
	newKEK := u.encryptionService.GenerateKEK(newPassword, newKEKSalt)
	newEncryptedDEK, err := u.encryptionService.Encrypt(dek, newKEK)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.encryptionService.Encrypt: %w", err)
	}

	newRecoveryKey, err := u.encryptionService.GenerateRecoveryKey()
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.encryptionService.GenerateRecoveryKey: %w", err)
	}

	newEncryptedRecoveryDEK, err := u.encryptionService.Encrypt(dek, newRecoveryKey)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.encryptionService.Encrypt: %w", err)
	}

	userInfo.UpdatedAt = time.Now().UTC()
	userInfo.PasswordHash = passwordHash
	userInfo.PasswordSalt = passwordSalt

	err = dbSess.UpdateUser(ctx, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.dbSess.UpdateUser: %w", err)
	}

	newKeysInfo := &domainmodels.KeysInfo{
		DEKCiphertext:         newEncryptedDEK,
		KEKSalt:               newKEKSalt,
		RecoveryDEKCiphertext: newEncryptedRecoveryDEK,
	}

	err = dbSess.SaveKeysInfo(ctx, userInfo.UUID, newKeysInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.dbSess.SaveKeysInfo: %w", err)
	}

	err = dbSess.Commit(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRecoverUsecase.dbSess.Commit: %w", err)
	}

	keysDTO := &dto.Keys{
		UserUUID:      userInfo.UUID,
		DEKCiphertext: newKeysInfo.DEKCiphertext,
		KEKSalt:       newKeysInfo.KEKSalt,
		RecoveryKey:   newRecoveryKey,
	}

	return userInfo, keysDTO, nil
}

// NewClientRecoverUsecase constructor for ClientRecoverUsecase
func NewClientRecoverUsecase(
	ps interfaces.PasswordValidator,
	es ClientRecoverEncryptionService,
	lks interfaces.KeyStorage,
	gw interfaces.RecoverServerGateway,
) *ClientRecoverUsecase {
	return &ClientRecoverUsecase{
		passwordService:   ps,
		encryptionService: es,
		localKeyStorage:   lks,
		gateway:           gw,
	}
}

// Handle validates new password, makes call to server,
// stores encrypted dek and kek salt locally for offline work.
// If succeeded returns user uuid, dek for current session and new recovery key.
// DEK stays the same, so no need to recalculate encryption for data
func (u *ClientRecoverUsecase) Handle(
	ctx context.Context,
	username string,
	recoveryKey []byte,
	newPassword string,
) (*dto.ClientRecoverResult, error) {
	if !u.passwordService.Validate(newPassword) {
		return nil, domainmodels.ErrPasswordComplexity
	}

	keysInfo, err := u.gateway.Recover(ctx, username, recoveryKey, newPassword)
	if err != nil {
		return nil, fmt.Errorf("ClientRecoverUsecase.gateway.Recover: %w", err)
	}
	localKeysInfo := keysInfo.ToShort()
	err = u.localKeyStorage.SaveKeys(ctx, localKeysInfo)
	if err != nil {
		return nil, fmt.Errorf("ClientRecoverUsecase.localKeyStorage.SaveKeys: %w", err)
	}

	kek := u.encryptionService.GenerateKEK(newPassword, localKeysInfo.KEKSalt)

	dek, err := u.encryptionService.Decrypt(localKeysInfo.DEKCiphertext, kek)
	if err != nil {
		return nil, fmt.Errorf("ClientRecoverUsecase.encryptionService.Decrypt: %w", err)
	}

	return &dto.ClientRecoverResult{UserUUID: keysInfo.UserUUID, DEK: dek, RecoveryKey: keysInfo.RecoveryKey}, nil
}
