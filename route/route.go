package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"ruangku-backend/app/model"
	"ruangku-backend/app/service"
	"ruangku-backend/helper"
	"ruangku-backend/middleware"
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

	registerAuth(api, d)
	registerRooms(api, d)
	registerBookings(api, d)
}

func registerAuth(api fiber.Router, d Dependencies) {
	requireAuth := middleware.RequireAuth(d.JWT)

	auth := api.Group("/auth")
	auth.Post("/register", d.Auth.Register)
	auth.Post("/login", d.Auth.Login)
	auth.Get("/me", requireAuth, d.Auth.Me)

	api.Patch("/users/:id/role", requireAuth, middleware.RequireRole(model.RoleAdmin), d.User.ChangeRole)
}

func registerRooms(api fiber.Router, d Dependencies) {
	requireAuth := middleware.RequireAuth(d.JWT)
	requireAdmin := middleware.RequireRole(model.RoleAdmin)

	rooms := api.Group("/rooms", requireAuth)
	rooms.Get("/", d.Room.List)
	rooms.Get("/:id", d.Room.Get)
	rooms.Get("/:id/availability", d.Room.Availability)
	rooms.Post("/", requireAdmin, d.Room.Create)
	rooms.Put("/:id", requireAdmin, d.Room.Update)
	rooms.Delete("/:id", requireAdmin, d.Room.Delete)
}

func registerBookings(api fiber.Router, d Dependencies) {
	requireAuth := middleware.RequireAuth(d.JWT)
	requirePeminjam := middleware.RequireRole(model.RolePeminjam)
	requirePengelola := middleware.RequireRole(model.RolePengelola)

	bookings := api.Group("/bookings", requireAuth)
	// /mine harus didaftarkan sebelum route /:id apa pun.
	bookings.Get("/mine", requirePeminjam, d.History.Mine)
	bookings.Get("/", requirePengelola, d.History.ListAll)
	bookings.Post("/", requirePeminjam, d.Booking.Create)
	bookings.Patch("/:id/cancel", requirePeminjam, d.Booking.Cancel)
	bookings.Patch("/:id/approve", requirePengelola, d.Booking.Approve)
	bookings.Patch("/:id/reject", requirePengelola, d.Booking.Reject)
	bookings.Get("/:id/logs", d.Booking.Logs)
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
