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

	// maxNotificationList adalah batas jumlah notifikasi dalam satu daftar.
	maxNotificationList = 100

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
	if limit < 1 || limit > maxNotificationList {
		limit = maxNotificationList
	}

	filter := bson.D{{Key: "user_id", Value: userID}}
	if onlyUnread {
		filter = append(filter, bson.E{Key: "is_read", Value: false})
	}

	cursor, err := r.collection().Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetLimit(int64(limit)))
	if err != nil {
		return nil, TranslateMongoError(err)
	}
	defer func() { _ = cursor.Close(ctx) }()

	var docs []notificationDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, TranslateMongoError(err)
	}

	notifications := make([]model.Notification, 0, len(docs))
	for _, doc := range docs {
		notifications = append(notifications, doc.toModel())
	}
	return notifications, nil
}

func (r *notificationRepository) CountUnread(ctx context.Context, userID int) (int, error) {
	count, err := r.collection().CountDocuments(ctx, bson.D{
		{Key: "user_id", Value: userID},
		{Key: "is_read", Value: false},
	})
	if err != nil {
		return 0, TranslateMongoError(err)
	}
	return int(count), nil
}

func (r *notificationRepository) UpdateRead(ctx context.Context, userID int, id string) (model.Notification, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return model.Notification{}, ErrNotFound
	}

	// read_at hanya diisi sekali, jadi pemanggilan berulang tetap idempotent.
	update := mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "is_read", Value: true},
			{Key: "read_at", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$read_at", "$$NOW"}}}},
		}}},
	}

	var doc notificationDoc
	err = r.collection().FindOneAndUpdate(
		ctx,
		bson.D{{Key: "_id", Value: objectID}, {Key: "user_id", Value: userID}},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		return model.Notification{}, TranslateMongoError(err)
	}
	return doc.toModel(), nil
}

func (r *notificationRepository) UpdateAllRead(ctx context.Context, userID int) (int, error) {
	return 0, ErrNotImplemented
}

func (r *notificationRepository) collection() *mongo.Collection {
	return r.db.Collection(notificationCollection)
}

var _ NotificationRepository = (*notificationRepository)(nil)
