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

func (s *StorePGX) InitData(ctx context.Context, tx pgx.Tx) error {
	s.logger.Info("Initializing `data` table")
	_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS public.data (
        uuid UUID NOT NULL PRIMARY KEY,
        user_uuid UUID NOT NULL REFERENCES public.users(uuid) ON DELETE CASCADE, 
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        is_deleted BOOLEAN NOT NULL DEFAULT FALSE)
    `)
	if err != nil {
		s.logger.Error("Failed to create table `data`", zap.Error(err))
		return errors.Join(ErrDBAPI, err)
	}
	return nil
}

func (s *SessionPGX) SaveDataInfo(ctx context.Context, dataInfo *domainmodels.DataInfo) error {
	s.logger.Debug("Saving User Data", zap.Any("dataInfo", dataInfo))

	ct, err := s.tx.Exec(ctx, `
		INSERT INTO public.data (uuid, user_uuid, updated_at, is_deleted) VALUES 
			(@uuid, @user_uuid, @updated_at, @is_deleted)
		ON CONFLICT (uuid) DO UPDATE 
		SET updated_at = excluded.updated_at, 
			is_deleted = excluded.is_deleted
		`,
		pgx.NamedArgs{
			"uuid":       dataInfo.UUID,
			"user_uuid":  dataInfo.OwnerUUID,
			"updated_at": dataInfo.LastUpdated,
			"is_deleted": dataInfo.IsDeleted,
		})

	if err != nil {
		s.logger.Error("Failed to upsert into data",
			zap.Any("dataInfo", dataInfo),
			zap.Error(err))
		return errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.SaveDataInfo: %w", err))
	}
	if !ct.Insert() && !ct.Update() {
		s.logger.Error("Failed to upsert into data",
			zap.Any("dataInfo", dataInfo),
			zap.Any("commandTag", ct))
		return errors.Join(ErrDBAPI,
			fmt.Errorf("error in SessionPGX.SaveDataInfo: not upserted data for user: %s", dataInfo.OwnerUUID))
	}
	return nil
}

func (s *SessionPGX) ReadDataInfo(
	ctx context.Context,
	userUUID string,
	dataUUID string,
) (*domainmodels.DataInfo, error) {
	s.logger.Debug("ReadDataInfo", zap.String("userUUID", userUUID), zap.String("dataUUID", dataUUID))

	var updatedAt time.Time
	var isDeleted bool
	err := s.tx.QueryRow(ctx, `SELECT updated_at, is_deleted FROM public.data 
                                   WHERE uuid = @uuid AND user_uuid = @user_uuid`,
		pgx.NamedArgs{"user_uuid": userUUID, "uuid": dataUUID},
	).Scan(&updatedAt, &isDeleted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainmodels.ErrDataInfoNotFound
	}
	if err != nil {
		s.logger.Error("Failed to get data by user by uuid",
			zap.String("userUUID", userUUID), zap.String("dataUUID", dataUUID), zap.Error(err))
		return nil, errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.ReadDataInfo: %w", err))
	}

	return &domainmodels.DataInfo{
		UUID:        dataUUID,
		OwnerUUID:   userUUID,
		LastUpdated: updatedAt,
		IsDeleted:   isDeleted,
	}, nil
}

func (s *SessionPGX) ReadDataInfoByUserUUID(ctx context.Context, userUUID string) ([]*domainmodels.DataInfo, error) {
	s.logger.Debug("ReadDataInfoByUserUUID", zap.String("userUUID", userUUID))

	rows, err := s.tx.Query(ctx, `SELECT uuid, updated_at, is_deleted FROM public.data WHERE user_uuid = @user_uuid`,
		pgx.NamedArgs{"user_uuid": userUUID},
	)
	if err != nil {
		s.logger.Error("Failed to get data by user uuid",
			zap.String("userUUID", userUUID), zap.Error(err))
		return nil, errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.ReadDataInfoByUserUUID: %w", err))
	}
	dataList := make([]*domainmodels.DataInfo, 0)
	for rows.Next() {
		var updatedAt time.Time
		var isDeleted bool
		var uuid string
		if err = rows.Scan(&uuid, &updatedAt, &isDeleted); err != nil {
			s.logger.Error("Failed to scan row data by user uuid",
				zap.String("userUUID", userUUID), zap.Error(err))
			return nil, errors.Join(ErrDBAPI, fmt.Errorf("SessionPGX.ReadDataInfoByUserUUID: %w", err))
		}
		data := &domainmodels.DataInfo{
			UUID:        uuid,
			OwnerUUID:   userUUID,
			LastUpdated: updatedAt,
			IsDeleted:   isDeleted,
		}
		dataList = append(dataList, data)
	}

	return dataList, nil
}
