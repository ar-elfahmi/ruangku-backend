package model

import "time"

// Notification adalah notifikasi untuk pemilik booking, disimpan di MongoDB.
type Notification struct {
	ID        string     `json:"id"`      // ObjectID hex
	UserID    int        `json:"user_id"` // pemilik booking
	BookingID int        `json:"booking_id"`
	Status    string     `json:"status"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at"`
}
