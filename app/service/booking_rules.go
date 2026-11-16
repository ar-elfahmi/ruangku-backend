package service

import (
	"errors"
	"time"

	"ruangku-backend/app/model"
)

var (
	ErrInvalidWindow = errors.New("waktu selesai harus setelah waktu mulai")
	ErrPastBooking   = errors.New("waktu mulai harus di masa depan")
)

// ValidateBookingWindow memastikan start < end dan start berada setelah now.
// now dikirim dari luar agar fungsi ini tetap murni dan mudah dites.
func ValidateBookingWindow(start, end, now time.Time) error {
	if !start.Before(end) {
		return ErrInvalidWindow
	}
	if !start.After(now) {
		return ErrPastBooking
	}
	return nil
}

// CanTransition hanya mengizinkan perpindahan status yang tercantum di
// system-requirements §8.3.
func CanTransition(from, to, actorRole string, isOwner bool) bool {
	switch {
	case from == model.StatusPending && (to == model.StatusApproved || to == model.StatusRejected):
		return actorRole == model.RolePengelola
	case (from == model.StatusPending || from == model.StatusApproved) && to == model.StatusCancelled:
		return actorRole == model.RolePeminjam && isOwner
	}
	return false
}

// CanActOn menjawab "apakah peran ini boleh mencoba aksi ke status target?",
// terpisah dari CanTransition agar peran/kepemilikan salah menjadi 403,
// sedangkan status yang tidak cocok menjadi 409.
func CanActOn(target, actorRole string, isOwner bool) bool {
	if target == model.StatusCancelled {
		return actorRole == model.RolePeminjam && isOwner
	}
	return actorRole == model.RolePengelola
}
