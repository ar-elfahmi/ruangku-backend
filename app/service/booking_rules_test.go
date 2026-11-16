package service

import (
	"errors"
	"testing"
	"time"

	"ruangku-backend/app/model"
)

func TestValidateBookingWindow(t *testing.T) {
	now := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	at := func(h int) time.Time { return time.Date(2026, 12, 1, h, 0, 0, 0, time.UTC) }

	tests := []struct {
		name       string
		start, end time.Time
		want       error
	}{
		{"mulai sama dengan selesai", at(10), at(10), ErrInvalidWindow},
		{"mulai setelah selesai", at(12), at(10), ErrInvalidWindow},
		{"mulai di masa lalu", now.Add(-time.Hour), now.Add(time.Hour), ErrPastBooking},
		{"mulai tepat sekarang", now, now.Add(time.Hour), ErrPastBooking},
		{"jendela valid", at(10), at(12), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateBookingWindow(tt.start, tt.end, now)
			if !errors.Is(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name     string
		from, to string
		role     string
		isOwner  bool
		want     bool
	}{
		{"pengelola menyetujui pending", model.StatusPending, model.StatusApproved, model.RolePengelola, false, true},
		{"pengelola menolak pending", model.StatusPending, model.StatusRejected, model.RolePengelola, false, true},
		{"pemilik membatalkan pending", model.StatusPending, model.StatusCancelled, model.RolePeminjam, true, true},
		{"pemilik membatalkan approved", model.StatusApproved, model.StatusCancelled, model.RolePeminjam, true, true},
		{"rejected tidak bisa disetujui", model.StatusRejected, model.StatusApproved, model.RolePengelola, false, false},
		{"cancelled tidak bisa disetujui", model.StatusCancelled, model.StatusApproved, model.RolePengelola, false, false},
		{"approved tidak bisa ditolak", model.StatusApproved, model.StatusRejected, model.RolePengelola, false, false},
		{"peminjam non-pemilik tidak bisa membatalkan", model.StatusPending, model.StatusCancelled, model.RolePeminjam, false, false},
		{"peminjam tidak bisa menyetujui", model.StatusPending, model.StatusApproved, model.RolePeminjam, true, false},
		{"pengelola tidak bisa membatalkan", model.StatusPending, model.StatusCancelled, model.RolePengelola, false, false},
		{"cancelled tidak bisa dibatalkan lagi", model.StatusCancelled, model.StatusCancelled, model.RolePeminjam, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to, tt.role, tt.isOwner); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
