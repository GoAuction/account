package grpc

import (
	"account/domain"
	accountv1 "account/proto/gen"
	"context"
	"database/sql"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type AccountServiceServer struct {
	accountv1.UnimplementedAccountServiceServer
	db *gorm.DB
}

func NewAccountServiceServer(db *gorm.DB) *AccountServiceServer {
	return &AccountServiceServer{
		db: db,
	}
}

func (s *AccountServiceServer) GetProfile(ctx context.Context, req *accountv1.GetProfileRequest) (*accountv1.GetProfileResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var profile domain.Profile
	err := s.db.Where("user_id = ?", req.UserId).First(&profile).Error
	if err != nil {
		return nil, s.mapError(err)
	}

	return &accountv1.GetProfileResponse{
		Id:            profile.ID.String(),
		UserId:        profile.UserID.String(),
		Name:          profile.Name,
		Phone:         profile.Phone,
		Country:       profile.Country,
		City:          profile.City,
		Description:   profile.Description,
		PhoneVerified: profile.PhoneVerified,
		EmailVerified: profile.EmailVerified,
		CreatedAt:     timestamppb.New(profile.CreatedAt),
		UpdatedAt:     timestamppb.New(profile.UpdatedAt),
	}, nil
}

func (s *AccountServiceServer) mapError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return status.Error(codes.NotFound, "profile not found")
	}
	return status.Error(codes.Internal, "internal error")
}
