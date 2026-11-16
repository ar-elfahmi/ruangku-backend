package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"ruangku-backend/app/model"
	"ruangku-backend/app/service"
	"ruangku-backend/helper"
)

// Dependencies berisi semua yang dibutuhkan route. Service bertipe interface
// agar route tidak bergantung pada implementasi.
type Dependencies struct {
	Pool    *pgxpool.Pool
	JWT     *helper.JWTManager
	Auth    service.AuthService
	User    service.UserService
	Room    service.RoomService
	Booking service.BookingService
	History service.HistoryService
}

// Register memasang semua route di bawah /api/v1.
func Register(app *fiber.App, d Dependencies) {
	api := app.Group("/api/v1")

	api.Get("/health", health(d.Pool))
}

func health(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := helper.RequestContext(c)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(model.WebResponse{
				Success: false,
				Message: "database tidak dapat dihubungi",
			})
		}
		return helper.Success(c, fiber.StatusOK, "service berjalan normal", fiber.Map{"database": "up"})
	}
}
