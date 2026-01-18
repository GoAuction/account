package app

import (
	"account/domain"
	"account/pkg/events"
	"account/pkg/httperror"
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GetProfileRequest struct {
	UserID uuid.UUID `params:"id"`
}

type GetProfileResponse struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	Name           string    `json:"name"`
	Phone          string    `json:"phone"`
	Country        string    `json:"country"`
	City           string    `json:"city"`
	Description    string    `json:"description"`
	Photo          string    `json:"photo"`
	PhoneVerified  bool      `json:"phone_verified"`
	EmailVerified  bool      `json:"email_verified"`
	Verified       bool      `json:"verified"`
	Business       bool      `json:"business"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Rating         float64   `json:"rating"`
	TotalSales     int       `json:"total_sales"`
	ActiveListings int       `json:"active_listings"`
	ResponseRate   float64   `json:"response_rate"`
	ResponseTime   float64   `json:"response_time"`
}

type GetProfileHandler struct {
	db             *gorm.DB
	eventPublisher events.Publisher
}

func NewGetProfileHandler(db *gorm.DB, eventPublisher events.Publisher) *GetProfileHandler {
	return &GetProfileHandler{
		db:             db,
		eventPublisher: eventPublisher,
	}
}

func (r GetProfileHandler) Handle(ctx context.Context, req *GetProfileRequest) (*GetProfileResponse, error) {
	profile, err := gorm.G[domain.Profile](r.db).Where("user_id = ?", req.UserID).First(ctx)
	if err != nil {
		return nil, httperror.NotFound("account.profile.show.not_found", "Profile not found", nil)
	}

	return &GetProfileResponse{
		ID:             profile.ID,
		UserID:         profile.UserID,
		Name:           profile.Name,
		Phone:          profile.Phone,
		Country:        profile.Country,
		City:           profile.City,
		Description:    profile.Description,
		PhoneVerified:  profile.PhoneVerified,
		EmailVerified:  profile.EmailVerified,
		Verified:       profile.Verified,
		Business:       profile.Business,
		CreatedAt:      profile.CreatedAt,
		UpdatedAt:      profile.UpdatedAt,
		Rating:         3.2,
		TotalSales:     10,
		ActiveListings: 5,
		ResponseRate:   90,
		ResponseTime:   2,
		Photo:          profile.Photo,
	}, nil
}
