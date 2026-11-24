package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"ruangku-backend/app/model"
)

const notificationCollection = "notifications"

// notificationDoc adalah bentuk dokumen di MongoDB. ObjectID dan zona waktu
// dikonversi ke bentuk API oleh toModel.
type notificationDoc struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    int           `bson:"user_id"`
	BookingID int           `bson:"booking_id"`
	Status    string        `bson:"status"`
	Title     string        `bson:"title"`
	Message   string        `bson:"message"`
	IsRead    bool          `bson:"is_read"`
	CreatedAt time.Time     `bson:"created_at"`
	ReadAt    *time.Time    `bson:"read_at,omitempty"`
}

func (d notificationDoc) toModel() model.Notification {
	n := model.Notification{
		ID:        d.ID.Hex(),
		UserID:    d.UserID,
		BookingID: d.BookingID,
		Status:    d.Status,
		Title:     d.Title,
		Message:   d.Message,
		IsRead:    d.IsRead,
		CreatedAt: d.CreatedAt.In(time.Local),
	}
	if d.ReadAt != nil {
		readAt := d.ReadAt.In(time.Local)
		n.ReadAt = &readAt
	}
	return n
}

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
