package queue

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"

	"ruangku-backend/app/model"
)

const (
	// TaskTypeBookingStatus adalah tipe task notifikasi perubahan status booking.
	TaskTypeBookingStatus = "notification:booking_status"

	maxRetry    = 5
	taskTimeout = 30 * time.Second
)

// ErrInvalidEvent berarti event tidak bisa dijadikan notifikasi (status tidak
// dikenal, atau id kosong). Mengulang task seperti ini tidak akan berhasil.
var ErrInvalidEvent = errors.New("event status booking tidak valid")

// TaskID membuat Task ID unik per booking dan status. Satu booking hanya
// mencapai satu status satu kali, jadi ID ini tidak pernah bentrok sah.
func TaskID(bookingID int, status string) string {
	return fmt.Sprintf("booking-%d-%s", bookingID, status)
}

// newBookingStatusTask memetakan event ke task Asynq beserta opsinya.
func newBookingStatusTask(e model.BookingStatusEvent, cfg Config) (*asynq.Task, []asynq.Option, error) {
	if e.BookingID < 1 || e.OwnerID < 1 || !knownStatus(e.Status) {
		return nil, nil, ErrInvalidEvent
	}
	payload, err := json.Marshal(e)
	if err != nil {
		return nil, nil, fmt.Errorf("encode event booking %d: %w", e.BookingID, err)
	}

	queueName := cfg.Queue
	if queueName == "" {
		queueName = DefaultQueue
	}
	opts := []asynq.Option{
		asynq.TaskID(TaskID(e.BookingID, e.Status)),
		asynq.Queue(queueName),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(taskTimeout),
	}
	return asynq.NewTask(TaskTypeBookingStatus, payload), opts, nil
}

func knownStatus(status string) bool {
	switch status {
	case model.StatusPending, model.StatusApproved, model.StatusRejected, model.StatusCancelled:
		return true
	}
	return false
}

// bookingStatusText membuat judul dan isi notifikasi (bahasa Indonesia,
// teks untuk pengguna) dari status tujuan.
func bookingStatusText(status string, bookingID int) (title, message string, err error) {
	switch status {
	case model.StatusPending:
		return "Peminjaman diajukan", fmt.Sprintf("Peminjaman #%d berhasil diajukan dan menunggu persetujuan.", bookingID), nil
	case model.StatusApproved:
		return "Peminjaman disetujui", fmt.Sprintf("Peminjaman #%d telah disetujui.", bookingID), nil
	case model.StatusRejected:
		return "Peminjaman ditolak", fmt.Sprintf("Peminjaman #%d ditolak oleh pengelola.", bookingID), nil
	case model.StatusCancelled:
		return "Peminjaman dibatalkan", fmt.Sprintf("Peminjaman #%d telah dibatalkan.", bookingID), nil
	}
	return "", "", ErrInvalidEvent
}
