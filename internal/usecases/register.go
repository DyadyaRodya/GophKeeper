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
	RegisterPasswordService interface {
		interfaces.PasswordHashGenerator
		interfaces.PasswordValidator
		interfaces.PasswordSaltGenerator
	}
	ServerRegisterEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.DEKGenerator
		interfaces.SaltGenerator
		interfaces.RecoveryKeyGenerator
		interfaces.Encryptor
	}
	ClientRegisterEncryptionService interface {
		interfaces.KEKGenerator
		interfaces.Decryptor
	}
	// ServerRegisterUsecase usecase for handling register on sever side
	ServerRegisterUsecase struct {
		repo              interfaces.Repository
		uuidGenerator     interfaces.UUIDGenerator
		passwordService   RegisterPasswordService
		usernameService   interfaces.UsernameService
		saltSize          int
		encryptionService ServerRegisterEncryptionService
	}
	// ClientRegisterUsecase usecase for handling register on client side
	ClientRegisterUsecase struct {
		passwordService   interfaces.PasswordValidator
		usernameService   interfaces.UsernameService
		encryptionService ClientRegisterEncryptionService
		localKeyStorage   interfaces.KeyStorage
		gateway           interfaces.RegisterServerGateway
	}
)

// NewServerRegisterUsecase constructor for ServerRegisterUsecase
func NewServerRegisterUsecase(
	repo interfaces.Repository,
	uuidGenerator interfaces.UUIDGenerator,
	ps RegisterPasswordService,
	us interfaces.UsernameService,
	saltSize int,
	es ServerRegisterEncryptionService,
) *ServerRegisterUsecase {
	return &ServerRegisterUsecase{
		repo:              repo,
		uuidGenerator:     uuidGenerator,
		passwordService:   ps,
		usernameService:   us,
		saltSize:          saltSize,
		encryptionService: es,
	}
}

// Handle validates username and password, creates user, generates dek, kek, recovery key
func (u *ServerRegisterUsecase) Handle(
	ctx context.Context,
	username string,
	password string,
) (*domainmodels.User, *dto.Keys, error) {
	dbSess, err := u.repo.NewSession(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.repo.NewSession: %w", err)
	}
	defer dbSess.Close(ctx)

	err = u.usernameService.Validate(username)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.usernameService.Validate: %w", err)
	}

	userInfo, err := dbSess.GetUserByUsername(ctx, username)
	if err != nil && !errors.Is(err, domainmodels.ErrUserNotFound) {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.dbSess.GetUserByUsername: %w", err)
	}

	if userInfo != nil { // same as !errors.Is(err, domainmodels.ErrUserNotFound)
		return nil, nil, domainmodels.ErrLoginTaken
	}

	if !u.passwordService.Validate(password) {
		return nil, nil, domainmodels.ErrPasswordComplexity
	}

	uuid, err := u.uuidGenerator.Generate()
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.uuidGenerator.Generate: %w", err)
	}

	passwordSalt, err := u.passwordService.GenerateRandomSalt(u.saltSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.passwordService.GenerateRandomSalt: %w", err)
	}

	passwordHash, err := u.passwordService.Hash(password, passwordSalt)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.passwordService.Hash: %w", err)
	}

	dek, err := u.encryptionService.GenerateDEK()
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.encryptionService.GenerateDEK: %w", err)
	}
	kekSalt, err := u.encryptionService.GenerateSalt(u.saltSize)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.passwordService.GenerateRandomSalt: %w", err)
	}
	kek := u.encryptionService.GenerateKEK(password, kekSalt)
	encryptedDEK, err := u.encryptionService.Encrypt(dek, kek)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.encryptionService.Encrypt: %w", err)
	}

	recoveryKey, err := u.encryptionService.GenerateRecoveryKey()
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.encryptionService.GenerateRandomSalt: %w", err)
	}

	encryptedRecoveryDEK, err := u.encryptionService.Encrypt(dek, recoveryKey)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.encryptionService.Encrypt: %w", err)
	}

	t := time.Now().UTC()
	userInfo = &domainmodels.User{
		UUID:         uuid,
		Username:     username,
		CreatedAt:    t,
		UpdatedAt:    t,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		IsActive:     true,
	}

	err = dbSess.AddUser(ctx, userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.dbSess.AddUser: %w", err)
	}

	keysInfo := &domainmodels.KeysInfo{
		DEKCiphertext:         encryptedDEK,
		KEKSalt:               kekSalt,
		RecoveryDEKCiphertext: encryptedRecoveryDEK,
	}

	err = dbSess.SaveKeysInfo(ctx, uuid, keysInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.dbSess.SaveKeysInfo: %w", err)
	}

	err = dbSess.Commit(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("ServerRegisterUsecase.dbSess.Commit: %w", err)
	}

	keysDTO := &dto.Keys{
		UserUUID:      uuid,
		DEKCiphertext: keysInfo.DEKCiphertext,
		KEKSalt:       keysInfo.KEKSalt,
		RecoveryKey:   recoveryKey,
	}

	return userInfo, keysDTO, nil
}

// NewClientRegisterUsecase constructor for ClientRegisterUsecase
func NewClientRegisterUsecase(
	ps interfaces.PasswordValidator,
	us interfaces.UsernameService,
	es ClientRegisterEncryptionService,
	lks interfaces.KeyStorage,
	gw interfaces.RegisterServerGateway,
) *ClientRegisterUsecase {
	return &ClientRegisterUsecase{
		passwordService:   ps,
		usernameService:   us,
		encryptionService: es,
		localKeyStorage:   lks,
		gateway:           gw,
	}
}

// Handle validates username and password, makes call to server,
// stores encrypted dek and kek salt locally for offline work.
// If succeeded returns user uuid, current dek for current session and recovery key.
func (u *ClientRegisterUsecase) Handle(
	ctx context.Context,
	username string,
	password string,
) (*dto.ClientRegisterResult, error) {
	err := u.usernameService.Validate(username)
	if err != nil {
		return nil, fmt.Errorf("ClientRegisterUsecase.usernameService.Validate: %w", err)
	}

	if !u.passwordService.Validate(password) {
		return nil, domainmodels.ErrPasswordComplexity
	}

	keysInfo, err := u.gateway.Register(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("ClientRegisterUsecase.gateway.Register: %w", err)
	}
	localKeysInfo := keysInfo.ToShort()
	err = u.localKeyStorage.SaveKeys(ctx, localKeysInfo)
	if err != nil {
		return nil, fmt.Errorf("ClientRegisterUsecase.localKeyStorage.SaveKeys: %w", err)
	}

	kek := u.encryptionService.GenerateKEK(password, localKeysInfo.KEKSalt)

	dek, err := u.encryptionService.Decrypt(localKeysInfo.DEKCiphertext, kek)
	if err != nil {
		return nil, fmt.Errorf("ClientRegisterUsecase.encryptionService.Decrypt: %w", err)
	}

	return &dto.ClientRegisterResult{UserUUID: keysInfo.UserUUID, DEK: dek, RecoveryKey: keysInfo.RecoveryKey}, nil
}
