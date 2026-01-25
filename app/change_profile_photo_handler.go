package app

import (
	"account/domain"
	"account/pkg/aws"
	"account/pkg/config"
	"account/pkg/events"
	"account/pkg/httperror"
	"context"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChangeProfilePhotoRequest struct {
}

type ChangeProfilePhotoResponse struct {
}

type ChangeProfilePhotoHandler struct {
	db             *gorm.DB
	eventPublisher events.Publisher
}

func NewChangeProfilePhotoHandler(db *gorm.DB, eventPublisher events.Publisher) *ChangeProfilePhotoHandler {
	return &ChangeProfilePhotoHandler{
		db:             db,
		eventPublisher: eventPublisher,
	}
}

func (r *ChangeProfilePhotoHandler) Handle(ctx context.Context, req *ChangeProfilePhotoRequest) (*ChangeProfilePhotoResponse, error) {
	fiberCtx := ctx.Value("fiber")
	if fiberCtx == nil {
		return nil, httperror.InternalServerError("account.profile.no_context", "Fiber context not found", nil)
	}

	c, ok := fiberCtx.(*fiber.Ctx)
	if !ok {
		return nil, httperror.InternalServerError("account.profile.invalid_context", "Invalid Fiber context", nil)
	}

	userID := ctx.Value("UserID")

	profile, err := gorm.G[domain.Profile](r.db).Where("user_id = ?", userID).First(ctx)
	if err != nil {
		return nil, httperror.NotFound("account.profile.update.not_found", "Profile not found", nil)
	}

	file, err := c.FormFile("photo")
	if err != nil {
		return nil, httperror.BadRequest("account.profile.missing_file", "Image file is required (use 'image' field)", fiber.Map{"error": err.Error()})
	}

	const maxFileSize = 5 * 1024 * 1024
	if file.Size > maxFileSize {
		return nil, httperror.BadRequest("account.profile.file_too_large", "File size must not exceed 5MB",
			fiber.Map{
				"size_mb": float64(file.Size) / 1024 / 1024,
				"max_mb":  5,
			})
	}

	contentType := file.Header.Get("Content-Type")

	allowedTypes := map[string]bool{
		"image/png":  true,
		"image/jpeg": true,
		"image/jpg":  true,
	}
	if !allowedTypes[contentType] {
		return nil, httperror.BadRequest("account.profile.invalid_content_type", "Only PNG, JPEG/JPG images are allowed",
			fiber.Map{
				"received": contentType,
				"allowed":  []string{"image/png", "image/jpeg", "image/jpg"},
			})
	}

	fileReader, err := file.Open()
	if err != nil {
		return nil, httperror.InternalServerError("account.profile.file_open_error", "Failed to open uploaded file", err.Error())
	}
	defer fileReader.Close()

	fileBytes, err := io.ReadAll(fileReader)
	if err != nil {
		return nil, httperror.InternalServerError("account.profile.file_read_error", "Failed to read file content", err.Error())
	}

	extension := getExtensionFromContentType(contentType)

	key := fmt.Sprintf("user-profiles/%s/%s%s", userID, uuid.New().String(), extension)

	bucket := aws.NewS3Bucket()

	err = bucket.Upload(key, fileBytes)
	if err != nil {
		return nil, httperror.InternalServerError("account.profile.upload.failed", "Failed to upload image to storage", err.Error())
	}

	profile.Photo = constructImageURL(key)

	rows, err := gorm.G[domain.Profile](r.db).Updates(ctx, profile)
	if err != nil || rows == 0 {
		_ = bucket.Delete(key)
		return nil, httperror.InternalServerError("account.profile.update.failed", "Failed to update profile", err)
	}

	return &ChangeProfilePhotoResponse{}, nil
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "image/svg+xml":
		return ".svg"
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func constructImageURL(key string) string {
	cfg := config.Read()

	if cfg.AWSEndpoint != "" {
		return fmt.Sprintf("%s/%s/%s", cfg.AWSEndpoint, cfg.AWSBucket, key)
	}

	if cfg.AWSDefaultRegion != "" {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.AWSBucket, cfg.AWSDefaultRegion, key)
	}

	return key
}
