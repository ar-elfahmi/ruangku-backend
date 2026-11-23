package service

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
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

// NotificationEnqueuer mengirim event perubahan status ke antrean. Hanya boleh
// dipanggil SETELAH transaksi PostgreSQL commit, tidak pernah di dalamnya.
type NotificationEnqueuer interface {
	EnqueueBookingStatus(ctx context.Context, e model.BookingStatusEvent) error
}

// NotificationService: /notifications.
type NotificationService interface {
	List(c *fiber.Ctx) error        // GET   /notifications              200 / 400 / 401
	UnreadCount(c *fiber.Ctx) error // GET   /notifications/unread-count 200 / 401
	MarkRead(c *fiber.Ctx) error    // PATCH /notifications/:id/read     200 / 400 / 401 / 404
	MarkAllRead(c *fiber.Ctx) error // PATCH /notifications/read-all     200 / 401
}