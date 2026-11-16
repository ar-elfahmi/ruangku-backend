package repository

import (
	"context"

	"ruangku-backend/app/model"
)

type historyRepository struct {
	db DBTX
}

func NewHistoryRepository(db DBTX) HistoryRepository {
	return &historyRepository{db: db}
}

func (r *historyRepository) FindBookings(ctx context.Context, f model.HistoryFilter) ([]model.BookingDetail, error) {
	return nil, ErrNotImplemented
}

var _ HistoryRepository = (*historyRepository)(nil)
