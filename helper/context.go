package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
)

// UserLocalsKey adalah key c.Locals tempat middleware menyimpan model.AuthUser.
const UserLocalsKey = "auth_user"

const requestTimeout = 5 * time.Second

// CurrentUser mengambil user yang sudah diautentikasi oleh middleware.
func CurrentUser(c *fiber.Ctx) (model.AuthUser, bool) {
	user, ok := c.Locals(UserLocalsKey).(model.AuthUser)
	return user, ok
}

// ParamID membaca path param sebagai integer positif.
func ParamID(c *fiber.Ctx, name string) (int, bool) {
	id, err := strconv.Atoi(c.Params(name))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// RequestContext membuat context dengan timeout 5 detik untuk satu request.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), requestTimeout)
}
