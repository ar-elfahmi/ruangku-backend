package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ruangku-backend/app/model"
)

const (
	notificationCollection = "notifications"

	indexCreateTimeout = 10 * time.Second
)

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

// NewNotificationRepository membuat repository notifikasi dan memasang index
// secara idempotent: unik (booking_id, status) sebagai pengaman duplikat, dan
// (user_id, created_at desc) untuk daftar per user.
func NewNotificationRepository(db *mongo.Database) (NotificationRepository, error) {
	ctx, cancel := context.WithTimeout(context.Background(), indexCreateTimeout)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "booking_id", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
	}
	if _, err := db.Collection(notificationCollection).Indexes().CreateMany(ctx, indexes); err != nil {
		return nil, TranslateMongoError(err)
	}
	return &notificationRepository{db: db}, nil
}

func (r *notificationRepository) InsertNotification(ctx context.Context, n model.Notification) error {
	_, err := r.collection().InsertOne(ctx, notificationDoc{
		UserID:    n.UserID,
		BookingID: n.BookingID,
		Status:    n.Status,
		Title:     n.Title,
		Message:   n.Message,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
		ReadAt:    n.ReadAt,
	})
	return TranslateMongoError(err)
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

func (r *notificationRepository) collection() *mongo.Collection {
	return r.db.Collection(notificationCollection)
}

var _ NotificationRepository = (*notificationRepository)(nil)
