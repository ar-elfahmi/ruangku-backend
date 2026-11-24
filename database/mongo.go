package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// mongoServerSelectionTimeout menjaga startup tidak menggantung lama saat
// MongoDB tidak bisa dihubungi.
const mongoServerSelectionTimeout = 5 * time.Second

// NewMongo membuka koneksi ke MongoDB dan memastikan server bisa dihubungi
// sebelum database dikembalikan.
func NewMongo(ctx context.Context, uri, dbName string) (*mongo.Database, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(mongoServerSelectionTimeout)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("membuat koneksi mongo: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("ping mongo: %w", err)
	}
	return client.Database(dbName), nil
}
