package model

import "time"

type Room struct {
	ID        int       `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Building  string    `json:"building"`
	Capacity  int       `json:"capacity"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type RoomRequest struct {
	Code     string `json:"code" validate:"required,max=20"`
	Name     string `json:"name" validate:"required,max=100"`
	Building string `json:"building" validate:"required,max=100"`
	Capacity int    `json:"capacity" validate:"required,min=1,max=1000"`
}

// TimeSlot adalah satu booking aktif (pending/approved) pada sebuah ruang.
type TimeSlot struct {
	BookingID int       `json:"booking_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

type Availability struct {
	Room   Room       `json:"room"`
	Date   string     `json:"date"`
	Booked []TimeSlot `json:"booked"`
}
