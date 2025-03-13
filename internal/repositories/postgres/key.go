package postgres

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

func (s *StorePGX) InitKeys(ctx context.Context, tx pgx.Tx) error {
	s.logger.Info("Initializing `keys` table")
	_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS public.keys (
        user_uuid UUID NOT NULL PRIMARY KEY REFERENCES public.users(uuid) ON DELETE CASCADE, 
        dek_ciphertext TEXT NOT NULL,
        kek_salt TEXT NOT NULL,
        recovery_dek_ciphertext TEXT NOT NULL)
    `)
	if err != nil {
		s.logger.Error("Failed to create table `keys`", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	return nil
}

func (s *SessionPGX) SaveKeysInfo(ctx context.Context, userUUID string, keysInfo *domainmodels.KeysInfo) error {
	s.logger.Debug("Saving User Keys", zap.String("userUUID", userUUID), zap.Any("keysInfo", keysInfo))

	ct, err := s.tx.Exec(ctx, `
		INSERT INTO public.keys (user_uuid, dek_ciphertext, kek_salt, recovery_dek_ciphertext) VALUES 
			(@user_uuid, @dek_ciphertext, @kek_salt, @recovery_dek_ciphertext)
		ON CONFLICT (user_uuid) DO UPDATE 
		SET dek_ciphertext = excluded.dek_ciphertext, 
			kek_salt = excluded.kek_salt, 
			recovery_dek_ciphertext = excluded.recovery_dek_ciphertext
		`,
		pgx.NamedArgs{
			"user_uuid":               userUUID,
			"dek_ciphertext":          base64.StdEncoding.EncodeToString(keysInfo.DEKCiphertext),
			"kek_salt":                base64.StdEncoding.EncodeToString(keysInfo.KEKSalt),
			"recovery_dek_ciphertext": base64.StdEncoding.EncodeToString(keysInfo.RecoveryDEKCiphertext),
		})

	if err != nil {
		s.logger.Error("Failed to upsert into keys",
			zap.String("userUUID", userUUID),
			zap.Any("keysInfo", keysInfo),
			zap.Error(err))
		return errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.SaveKeysInfo: %w", err))
	}
	if !ct.Insert() && !ct.Update() {
		s.logger.Error("Failed to upsert into keys",
			zap.String("userUUID", userUUID),
			zap.Any("keysInfo", keysInfo),
			zap.Any("commandTag", ct))
		return errors.Join(ErrDBAPI,
			fmt.Errorf("error in SessionPGX.SaveKeysInfo: not upserted keys for user: %s", userUUID))
	}
	return nil
}
func (s *SessionPGX) GetKeysInfo(ctx context.Context, userUUID string) (*domainmodels.KeysInfo, error) {
	s.logger.Debug("GetKeysInfo", zap.String("userUUID", userUUID))

	var dekCiphertext, kekSalt, recoveryDEKCiphertext string
	err := s.tx.QueryRow(ctx, `SELECT dek_ciphertext, kek_salt, recovery_dek_ciphertext FROM public.keys 
                                   WHERE user_uuid = @user_uuid`,
		pgx.NamedArgs{"user_uuid": userUUID},
	).Scan(&dekCiphertext, &kekSalt, &recoveryDEKCiphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainmodels.ErrUserKeysNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get keys by user uuid", zap.String("userUUID", userUUID), zap.Error(err))
		return nil, errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.GetKeysInfo: %w", err))
	}

	DEKCiphertextBytes, err := base64.StdEncoding.DecodeString(dekCiphertext)
	if err != nil {
		s.logger.Error("Failed to decode dek ciphertext", zap.String("userUUID", userUUID), zap.Error(err))
		return nil, fmt.Errorf("SessionPGX.GetKeysInfo.base64.StdEncoding.DecodeString: %w", err)
	}
	KEKSaltBytes, err := base64.StdEncoding.DecodeString(kekSalt)
	if err != nil {
		s.logger.Error("Failed to decode kek salt", zap.String("userUUID", userUUID), zap.Error(err))
		return nil, fmt.Errorf("SessionPGX.GetKeysInfo.base64.StdEncoding.DecodeString: %w", err)
	}
	RecoveryDEKCiphertextBytes, err := base64.StdEncoding.DecodeString(recoveryDEKCiphertext)
	if err != nil {
		s.logger.Error("Failed to decode recovery dek ciphertext",
			zap.String("userUUID", userUUID), zap.Error(err))
		return nil, fmt.Errorf("SessionPGX.GetKeysInfo.base64.StdEncoding.DecodeString: %w", err)
	}

	return &domainmodels.KeysInfo{
		DEKCiphertext:         DEKCiphertextBytes,
		KEKSalt:               KEKSaltBytes,
		RecoveryDEKCiphertext: RecoveryDEKCiphertextBytes,
	}, nil
}
