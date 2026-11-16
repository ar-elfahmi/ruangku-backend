package repository

import (
	"context"
	"time"

	"ruangku-backend/app/model"
)

type roomRepository struct {
	db DBTX
}

func NewRoomRepository(db DBTX) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) FindAllActive(ctx context.Context) ([]model.Room, error) {
	return nil, ErrNotImplemented
}

func (r *roomRepository) FindByID(ctx context.Context, id int) (model.Room, error) {
	return model.Room{}, ErrNotImplemented
}

func (r *roomRepository) Create(ctx context.Context, room model.Room) (model.Room, error) {
	return model.Room{}, ErrNotImplemented
}

func (r *roomRepository) Update(ctx context.Context, room model.Room) (model.Room, error) {
	return model.Room{}, ErrNotImplemented
}

func (r *roomRepository) SoftDelete(ctx context.Context, id int) error {
	return ErrNotImplemented
}

func (r *roomRepository) FindBookedSlots(ctx context.Context, roomID int, dayStart, dayEnd time.Time) ([]model.TimeSlot, error) {
	return nil, ErrNotImplemented
}

var _ RoomRepository = (*roomRepository)(nil)
