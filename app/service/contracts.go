package service

import (
	"context"

	"github.com/gofiber/fiber/v2"
)

// AuthService: POST /auth/register, POST /auth/login, GET /auth/me.
type AuthService interface {
	Register(c *fiber.Ctx) error // 201 / 409 / 422
	Login(c *fiber.Ctx) error    // 200 TokenResponse / 401 / 422
	Me(c *fiber.Ctx) error       // 200 / 401
	// EnsureAdmin dipanggil sekali dari main.go. Bukan route.
	EnsureAdmin(ctx context.Context, name, email, password string) error
}

// UserService: PATCH /users/:id/role (admin).
type UserService interface {
	ChangeRole(c *fiber.Ctx) error // 200 / 400 / 404 / 422
}

// RoomService: /rooms.
type RoomService interface {
	List(c *fiber.Ctx) error         // GET    /rooms                   200
	Get(c *fiber.Ctx) error          // GET    /rooms/:id               200 / 400 / 404
	Create(c *fiber.Ctx) error       // POST   /rooms                   201 / 409 / 422
	Update(c *fiber.Ctx) error       // PUT    /rooms/:id               200 / 400 / 404 / 409 / 422
	Delete(c *fiber.Ctx) error       // DELETE /rooms/:id               204 / 404
	Availability(c *fiber.Ctx) error // GET    /rooms/:id/availability  200 / 400 / 404
}

// BookingService: siklus hidup peminjaman.
type BookingService interface {
	Create(c *fiber.Ctx) error  // POST  /bookings              201 / 400 / 404 / 409 / 422
	Cancel(c *fiber.Ctx) error  // PATCH /bookings/:id/cancel   200 / 403 / 404 / 409
	Approve(c *fiber.Ctx) error // PATCH /bookings/:id/approve  200 / 403 / 404 / 409
	Reject(c *fiber.Ctx) error  // PATCH /bookings/:id/reject   200 / 403 / 404 / 409
	Logs(c *fiber.Ctx) error    // GET   /bookings/:id/logs     200 / 403 / 404
}

// HistoryService: riwayat peminjaman.
type HistoryService interface {
	Mine(c *fiber.Ctx) error    // GET /bookings/mine?status=        200 / 400
	ListAll(c *fiber.Ctx) error // GET /bookings?status=&room_id=    200 / 400
}
