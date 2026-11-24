//go:build integration

// Integration test repository notifikasi (Task C2 — matriks C-I1 s.d. C-I5
// bagian 11.2 dokumen Phase 2).
//
// Menjalankan (lihat bagian 8.6):
//
//  1. docker compose up -d
//  2. set TEST_* (TEST_MONGO_URI, TEST_MONGO_DB=ruangku_test, dst.)
//  3. go test -tags integration ./app/repository -count=1
//
// Tanpa variabel TEST_* test ini otomatis di-skip (testutil.Require).
package repository_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/database"
	"ruangku-backend/testutil"
)

const testTimeout = 10 * time.Second

var idSeq atomic.Int64

func init() {
	idSeq.Store(time.Now().UnixNano() % 1_000_000_000)
}

// nextID menghasilkan angka unik untuk user_id/booking_id antar test, supaya
// data test tidak saling bentrok di Mongo test yang dipakai bersama.
func nextID() int {
	return int(idSeq.Add(1))
}

func newRepository(t *testing.T) (repository.NotificationRepository, *mongo.Database) {
	t.Helper()

	env := testutil.Require(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	db, err := database.NewMongo(ctx, env.MongoURI, env.MongoDB)
	if err != nil {
		t.Fatalf("konek mongodb test: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), testTimeout)
		defer cancelCleanup()
		_ = db.Client().Disconnect(cleanupCtx)
	})

	repo, err := repository.NewNotificationRepository(db)
	if err != nil {
		t.Fatalf("buat repository notifikasi: %v", err)
	}
	return repo, db
}

func testNotification(userID, bookingID int, status string, createdAt time.Time) model.Notification {
	return model.Notification{
		UserID:    userID,
		BookingID: bookingID,
		Status:    status,
		Title:     "Peminjaman " + status,
		Message:   "Peminjaman uji " + status,
		CreatedAt: createdAt,
	}
}

func insert(t *testing.T, repo repository.NotificationRepository, n model.Notification) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := repo.InsertNotification(ctx, n); err != nil {
		t.Fatalf("insert notifikasi: %v", err)
	}
}

// C-I1
func TestInsertNotificationDuplicateAndIndexIdempotent(t *testing.T) {
	repo, db := newRepository(t)

	n := testNotification(nextID(), nextID(), "pending", time.Now())
	insert(t, repo, n)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := repo.InsertNotification(ctx, n); !errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("insert duplikat: error = %v, mau ErrDuplicate", err)
	}

	if _, err := repository.NewNotificationRepository(db); err != nil {
		t.Fatalf("membuat index dua kali: %v", err)
	}
}

// C-I2
func TestInsertNotificationConnectionFailureNotDuplicate(t *testing.T) {
	env := testutil.Require(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().
		ApplyURI(env.MongoURI).
		SetServerSelectionTimeout(2 * time.Second))
	if err != nil {
		t.Fatalf("konek mongo: %v", err)
	}

	db := client.Database(env.MongoDB)
	repo, err := repository.NewNotificationRepository(db)
	if err != nil {
		t.Fatalf("buat repository: %v", err)
	}
	if err := client.Disconnect(ctx); err != nil {
		t.Fatalf("putus koneksi: %v", err)
	}

	err = repo.InsertNotification(ctx, testNotification(nextID(), nextID(), "pending", time.Now()))
	if err == nil {
		t.Fatal("insert setelah koneksi ditutup seharusnya gagal")
	}
	if errors.Is(err, repository.ErrDuplicate) {
		t.Fatalf("gagal koneksi dipetakan ke ErrDuplicate: %v", err)
	}
}
