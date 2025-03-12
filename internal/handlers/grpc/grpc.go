package grpc

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/internal/usecases/dto"
	pb "github.com/DyadyaRodya/GophKeeper/proto/v1"
)

type (
	ListDataUsecase interface {
		Handle(ctx context.Context, userUUID string) ([]*domainmodels.DataInfo, error)
	}
	LoginUsecase interface {
		Handle(
			ctx context.Context,
			username string,
			password string,
		) (*domainmodels.User, *domainmodels.ShortKeyInfo, error)
	}
	ReadDataUsecase interface {
		Handle(
			ctx context.Context,
			userUUID string,
			dataUUID string,
		) (*domainmodels.DataInfo, []byte, error)
	}
	RecoverUsecase interface {
		Handle(
			ctx context.Context,
			username string,
			recoveryKey []byte,
			newPassword string,
		) (*domainmodels.User, *dto.Keys, error)
	}
	RegisterUsecase interface {
		Handle(ctx context.Context, username string, password string) (*domainmodels.User, *dto.Keys, error)
	}
	SaveDataUsecase interface {
		Handle(
			ctx context.Context,
			userUUID string,
			meta *domainmodels.DataInfo,
			encryptedData []byte,
		) (*domainmodels.DataInfo, error)
	}
	UpdatePasswordUsecase interface {
		Handle(
			ctx context.Context,
			userUUID string,
			oldPassword string,
			newPassword string,
		) (*domainmodels.User, *dto.Keys, error)
	}

	JWTService interface {
		ProcessToken(token string) (userUUID string, authenticated bool)
		GenerateToken(userUUID string) (string, error)
	}

	// Handlers supports all required server calls
	Handlers struct {
		pb.UnimplementedGophKeeperServiceServer
		JWTService            JWTService
		ListDataUsecase       ListDataUsecase
		LoginUsecase          LoginUsecase
		ReadDataUsecase       ReadDataUsecase
		RecoverUsecase        RecoverUsecase
		RegisterUsecase       RegisterUsecase
		SaveDataUsecase       SaveDataUsecase
		UpdatePasswordUsecase UpdatePasswordUsecase
	}
)

// NewHandlers constructor for Handlers
func NewHandlers(
	jwtService JWTService,
	listDataUsecase ListDataUsecase,
	loginUsecase LoginUsecase,
	readDataUsecase ReadDataUsecase,
	recoverUsecase RecoverUsecase,
	registerUsecase RegisterUsecase,
	saveDataUsecase SaveDataUsecase,
	updatePasswordUsecase UpdatePasswordUsecase,
) *Handlers {
	return &Handlers{
		JWTService:            jwtService,
		ListDataUsecase:       listDataUsecase,
		LoginUsecase:          loginUsecase,
		ReadDataUsecase:       readDataUsecase,
		RecoverUsecase:        recoverUsecase,
		RegisterUsecase:       registerUsecase,
		SaveDataUsecase:       saveDataUsecase,
		UpdatePasswordUsecase: updatePasswordUsecase,
	}
}

func (h *Handlers) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, keys, err := h.RegisterUsecase.Handle(ctx, in.Username, in.Password)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrLoginTaken):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		case errors.Is(err, domainmodels.ErrPasswordComplexity) || errors.Is(err, domainmodels.ErrLoginValidation):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := h.JWTService.GenerateToken(user.UUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.RegisterResponse{
		NewJwtToken:   token,
		DekCiphertext: keys.DEKCiphertext,
		KekSalt:       keys.KEKSalt,
		RecoveryKey:   keys.RecoveryKey,
		UserUuid:      user.UUID,
	}, nil
}

func (h *Handlers) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, keys, err := h.LoginUsecase.Handle(ctx, in.Username, in.Password)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := h.JWTService.GenerateToken(user.UUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.LoginResponse{
		NewJwtToken:   token,
		DekCiphertext: keys.DEKCiphertext,
		KekSalt:       keys.KEKSalt,
		UserUuid:      user.UUID,
	}, nil
}

