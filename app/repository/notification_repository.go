package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"ruangku-backend/app/model"
)

const notificationCollection = "notifications"

type notificationRepository struct {
	db *mongo.Database
}

// NewNotificationRepository membuat repository notifikasi. Index unik
// (booking_id, status) dan index daftar per user dipasang secara idempotent
// saat konstruksi pada C2.
func NewNotificationRepository(db *mongo.Database) (NotificationRepository, error) {
	return &notificationRepository{db: db}, nil
}

func (r *notificationRepository) InsertNotification(ctx context.Context, n model.Notification) error {
	return ErrNotImplemented
}

func (r *notificationRepository) FindByUser(ctx context.Context, userID int, onlyUnread bool, limit int) ([]model.Notification, error) {
	return nil, ErrNotImplemented
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID int) (int, error) {
	return 0, ErrNotImplemented
}

func (r *notificationRepository) UpdateRead(ctx context.Context, userID int, id string) (model.Notification, error) {
	return model.Notification{}, ErrNotImplemented
}

func (r *notificationRepository) UpdateAllRead(ctx context.Context, userID int) (int, error) {
	return 0, ErrNotImplemented
}

var _ NotificationRepository = (*notificationRepository)(nil)