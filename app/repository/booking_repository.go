package repository

import (
	"context"

	"ruangku-backend/app/model"
)

type bookingRepository struct {
	db DBTX
}

func NewBookingRepository(db DBTX) BookingRepository {
	return &bookingRepository{db: db}
}

// WithTx mengembalikan salinan repository yang berjalan di dalam transaksi tx.
func (r *bookingRepository) WithTx(tx DBTX) BookingRepository {
	return &bookingRepository{db: tx}
}

func (r *bookingRepository) Create(ctx context.Context, b model.Booking) (model.Booking, error) {
	return model.Booking{}, ErrNotImplemented
}

func (r *bookingRepository) FindByID(ctx context.Context, id int) (model.Booking, error) {
	return model.Booking{}, ErrNotImplemented
}

func (r *bookingRepository) UpdateStatus(ctx context.Context, id int, from, to string) (model.Booking, error) {
	return model.Booking{}, ErrNotImplemented
}

func (r *bookingRepository) InsertLog(ctx context.Context, l model.BookingLog) error {
	return ErrNotImplemented
}

func (r *bookingRepository) FindLogs(ctx context.Context, bookingID int) ([]model.BookingLog, error) {
	return nil, ErrNotImplemented
}

var _ BookingRepository = (*bookingRepository)(nil)
