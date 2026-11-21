package queue

import (
	"context"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/app/service"
)

// Producer mengirim task ke Redis. Close dipanggil saat API berhenti.
type Producer interface {
	service.NotificationEnqueuer
	Close() error
}

type producer struct {
	cfg Config
}

// NewProducer belum terhubung ke Redis; implementasi menyusul.
func NewProducer(cfg Config) Producer {
	return &producer{cfg: cfg}
}

// EnqueueBookingStatus sengaja mengembalikan error, bukan nil: skeleton tidak
// boleh terlihat sukses padahal tidak ada task yang masuk antrean.
func (p *producer) EnqueueBookingStatus(ctx context.Context, e model.BookingStatusEvent) error {
	return repository.ErrNotImplemented
}

func (p *producer) Close() error {
	return nil
}

var _ Producer = (*producer)(nil)
