package model

import "time"

const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
)

type Booking struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	RoomID    int       `json:"room_id"`
	Purpose   string    `json:"purpose"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// BookingLog mencatat setiap perubahan status. FromStatus nil untuk pembuatan booking.
type BookingLog struct {
	ID         int       `json:"id"`
	BookingID  int       `json:"booking_id"`
	ActorID    int       `json:"actor_id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateBookingRequest struct {
	RoomID    int       `json:"room_id" validate:"required,min=1"`
	Purpose   string    `json:"purpose" validate:"required,max=255"`
	StartTime time.Time `json:"start_time" validate:"required"`
	EndTime   time.Time `json:"end_time" validate:"required"`
}

type StatusChangeRequest struct {
	Note string `json:"note" validate:"max=255"`
}
