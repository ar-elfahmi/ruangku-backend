package model

import "time"

// BookingStatusEvent adalah payload task: cukup untuk membuat notifikasi
// tanpa membaca PostgreSQL lagi.
type BookingStatusEvent struct {
	BookingID  int       `json:"booking_id"`
	OwnerID    int       `json:"owner_id"` // bookings.user_id
	Status     string    `json:"status"`   // status tujuan: pending|approved|rejected|cancelled
	OccurredAt time.Time `json:"occurred_at"`
}
