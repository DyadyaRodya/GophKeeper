package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/dto"
	pb "github.com/DyadyaRodya/GophKeeper/proto/v1"
)

type ClientGRPC struct {
	cc  pb.GophKeeperServiceClient
	jwt string
}

func NewClientGRPC(cc pb.GophKeeperServiceClient) *ClientGRPC {
	return &ClientGRPC{
		cc:  cc,
		jwt: "",
	}
}

func (c *ClientGRPC) Register(ctx context.Context, username, password string) (*dto.Keys, error) {
	res, err := c.cc.Register(ctx, &pb.RegisterRequest{Username: username, Password: password})
	s, _ := status.FromError(err)
	fmt.Println(s)
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		return &dto.Keys{
			UserUUID:      res.UserUuid,
			DEKCiphertext: res.DekCiphertext,
			KEKSalt:       res.KekSalt,
			RecoveryKey:   res.RecoveryKey,
		}, nil
	case codes.AlreadyExists:
		return nil, domainmodels.ErrLoginTaken
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}

func (c *ClientGRPC) Login(ctx context.Context, username, password string) (*domainmodels.ShortKeyInfo, error) {
	res, err := c.cc.Login(ctx, &pb.LoginRequest{Username: username, Password: password})
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		return &domainmodels.ShortKeyInfo{
			UserUUID:      res.UserUuid,
			DEKCiphertext: res.DekCiphertext,
			KEKSalt:       res.KekSalt,
		}, nil
	case codes.Unauthenticated:
		return nil, domainmodels.ErrWrongCredentials
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}

func (c *ClientGRPC) Recover(
	ctx context.Context,
	username string,
	recoveryKey []byte,
	newPassword string,
) (*dto.Keys, error) {
	res, err := c.cc.Recovery(ctx, &pb.RecoveryRequest{
		Username:    username,
		RecoveryKey: recoveryKey,
		NewPassword: newPassword,
	})
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		return &dto.Keys{
			UserUUID:      res.UserUuid,
			DEKCiphertext: res.DekCiphertext,
			KEKSalt:       res.KekSalt,
			RecoveryKey:   res.RecoveryKey,
		}, nil
	case codes.Unauthenticated:
		return nil, domainmodels.ErrWrongCredentials
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}

func (c *ClientGRPC) UpdatePassword(ctx context.Context, oldPassword, newPassword string) (*dto.Keys, error) {
	res, err := c.cc.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
		JwtToken:    c.jwt,
		OldPassword: oldPassword,
		NewPassword: newPassword,
	})
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		return &dto.Keys{
			UserUUID:      res.UserUuid,
			DEKCiphertext: res.DekCiphertext,
			KEKSalt:       res.KekSalt,
			RecoveryKey:   res.RecoveryKey,
		}, nil
	case codes.Unauthenticated:
		return nil, domainmodels.ErrWrongCredentials
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}

func (c *ClientGRPC) SaveData(
	ctx context.Context,
	meta *domainmodels.DataInfo,
	encryptedData []byte,
) (*domainmodels.DataInfo, error) {
	var m *pb.DataMetaInfo
	if meta != nil {
		m = &pb.DataMetaInfo{
			Uuid:        meta.UUID,
			OwnerUuid:   meta.OwnerUUID,
			LastUpdated: meta.LastUpdated.String(),
			IsDeleted:   meta.IsDeleted,
		}
	}
	res, err := c.cc.SaveData(ctx, &pb.SaveDataRequest{
		JwtToken:      c.jwt,
		Meta:          m,
		EncryptedData: encryptedData,
	})
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		t, parseErr := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", res.Meta.LastUpdated)
		if parseErr != nil {
			return nil, fmt.Errorf("ClientGRPC.SaveData: %w", parseErr)
		}
		return &domainmodels.DataInfo{
			UUID:        res.Meta.Uuid,
			OwnerUUID:   res.Meta.OwnerUuid,
			LastUpdated: t,
			IsDeleted:   res.Meta.IsDeleted,
		}, nil
	case codes.FailedPrecondition:
		s, _ := status.FromError(err)
		if strings.Contains(s.Message(), "deleted") {
			return nil, domainmodels.ErrDataDeleted
		}
		return nil, domainmodels.ErrDataConflict
	case codes.Unauthenticated:
		return nil, domainmodels.ErrWrongCredentials
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}

func (c *ClientGRPC) ReadData(ctx context.Context, meta *domainmodels.DataInfo) ([]byte, error) {
	res, err := c.cc.ReadData(ctx, &pb.ReadDataRequest{
		JwtToken: c.jwt,
		DataUuid: meta.UUID,
	})
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		if res.Meta.IsDeleted {
			return nil, domainmodels.ErrDataDeleted
		}
		return res.EncryptedData, nil
	case codes.Unauthenticated:
		return nil, domainmodels.ErrWrongCredentials
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	case codes.NotFound:
		return nil, domainmodels.ErrDataInfoNotFound
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}

func (c *ClientGRPC) ListData(ctx context.Context) ([]*domainmodels.DataInfo, error) {
	res, err := c.cc.ListData(ctx, &pb.ListDataRequest{JwtToken: c.jwt})
	code := status.Code(err)
	switch code {
	case codes.OK:
		c.jwt = res.NewJwtToken
		metas := make([]*domainmodels.DataInfo, 0, len(res.Metas))
		for _, meta := range res.Metas {
			t, parseErr := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", meta.LastUpdated)
			if parseErr != nil {
				return nil, fmt.Errorf("ClientGRPC.ListData: %w", parseErr)
			}
			metas = append(metas, &domainmodels.DataInfo{
				UUID:        meta.Uuid,
				OwnerUUID:   meta.OwnerUuid,
				LastUpdated: t,
				IsDeleted:   meta.IsDeleted,
			})
		}
		return metas, nil
	case codes.Unauthenticated:
		return nil, domainmodels.ErrWrongCredentials
	case codes.Unavailable:
		return nil, domainmodels.ErrOffline
	}
	return nil, errors.Join(err, domainmodels.ErrGateway)
}
