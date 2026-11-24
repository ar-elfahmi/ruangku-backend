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

// C-I3
func TestFindByUser(t *testing.T) {
	repo, _ := newRepository(t)

	owner, other := nextID(), nextID()
	base := time.Now().Add(-time.Hour)

	insert(t, repo, testNotification(owner, nextID(), "pending", base))
	insert(t, repo, testNotification(owner, nextID(), "approved", base.Add(time.Minute)))
	insert(t, repo, testNotification(owner, nextID(), "cancelled", base.Add(2*time.Minute)))
	insert(t, repo, testNotification(other, nextID(), "pending", base))

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	all, err := repo.FindByUser(ctx, owner, false, 100)
	if err != nil {
		t.Fatalf("FindByUser: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("jumlah notifikasi = %d, mau 3", len(all))
	}
	if all[0].Status != "cancelled" {
		t.Fatalf("urutan terbaru salah: dapat %q", all[0].Status)
	}
	for _, n := range all {
		if n.UserID != owner {
			t.Fatalf("notifikasi user lain ikut terbaca: user_id = %d", n.UserID)
		}
	}

	if _, err := repo.UpdateRead(ctx, owner, all[0].ID); err != nil {
		t.Fatalf("tandai satu sudah dibaca: %v", err)
	}

	unread, err := repo.FindByUser(ctx, owner, true, 100)
	if err != nil {
		t.Fatalf("FindByUser unread: %v", err)
	}
	if len(unread) != 2 {
		t.Fatalf("jumlah belum dibaca = %d, mau 2", len(unread))
	}

	// Limit di atas 100 tetap dipotong ke 100.
	capUser := nextID()
	for i := 0; i < 101; i++ {
		insert(t, repo, testNotification(capUser, nextID(), "pending", base.Add(time.Duration(i)*time.Second)))
	}
	limited, err := repo.FindByUser(ctx, capUser, false, 500)
	if err != nil {
		t.Fatalf("FindByUser limit: %v", err)
	}
	if len(limited) != 100 {
		t.Fatalf("jumlah dengan limit 500 = %d, mau 100", len(limited))
	}
}

// C-I4
func TestUpdateRead(t *testing.T) {
	repo, _ := newRepository(t)

	owner, other := nextID(), nextID()
	insert(t, repo, testNotification(owner, nextID(), "pending", time.Now()))

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	all, err := repo.FindByUser(ctx, owner, false, 100)
	if err != nil || len(all) != 1 {
		t.Fatalf("FindByUser: %v, jumlah %d", err, len(all))
	}
	id := all[0].ID

	read, err := repo.UpdateRead(ctx, owner, id)
	if err != nil {
		t.Fatalf("UpdateRead milik sendiri: %v", err)
	}
	if !read.IsRead || read.ReadAt == nil {
		t.Fatalf("notifikasi belum tertandai dibaca: %+v", read)
	}
	firstReadAt := *read.ReadAt

	again, err := repo.UpdateRead(ctx, owner, id)
	if err != nil {
		t.Fatalf("UpdateRead kedua: %v", err)
	}
	if again.ReadAt == nil || !again.ReadAt.Equal(firstReadAt) {
		t.Fatalf("read_at berubah pada pemanggilan ulang: %v -> %v", firstReadAt, again.ReadAt)
	}

	if _, err := repo.UpdateRead(ctx, other, id); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("UpdateRead milik user lain: error = %v, mau ErrNotFound", err)
	}
	if _, err := repo.UpdateRead(ctx, owner, "bukan-object-id"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("UpdateRead id tidak valid: error = %v, mau ErrNotFound", err)
	}
}

// C-I5
func TestUpdateAllReadAndCountUnread(t *testing.T) {
	repo, _ := newRepository(t)

	owner, other := nextID(), nextID()
	base := time.Now().Add(-time.Hour)

	insert(t, repo, testNotification(owner, nextID(), "pending", base))
	insert(t, repo, testNotification(owner, nextID(), "approved", base.Add(time.Minute)))
	insert(t, repo, testNotification(owner, nextID(), "cancelled", base.Add(2*time.Minute)))
	insert(t, repo, testNotification(other, nextID(), "pending", base))

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	all, err := repo.FindByUser(ctx, owner, false, 100)
	if err != nil || len(all) != 3 {
		t.Fatalf("FindByUser: %v, jumlah %d", err, len(all))
	}
	if _, err := repo.UpdateRead(ctx, owner, all[0].ID); err != nil {
		t.Fatalf("tandai satu sudah dibaca: %v", err)
	}

	unreadOwner, err := repo.CountUnread(ctx, owner)
	if err != nil || unreadOwner != 2 {
		t.Fatalf("CountUnread owner = %d, %v; mau 2", unreadOwner, err)
	}
	unreadOther, err := repo.CountUnread(ctx, other)
	if err != nil || unreadOther != 1 {
		t.Fatalf("CountUnread other = %d, %v; mau 1", unreadOther, err)
	}

	updated, err := repo.UpdateAllRead(ctx, owner)
	if err != nil {
		t.Fatalf("UpdateAllRead: %v", err)
	}
	if updated != 2 {
		t.Fatalf("UpdateAllRead mengubah %d dokumen, mau 2", updated)
	}

	unreadOwner, err = repo.CountUnread(ctx, owner)
	if err != nil || unreadOwner != 0 {
		t.Fatalf("CountUnread owner setelah read-all = %d, %v; mau 0", unreadOwner, err)
	}
	unreadOther, err = repo.CountUnread(ctx, other)
	if err != nil || unreadOther != 1 {
		t.Fatalf("notifikasi user lain ikut berubah: unread = %d, %v", unreadOther, err)
	}

	after, err := repo.FindByUser(ctx, owner, false, 100)
	if err != nil {
		t.Fatalf("FindByUser setelah read-all: %v", err)
	}
	for _, n := range after {
		if !n.IsRead || n.ReadAt == nil {
			t.Fatalf("notifikasi belum tertandai dibaca: %+v", n)
		}
		if n.CreatedAt.Location() != time.Local || n.ReadAt.Location() != time.Local {
			t.Fatalf("waktu tidak dalam time.Local: created=%v read=%v", n.CreatedAt.Location(), n.ReadAt.Location())
		}
	}
}