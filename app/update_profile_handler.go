package app

import (
	"account/domain"
	"account/pkg/events"
	"account/pkg/httperror"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpdateProfileRequest struct {
	UserID      uuid.UUID `params:"id" validate:"required"`
	Name        string    `json:"name" validate:"required"`
	Phone       string    `json:"phone"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Description string    `json:"description"`
}

type UpdateProfileResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Phone       string    `json:"phone"`
	Country     string    `json:"country"`
	City        string    `json:"city"`
	Description string    `json:"description"`
}

type UpdateProfileHandler struct {
	db             *gorm.DB
	eventPublisher events.Publisher
}

func NewUpdateProfileHandler(db *gorm.DB, eventPublisher events.Publisher) *UpdateProfileHandler {
	return &UpdateProfileHandler{
		db:             db,
		eventPublisher: eventPublisher,
	}
}

func validateRequestParams(_ context.Context, req *UpdateProfileRequest) error {
	if req.UserID == uuid.Nil {
		return httperror.BadRequest("account.profile.update.invalid_user_id", "Invalid user ID", nil)
	}

	if req.Name == "" {
		return httperror.BadRequest("account.profile.update.invalid_name", "Invalid name", nil)
	}

	return nil
}

func (r *UpdateProfileHandler) Handle(ctx context.Context, req *UpdateProfileRequest) (*UpdateProfileResponse, error) {
	if err := validateRequestParams(ctx, req); err != nil {
		return nil, httperror.BadRequest("account.profile.update.invalid_request", "Invalid request", err)
	}

	profile, err := gorm.G[domain.Profile](r.db).Where("user_id = ?", req.UserID).First(ctx)
	if err != nil {
		return nil, httperror.NotFound("account.profile.update.not_found", "Profile not found", nil)
	}

	profile.Name = req.Name
	profile.Phone = req.Phone
	profile.Country = req.Country
	profile.City = req.City
	profile.Description = req.Description

	rows, err := gorm.G[domain.Profile](r.db).Updates(ctx, profile)
	if err != nil {
		return nil, httperror.InternalServerError("account.profile.update.failed", "Failed to update profile", err)
	}

	if rows == 0 {
		return nil, httperror.InternalServerError("account.profile.update.failed", "Failed to update profile", err)
	}

	return &UpdateProfileResponse{
		UserID:      profile.UserID,
		Name:        profile.Name,
		Phone:       profile.Phone,
		Country:     profile.Country,
		City:        profile.City,
		Description: profile.Description,
	}, nil
}
