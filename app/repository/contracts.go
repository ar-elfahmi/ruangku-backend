package repository

import (
	"context"
	"time"

	"ruangku-backend/app/model"
)

type UserRepository interface {
	Create(ctx context.Context, u model.User) (model.User, error)            // ErrDuplicate
	FindByEmail(ctx context.Context, email string) (model.User, error)       // ErrNotFound
	FindByID(ctx context.Context, id int) (model.User, error)                // ErrNotFound
	UpdateRole(ctx context.Context, id int, role string) (model.User, error) // ErrNotFound
}

type RoomRepository interface {
	FindAllActive(ctx context.Context) ([]model.Room, error)
	FindByID(ctx context.Context, id int) (model.Room, error)     // termasuk ruang nonaktif; ErrNotFound
	Create(ctx context.Context, r model.Room) (model.Room, error) // ErrDuplicate
	Update(ctx context.Context, r model.Room) (model.Room, error) // ErrNotFound, ErrDuplicate
	SoftDelete(ctx context.Context, id int) error                 // ErrNotFound
	FindBookedSlots(ctx context.Context, roomID int, dayStart, dayEnd time.Time) ([]model.TimeSlot, error)
}

type BookingRepository interface {
	WithTx(tx DBTX) BookingRepository
	Create(ctx context.Context, b model.Booking) (model.Booking, error)               // ErrOverlap
	FindByID(ctx context.Context, id int) (model.Booking, error)                      // ErrNotFound
	UpdateStatus(ctx context.Context, id int, from, to string) (model.Booking, error) // ErrConflict bila 0 baris
	InsertLog(ctx context.Context, l model.BookingLog) error
	FindLogs(ctx context.Context, bookingID int) ([]model.BookingLog, error)
}

type HistoryRepository interface {
	FindBookings(ctx context.Context, f model.HistoryFilter) ([]model.BookingDetail, error)
}

// placeholder for future notification repository
type NotificationRepository interface {

}