package middleware

import (
	"account/pkg/httperror"
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func NewSecurityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userIDStr := strings.TrimSpace(c.Get("User-ID"))
		userEmail := strings.TrimSpace(c.Get("User-Email"))
		authorization := strings.TrimSpace(c.Get("Authorization"))

		if userIDStr == "" || userEmail == "" || authorization == "" {
			return unauthorized(c)
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return unauthorized(c)
		}

		userCtx := c.UserContext()
		if userCtx == nil {
			userCtx = context.Background()
		}

		userCtx = context.WithValue(userCtx, "UserID", userID)
		userCtx = context.WithValue(userCtx, "UserEmail", userEmail)
		userCtx = context.WithValue(userCtx, "Jwt", authorization)

		c.SetUserContext(userCtx)
		return c.Next()
	}
}

func unauthorized(c *fiber.Ctx) error {
	err := httperror.Unauthorized(
		"bid.security_headers.unauthorized",
		"Security headers mismatch",
		nil,
	)

	return c.Status(err.Status).JSON(fiber.Map{
		"code":    err.Code,
		"message": err.Message,
	})
}
