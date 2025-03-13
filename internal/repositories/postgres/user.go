package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
)

func (s *StorePGX) InitUsers(ctx context.Context, tx pgx.Tx) error {
	s.logger.Info("Initializing `users` table")
	_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS public.users (
        uuid UUID NOT NULL PRIMARY KEY, 
        username VARCHAR(255) UNIQUE NOT NULL,
        created_at TIMESTAMPTZ NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NULL DEFAULT NOW(),
		password_hash VARCHAR(255),
    	password_salt VARCHAR(255),
	    is_active BOOLEAN NOT NULL DEFAULT TRUE)
    `)
	if err != nil {
		s.logger.Error("Failed to create table `users`", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	return nil
}

func (s *SessionPGX) AddUser(ctx context.Context, user *domainmodels.User) error {
	s.logger.Debug("Adding User", zap.Any("user", user))

	ct, err := s.tx.Exec(ctx, `
		INSERT INTO public.users (uuid, username, created_at, updated_at, password_hash, password_salt, is_active) VALUES 
			(@uuid, @username, @created_at, @updated_at, @password_hash, @password_salt, @is_active)
		`,
		pgx.NamedArgs{
			"uuid":          user.UUID,
			"username":      user.Username,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
			"password_hash": user.PasswordHash,
			"password_salt": user.PasswordSalt,
			"is_active":     user.IsActive,
		})

	if err != nil {
		s.logger.Error("Failed to insert into users",
			zap.Any("user", user),
			zap.Error(err))
		return errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.AddUser: %w", err))
	}
	if !ct.Insert() {
		s.logger.Error("Failed to insert into users",
			zap.Any("user", user),
			zap.Any("commandTag", ct))
		return errors.Join(ErrDBAPI, fmt.Errorf("error in SessionPGX.AddUser: not inserted user: %v", user))
	}
	return nil
}

func (s *SessionPGX) UpdateUser(ctx context.Context, user *domainmodels.User) error {
	s.logger.Debug("Updating User", zap.Any("user", user))

	ct, err := s.tx.Exec(ctx, `
		UPDATE public.users SET 
		    username = @username, 
		    created_at = @created_at, 
		    updated_at = @updated_at, 
		    password_hash = @password_hash, 
		    password_salt = @password_salt, 
		    is_active = @is_active
		WHERE uuid = @uuid
		`,
		pgx.NamedArgs{
			"uuid":          user.UUID,
			"username":      user.Username,
			"created_at":    user.CreatedAt,
			"updated_at":    user.UpdatedAt,
			"password_hash": user.PasswordHash,
			"password_salt": user.PasswordSalt,
			"is_active":     user.IsActive,
		})

	if err != nil {
		s.logger.Error("Failed to update table users",
			zap.Any("user", user),
			zap.Error(err))
		return errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.UpdateUser: %w", err))
	}
	if !ct.Update() {
		s.logger.Error("Failed to update table users",
			zap.Any("user", user),
			zap.Any("commandTag", ct))
		return errors.Join(ErrDBAPI, fmt.Errorf("error in SessionPGX.UpdateUser: not updated user: %v", user))
	}
	return nil
}
func (s *SessionPGX) GetUserByUsername(ctx context.Context, username string) (*domainmodels.User, error) {
	s.logger.Debug("GetUserByUsername", zap.String("username", username))

	var uuid, passwordHash, passwordSalt string
	var createdAt time.Time
	var updatedAt time.Time
	err := s.tx.QueryRow(ctx, `SELECT 
										uuid, 
										created_at, 
										updated_at, 
										password_hash, 
										password_salt
                                   FROM public.users 
                                   WHERE username = @username AND is_active = TRUE`,
		pgx.NamedArgs{"username": username},
	).Scan(&uuid, &createdAt, &updatedAt, &passwordHash, &passwordSalt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainmodels.ErrUserNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get user by username", zap.String("username", username), zap.Error(err))
		return nil, errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.GetUserByUsername: %w", err))
	}

	return &domainmodels.User{
		UUID:         uuid,
		Username:     username,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		IsActive:     true,
	}, nil
}
func (s *SessionPGX) GetUserByUUID(ctx context.Context, uuid string) (*domainmodels.User, error) {
	s.logger.Debug("GetUserByUUID", zap.String("uuid", uuid))

	var username, passwordHash, passwordSalt string
	var createdAt time.Time
	var updatedAt time.Time
	err := s.tx.QueryRow(ctx, `SELECT 
										username, 
										created_at, 
										updated_at, 
										password_hash, 
										password_salt
                                   FROM public.users 
                                   WHERE uuid = @uuid AND is_active = TRUE`,
		pgx.NamedArgs{"uuid": uuid},
	).Scan(&username, &createdAt, &updatedAt, &passwordHash, &passwordSalt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainmodels.ErrUserNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get user by username", zap.String("uuid", uuid), zap.Error(err))
		return nil, errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.GetUserByUUID: %w", err))
	}

	return &domainmodels.User{
		UUID:         uuid,
		Username:     username,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		PasswordHash: passwordHash,
		PasswordSalt: passwordSalt,
		IsActive:     true,
	}, nil
}
