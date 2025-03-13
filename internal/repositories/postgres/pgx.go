package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/interfaces"
)

var ErrDBAPI = errors.Join(domainmodels.ErrInternalServer, errors.New("database api error"))

type StorePGX struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

func NewStorePGX(pool *pgxpool.Pool, logger *zap.Logger) *StorePGX {
	return &StorePGX{pool: pool, logger: logger}
}

func (s *StorePGX) InitSchema(ctx context.Context) error {
	s.logger.Info("Initializing schema")
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error("Failed to start transaction", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	defer tx.Rollback(ctx)

	err = s.InitUsers(ctx, tx)
	if err != nil {
		s.logger.Error("Failed to initialize users", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}

	err = s.InitKeys(ctx, tx)
	if err != nil {
		s.logger.Error("Failed to initialize keys", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}

	err = s.InitData(ctx, tx)
	if err != nil {
		s.logger.Error("Failed to initialize data", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	s.logger.Info("Initializing schema done")
	return nil
}

type SessionPGX struct {
	tx     pgx.Tx
	logger *zap.Logger
}

func (s *StorePGX) NewSession(ctx context.Context) (interfaces.RepositorySession, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.Error("Failed to start transaction", zap.Error(err))
		return nil, errors.Join(ErrDBAPI, fmt.Errorf("error in StorePGX.NewSession: %w", err))
	}
	txPGX := &SessionPGX{tx: tx, logger: s.logger}
	return txPGX, nil
}

func (s *SessionPGX) Commit(ctx context.Context) error {
	err := s.tx.Commit(ctx)
	if err != nil {
		s.logger.Error("Failed to Commit", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	return nil
}

func (s *SessionPGX) Close(ctx context.Context) error {
	err := s.tx.Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		s.logger.Error("Failed to Rollback", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	return nil
}
