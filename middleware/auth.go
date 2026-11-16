package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/helper"
)

const messageInvalidToken = "token tidak valid atau sudah kedaluwarsa"

// RequireAuth menolak request tanpa Bearer token yang valid (401).
func RequireAuth(jwt *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scheme, token, found := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			return helper.Unauthorized(messageInvalidToken)
		}

		user, err := jwt.Parse(token)
		if err != nil {
			return helper.Unauthorized(messageInvalidToken)
		}

		c.Locals(helper.UserLocalsKey, user)
		return c.Next()
	}
}
