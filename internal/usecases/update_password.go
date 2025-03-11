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
	UpdatePasswordPasswordService interface {
		interfaces.PasswordHashGenerator
		interfaces.PasswordValidator
		interfaces.PasswordSaltGenerator
		interfaces.PasswordComparator
	}
	ServerUpdatePasswordEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.SaltGenerator
		interfaces.RecoveryKeyGenerator
		interfaces.Decryptor
		interfaces.Encryptor
	}
	ClientUpdatePasswordEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.Decryptor
	}
	// ServerUpdatePasswordUsecase usecase for handling update password on sever side
	ServerUpdatePasswordUsecase struct {
		repo              interfaces.Repository
		passwordService   UpdatePasswordPasswordService
		saltSize          int
		encryptionService ServerUpdatePasswordEncryptionService
	}
	// ClientUpdatePasswordUsecase usecase for handling update password on client side
	ClientUpdatePasswordUsecase struct {
		passwordService   interfaces.PasswordValidator
		encryptionService ClientUpdatePasswordEncryptionService
		localKeyStorage   interfaces.KeyStorage
		gateway           interfaces.UpdatePasswordServerGateway
	}
)

// NewServerUpdatePasswordUsecase constructor for ServerUpdatePasswordUsecase
func NewServerUpdatePasswordUsecase(
	repo interfaces.Repository,
	ps UpdatePasswordPasswordService,
	saltSize int,
	es ServerUpdatePasswordEncryptionService,
) *ServerUpdatePasswordUsecase {
	return &ServerUpdatePasswordUsecase{
		repo:              repo,
		passwordService:   ps,
		saltSize:          saltSize,
		encryptionService: es,
	}
}

// Handle checks old password, validates new password, decrypts dek using old password, generates new kek, recovery key
// and encrypted dek ciphertext. DEK stays the same, so no need to recalculate encryption for data
func (u *ServerUpdatePasswordUsecase) Handle(
	ctx context.Context,
	userUUID string,
	oldPassword string,
	newPassword string,
) (*domainmodels.User, *dto.Keys, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	userInfo, err := dbSess.GetUserByUUID(ctx, userUUID)
	if err != nil && !errors.Is(err, domainmodels.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.dbSess.GetUserByUUID: %w", err)
	}

	if userInfo == nil { // same as errors.Is(err, domainmodels.ErrUserNotFound)
		return nil, nil, domainmodels.ErrUserNotFound
	}

	valid, err := u.passwordService.Compare(oldPassword, userInfo.PasswordHash, userInfo.PasswordSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerLoginUsecase.passwordService.Compare: %w", err)
	}

	if !valid {
		return nil, nil, domainmodels.ErrUserNotFound
	}

	if !u.passwordService.Validate(newPassword) {
		return nil, nil, domainmodels.ErrPasswordComplexity
	}

	passwordSalt, err := u.passwordService.GenerateRandomSalt(u.saltSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.passwordService.GenerateRandomSalt: %w", err)
	}

	passwordHash, err := u.passwordService.Hash(newPassword, passwordSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.passwordService.Hash: %w", err)
	}

	keysInfo, err := dbSess.GetKeysInfo(ctx, userUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.dbSess.GetKeysInfo: %w", err)
	}

	kek := u.encryptionService.GenerateKEK(oldPassword, keysInfo.KEKSalt)

	dek, err := u.encryptionService.Decrypt(keysInfo.DEKCiphertext, kek)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.encryptionService.GenerateDEK: %w", err)
	}

	newKEKSalt, err := u.encryptionService.GenerateSalt(u.saltSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.passwordService.GenerateRandomSalt: %w", err)
	}
	newKEK := u.encryptionService.GenerateKEK(newPassword, newKEKSalt)
	newEncryptedDEK, err := u.encryptionService.Encrypt(dek, newKEK)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.encryptionService.Encrypt: %w", err)
	}

	newRecoveryKey, err := u.encryptionService.GenerateRecoveryKey()
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.encryptionService.GenerateRecoveryKey: %w", err)
	}

	newEncryptedRecoveryDEK, err := u.encryptionService.Encrypt(dek, newRecoveryKey)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.encryptionService.Encrypt: %w", err)
	}

	userInfo.UpdatedAt = time.Now().UTC()
	userInfo.PasswordHash = passwordHash
	userInfo.PasswordSalt = passwordSalt

	err = dbSess.UpdateUser(ctx, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.dbSess.AddUser: %w", err)
	}

	newKeysInfo := &domainmodels.KeysInfo{
		DEKCiphertext:         newEncryptedDEK,
		KEKSalt:               newKEKSalt,
		RecoveryDEKCiphertext: newEncryptedRecoveryDEK,
	}

	err = dbSess.SaveKeysInfo(ctx, userUUID, newKeysInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.dbSess.SaveKeysInfo: %w", err)
	}

	err = dbSess.Commit(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerUpdatePasswordUsecase.dbSess.Commit: %w", err)
	}

	keysDTO := &dto.Keys{
		UserUUID:      userUUID,
		DEKCiphertext: newKeysInfo.DEKCiphertext,
		KEKSalt:       newKeysInfo.KEKSalt,
		RecoveryKey:   newRecoveryKey,
	}

	return userInfo, keysDTO, nil
}

// NewClientUpdatePasswordUsecase constructor for ClientUpdatePasswordUsecase
func NewClientUpdatePasswordUsecase(
	ps interfaces.PasswordValidator,
	es ClientUpdatePasswordEncryptionService,
	lks interfaces.KeyStorage,
	gw interfaces.UpdatePasswordServerGateway,
) *ClientUpdatePasswordUsecase {
	return &ClientUpdatePasswordUsecase{
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
func (u *ClientUpdatePasswordUsecase) Handle(
	ctx context.Context,
	oldPassword,
	newPassword string,
) (*dto.ClientRecoverResult, error) {
	if !u.passwordService.Validate(newPassword) {
		return nil, domainmodels.ErrPasswordComplexity
	}

	keysInfo, err := u.gateway.UpdatePassword(ctx, oldPassword, newPassword)
	if err != nil {
		return nil, fmt.Errorf("ClientUpdatePasswordUsecase.gateway.UpdatePassword: %w", err)
	}
	localKeysInfo := keysInfo.ToShort()
	err = u.localKeyStorage.SaveKeys(ctx, localKeysInfo)
	if err != nil {
		return nil, fmt.Errorf("ClientUpdatePasswordUsecase.localKeyStorage.SaveKeys: %w", err)
	}

	kek := u.encryptionService.GenerateKEK(newPassword, localKeysInfo.KEKSalt)

	dek, err := u.encryptionService.Decrypt(localKeysInfo.DEKCiphertext, kek)
	if err != nil {
		return nil, fmt.Errorf("ClientUpdatePasswordUsecase.encryptionService.Decrypt: %w", err)
	}

	return &dto.ClientRecoverResult{UserUUID: keysInfo.UserUUID, DEK: dek, RecoveryKey: keysInfo.RecoveryKey}, nil
}
