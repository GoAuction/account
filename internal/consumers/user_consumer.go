package consumers

import (
	"account/domain"
	"account/pkg/events"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type UserEventHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewUserEventHandler(db *gorm.DB, logger *zap.Logger) *UserEventHandler {
	return &UserEventHandler{
		db:     db,
		logger: logger,
	}
}

func (h *UserEventHandler) HandleEvent(ctx context.Context, event *events.Event) error {
	h.logger.Info("User event received",
		zap.String("event", event.Event),
		zap.String("version", event.Version),
		zap.String("traceId", event.TraceID),
		zap.Any("payload", event.Payload),
	)

	switch event.Event {
	case "identity.user.registered":
		return h.handleUserRegistered(ctx, event)
	default:
		h.logger.Warn("Unknown user event type", zap.String("event", event.Event))
		return nil
	}
}

func (h *UserEventHandler) handleUserRegistered(ctx context.Context, event *events.Event) error {
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("malformed payload - marshal failed: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("malformed payload - unmarshal failed: %w", err)
	}

	userID, ok := payload["UserID"].(string)
	if !ok || userID == "" {
		return fmt.Errorf("malformed payload - userId missing or invalid")
	}

	name, ok := payload["Name"].(string)
	if !ok || name == "" {
		return fmt.Errorf("malformed payload - name missing or invalid")
	}

	email, ok := payload["Email"].(string)
	if !ok || email == "" {
		return fmt.Errorf("malformed payload - email missing or invalid")
	}

	bidTime := event.Timestamp
	if bidTimeStr, ok := payload["Timestamp"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, bidTimeStr); err == nil {
			bidTime = parsedTime
		}
	}

	zap.L().Info("Processing identity.user.registered event",
		zap.String("userId", userID),
		zap.String("name", name),
		zap.String("email", email),
		zap.Time("bidTime", bidTime),
		zap.String("traceId", event.TraceID),
	)

	profile, err := gorm.G[domain.Profile](h.db).Where("user_id = ?", userID).First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newProfile := domain.Profile{
				ID:            uuid.New(),
				UserID:        uuid.MustParse(userID),
				Name:          name,
				PhoneVerified: false,
				EmailVerified: false,
			}

			if err := gorm.G[domain.Profile](h.db).Create(ctx, &newProfile); err != nil {
				return fmt.Errorf("failed to create profile: %w", err)
			}

			h.logger.Info("Profile created successfully",
				zap.String("userId", userID),
				zap.String("name", name),
			)
			return nil
		}
		// Başka bir hata varsa döndür
		return fmt.Errorf("failed to get profile: %w", err)
	}

	// Profile zaten varsa, bilgileri güncelle
	rows, err := gorm.G[domain.Profile](h.db).Where("id = ?", profile.ID).Updates(ctx, domain.Profile{
		Name:          name,
		PhoneVerified: false,
		EmailVerified: false,
	})

	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	if rows == 0 {
		h.logger.Warn("Profile update did not affect any rows",
			zap.String("userId", userID),
			zap.String("profileId", profile.ID.String()),
		)
	} else {
		h.logger.Info("Profile updated successfully",
			zap.String("userId", userID),
			zap.String("name", name),
		)
	}

	return nil
}
