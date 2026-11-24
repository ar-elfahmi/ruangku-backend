package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
)

// NotificationStore adalah satu-satunya hal yang dibutuhkan worker dari
// penyimpanan. repository.NotificationRepository memenuhinya.
type NotificationStore interface {
	InsertNotification(ctx context.Context, n model.Notification) error // repository.ErrDuplicate bila (booking_id, status) sudah ada
}

type processor struct {
	store NotificationStore
}

// ProcessBookingStatus membuat notifikasi dari task lalu menyimpannya.
//
//   - Berhasil, atau notifikasi itu sudah ada (ErrDuplicate): nil.
//   - Payload rusak atau status tidak dikenal: SkipRetry. Mengulang tidak
//     akan memperbaikinya, jadi task langsung diarsipkan.
//   - Kegagalan lain (misalnya MongoDB mati): error biasa, Asynq mengulang
//     dengan backoff sampai batas retry lalu mengarsipkannya.
func (p *processor) ProcessBookingStatus(ctx context.Context, task *asynq.Task) error {
	var e model.BookingStatusEvent
	if err := json.Unmarshal(task.Payload(), &e); err != nil {
		return fmt.Errorf("payload task rusak: %w: %w", err, asynq.SkipRetry)
	}
	title, message, err := bookingStatusText(e.Status, e.BookingID)
	if err != nil || e.BookingID < 1 || e.OwnerID < 1 {
		return fmt.Errorf("event booking %d (%s) tidak valid: %w: %w", e.BookingID, e.Status, ErrInvalidEvent, asynq.SkipRetry)
	}

	err = p.store.InsertNotification(ctx, model.Notification{
		UserID:    e.OwnerID,
		BookingID: e.BookingID,
		Status:    e.Status,
		Title:     title,
		Message:   message,
		CreatedAt: e.OccurredAt,
	})
	if err != nil && !errors.Is(err, repository.ErrDuplicate) {
		return fmt.Errorf("simpan notifikasi booking %d (%s): %w", e.BookingID, e.Status, err)
	}
	return nil
}