func (h *Handlers) Recovery(ctx context.Context, in *pb.RecoveryRequest) (*pb.RecoveryResponse, error) {
	user, keys, err := h.RecoverUsecase.Handle(ctx, in.Username, in.RecoveryKey, in.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, domainmodels.ErrPasswordComplexity):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	token, err := h.JWTService.GenerateToken(user.UUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.RecoveryResponse{
		NewJwtToken:   token,
		DekCiphertext: keys.DEKCiphertext,
		KekSalt:       keys.KEKSalt,
		RecoveryKey:   keys.RecoveryKey,
		UserUuid:      user.UUID,
	}, nil
}

func (h *Handlers) UpdatePassword(ctx context.Context, in *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	userUUID, authenticated := h.JWTService.ProcessToken(in.JwtToken)
	if !authenticated {
		return nil, status.Error(codes.Unauthenticated, "login first or use recovery key")
	}

	user, keys, err := h.UpdatePasswordUsecase.Handle(ctx, userUUID, in.OldPassword, in.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, domainmodels.ErrWrongCredentials):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, domainmodels.ErrPasswordComplexity):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	token, err := h.JWTService.GenerateToken(user.UUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.UpdatePasswordResponse{
		NewJwtToken:   token,
		DekCiphertext: keys.DEKCiphertext,
		KekSalt:       keys.KEKSalt,
		RecoveryKey:   keys.RecoveryKey,
		UserUuid:      user.UUID,
	}, nil
}

func (h *Handlers) SaveData(ctx context.Context, in *pb.SaveDataRequest) (*pb.SaveDataResponse, error) {
	userUUID, authenticated := h.JWTService.ProcessToken(in.JwtToken)
	if !authenticated {
		return nil, status.Error(codes.Unauthenticated, "login first or use recovery key")
	}
	var meta *domainmodels.DataInfo
	if in.Meta != nil {
		if userUUID != in.Meta.OwnerUuid {
			return nil, status.Error(codes.PermissionDenied, "Meta.OwnerUuid not matches JWT token")
		}
		t, parseErr := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", in.Meta.LastUpdated)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "cannot parse time from Meta.LastUpdated")
		}
		meta = &domainmodels.DataInfo{
			UUID:        in.Meta.Uuid,
			OwnerUUID:   in.Meta.OwnerUuid,
			LastUpdated: t,
			IsDeleted:   in.Meta.IsDeleted,
		}
	}
	meta, err := h.SaveDataUsecase.Handle(ctx, userUUID, meta, in.EncryptedData)
	if err != nil {
		if errors.Is(err, domainmodels.ErrDataDeleted) || errors.Is(err, domainmodels.ErrDataConflict) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := h.JWTService.GenerateToken(userUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.SaveDataResponse{
		NewJwtToken: token,
		Meta: &pb.DataMetaInfo{
			Uuid:        meta.UUID,
			OwnerUuid:   meta.OwnerUUID,
			LastUpdated: meta.LastUpdated.String(),
			IsDeleted:   meta.IsDeleted,
		},
	}, nil
}

func (h *Handlers) ReadData(ctx context.Context, in *pb.ReadDataRequest) (*pb.ReadDataResponse, error) {
	userUUID, authenticated := h.JWTService.ProcessToken(in.JwtToken)
	if !authenticated {
		return nil, status.Error(codes.Unauthenticated, "login first or use recovery key")
	}
	meta, data, err := h.ReadDataUsecase.Handle(ctx, userUUID, in.DataUuid)
	if err != nil && !errors.Is(err, domainmodels.ErrDataDeleted) {
		if errors.Is(err, domainmodels.ErrDataInfoNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := h.JWTService.GenerateToken(userUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ReadDataResponse{
		NewJwtToken: token,
		Meta: &pb.DataMetaInfo{
			Uuid:        meta.UUID,
			OwnerUuid:   meta.OwnerUUID,
			LastUpdated: meta.LastUpdated.String(),
			IsDeleted:   meta.IsDeleted,
		},
		EncryptedData: data,
	}, nil
}

func (h *Handlers) ListData(ctx context.Context, in *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	userUUID, authenticated := h.JWTService.ProcessToken(in.JwtToken)
	if !authenticated {
		return nil, status.Error(codes.Unauthenticated, "login first or use recovery key")
	}
	metas, err := h.ListDataUsecase.Handle(ctx, userUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, err := h.JWTService.GenerateToken(userUUID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	apiMetas := make([]*pb.DataMetaInfo, 0, len(metas))
	for _, meta := range metas {
		apiMetas = append(apiMetas, &pb.DataMetaInfo{
			Uuid:        meta.UUID,
			OwnerUuid:   meta.OwnerUUID,
			LastUpdated: meta.LastUpdated.String(),
			IsDeleted:   meta.IsDeleted,
		})
	}
	return &pb.ListDataResponse{
		NewJwtToken: token,
		Metas:       apiMetas,
	}, nil
}
