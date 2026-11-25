//go:build integration

// Integration test API notifikasi (Task C3 — matriks C-A1 s.d. C-A6 dari
// bagian 11.2 dokumen Phase 2). Semua test menembak API lewat HTTP; dokumen
// notifikasi disisipkan langsung ke Mongo test, bukan lewat worker.
//
// Menjalankan (lihat bagian 8.6):
//
//	1 terminal : jalankan API dengan konfigurasi test, misalnya APP_PORT=8081,
//	             DB_URL ke ruangku_test, REDIS_DB=15, MONGO_DB=ruangku_test,
//	             lalu `go run .`
//	2 terminal : set TEST_* (TEST_MONGO_DB=ruangku_test, dst.), lalu
//	             `go test -tags integration ./app/service -count=1`
//
// Tanpa variabel TEST_* test ini otomatis di-skip (testutil.Require).
package service_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"ruangku-backend/app/model"
	"ruangku-backend/database"
	"ruangku-backend/testutil"
)

const (
	notifPath = "/api/v1/notifications"

	// notifCollection sama dengan nama koleksi yang dipakai repository.
	notifCollection = "notifications"

	notifTimeout = 10 * time.Second
)

var notifSeq atomic.Int64

func init() {
	notifSeq.Store(time.Now().UnixNano() % 1_000_000_000)
}

// notifNextID menghasilkan booking_id unik antar test supaya index unik
// (booking_id, status) tidak bentrok.
func notifNextID() int {
	return int(notifSeq.Add(1))
}

type notifAccount struct {
	ID    int
	Email string
	Token string
}

// notifJSON mengirim request JSON dan men-decode envelope {data:...} ke out.
func notifJSON(t *testing.T, method, url, bearer string, body any, out any) int {
	t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = strings.NewReader(string(raw))
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("membuat request %s %s: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if out != nil {
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil && err != io.EOF {
			t.Fatalf("decode envelope %s: %v", url, err)
		}
		if len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, out); err != nil {
				t.Fatalf("decode data %s: %v", url, err)
			}
		}
	}
	return resp.StatusCode
}

// notifLogin mengembalikan access token untuk email/password tertentu.
func notifLogin(t *testing.T, env testutil.Env, email, password string) string {
	t.Helper()

	var login model.TokenResponse
	if code := notifJSON(t, http.MethodPost, env.APIURL+"/api/v1/auth/login", "", map[string]string{
		"email": email, "password": password,
	}, &login); code != http.StatusOK {
		t.Fatalf("login %s = %d, want 200", email, code)
	}
	return login.AccessToken
}

// notifRegister mendaftarkan user unik lalu login.
func notifRegister(t *testing.T, env testutil.Env, label string) notifAccount {
	t.Helper()

	email := testutil.Unique("notif-"+label) + "@example.test"
	var user model.User
	if code := notifJSON(t, http.MethodPost, env.APIURL+"/api/v1/auth/register", "", map[string]string{
		"name": "IT " + label, "email": email, "password": "rahasia123",
	}, &user); code != http.StatusCreated {
		t.Fatalf("register %s = %d, want 201", label, code)
	}

	return notifAccount{ID: user.ID, Email: email, Token: notifLogin(t, env, email, "rahasia123")}
}

// notifDB membuka koneksi Mongo test dan menutupnya saat test selesai.
func notifDB(t *testing.T, env testutil.Env) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), notifTimeout)
	defer cancel()

	db, err := database.NewMongo(ctx, env.MongoURI, env.MongoDB)
	if err != nil {
		t.Fatalf("konek mongodb test: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), notifTimeout)
		defer cancelCleanup()
		_ = db.Client().Disconnect(cleanupCtx)
	})
	return db
}

// notifCleanup menghapus dokumen milik user test setelah test selesai.
func notifCleanup(t *testing.T, db *mongo.Database, userIDs ...int) {
	t.Helper()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), notifTimeout)
		defer cancel()
		_, _ = db.Collection(notifCollection).DeleteMany(ctx, bson.M{
			"user_id": bson.M{"$in": userIDs},
		})
	})
}

// notifInsert menyisipkan satu dokumen notifikasi langsung ke Mongo dan
// mengembalikan ObjectID hex-nya.
func notifInsert(t *testing.T, db *mongo.Database, userID int, status string, isRead bool, createdAt time.Time) string {
	t.Helper()

	doc := bson.M{
		"_id":        bson.NewObjectID(),
		"user_id":    userID,
		"booking_id": notifNextID(),
		"status":     status,
		"title":      "Peminjaman " + status,
		"message":    "Peminjaman uji " + status,
		"is_read":    isRead,
		"created_at": createdAt,
	}
	if isRead {
		doc["read_at"] = createdAt.Add(time.Minute)
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifTimeout)
	defer cancel()

	if _, err := db.Collection(notifCollection).InsertOne(ctx, doc); err != nil {
		t.Fatalf("insert notifikasi: %v", err)
	}
	return doc["_id"].(bson.ObjectID).Hex()
}

// C-A1: GET /notifications 200, hanya milik pemanggil, terbaru dulu; 401
// tanpa token. Data disisipkan langsung ke Mongo, bukan lewat worker.
func TestCA1_ListOnlyOwnNotifications(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)

	db := notifDB(t, env)
	owner := notifRegister(t, env, "ca1a")
	other := notifRegister(t, env, "ca1b")
	notifCleanup(t, db, owner.ID, other.ID)

	base := time.Now().Add(-time.Hour)
	notifInsert(t, db, owner.ID, "pending", false, base)
	notifInsert(t, db, owner.ID, "approved", false, base.Add(time.Minute))
	notifInsert(t, db, other.ID, "pending", false, base)

	var list []model.Notification
	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath, owner.Token, nil, &list); code != http.StatusOK {
		t.Fatalf("daftar notifikasi = %d, want 200", code)
	}
	if len(list) != 2 {
		t.Fatalf("jumlah notifikasi = %d, want 2", len(list))
	}
	if list[0].Status != "approved" {
		t.Fatalf("urutan terbaru salah: dapat %q, want 'approved'", list[0].Status)
	}
	for _, n := range list {
		if n.UserID != owner.ID {
			t.Fatalf("notifikasi user lain ikut terbaca: user_id = %d", n.UserID)
		}
	}

	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath, "", nil, nil); code != http.StatusUnauthorized {
		t.Fatalf("daftar tanpa token = %d, want 401", code)
	}
}

// C-A2: ?is_read=false hanya mengembalikan yang belum dibaca; nilai selain
// kosong dan false ditolak 400.
func TestCA2_UnreadFilter(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)

	db := notifDB(t, env)
	owner := notifRegister(t, env, "ca2")
	notifCleanup(t, db, owner.ID)

	base := time.Now().Add(-time.Hour)
	notifInsert(t, db, owner.ID, "pending", false, base)
	notifInsert(t, db, owner.ID, "approved", false, base.Add(time.Minute))
	notifInsert(t, db, owner.ID, "cancelled", true, base.Add(2*time.Minute))

	var unread []model.Notification
	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath+"?is_read=false", owner.Token, nil, &unread); code != http.StatusOK {
		t.Fatalf("filter belum dibaca = %d, want 200", code)
	}
	if len(unread) != 2 {
		t.Fatalf("jumlah belum dibaca = %d, want 2", len(unread))
	}
	for _, n := range unread {
		if n.IsRead {
			t.Fatalf("notifikasi sudah dibaca ikut terfilter: id = %s", n.ID)
		}
	}

	var all []model.Notification
	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath, owner.Token, nil, &all); code != http.StatusOK {
		t.Fatalf("daftar tanpa filter = %d, want 200", code)
	}
	if len(all) != 3 {
		t.Fatalf("jumlah tanpa filter = %d, want 3", len(all))
	}

	for _, invalid := range []string{"true", "0", "semua"} {
		if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath+"?is_read="+invalid, owner.Token, nil, nil); code != http.StatusBadRequest {
			t.Fatalf("is_read=%q = %d, want 400", invalid, code)
		}
	}
}

// C-A3: GET /notifications/unread-count menghitung hanya notifikasi belum
// dibaca milik pemanggil.
func TestCA3_UnreadCount(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)

	db := notifDB(t, env)
	owner := notifRegister(t, env, "ca3a")
	other := notifRegister(t, env, "ca3b")
	notifCleanup(t, db, owner.ID, other.ID)

	base := time.Now().Add(-time.Hour)
	notifInsert(t, db, owner.ID, "pending", false, base)
	notifInsert(t, db, owner.ID, "approved", false, base.Add(time.Minute))
	notifInsert(t, db, owner.ID, "cancelled", true, base.Add(2*time.Minute))
	notifInsert(t, db, other.ID, "pending", false, base)

	var count struct {
		Count int `json:"count"`
	}
	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath+"/unread-count", owner.Token, nil, &count); code != http.StatusOK {
		t.Fatalf("jumlah belum dibaca = %d, want 200", code)
	}
	if count.Count != 2 {
		t.Fatalf("count = %d, want 2", count.Count)
	}

	var otherCount struct {
		Count int `json:"count"`
	}
	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath+"/unread-count", other.Token, nil, &otherCount); code != http.StatusOK {
		t.Fatalf("jumlah belum dibaca user lain = %d, want 200", code)
	}
	if otherCount.Count != 1 {
		t.Fatalf("count user lain = %d, want 1", otherCount.Count)
	}

	if code := notifJSON(t, http.MethodGet, env.APIURL+notifPath+"/unread-count", "", nil, nil); code != http.StatusUnauthorized {
		t.Fatalf("unread-count tanpa token = %d, want 401", code)
	}
}

// C-A4: PATCH /notifications/:id/read — 200 untuk milik sendiri dan
// idempotent, 404 untuk milik user lain, 400 untuk id tidak valid.
func TestCA4_MarkRead(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)

	db := notifDB(t, env)
	owner := notifRegister(t, env, "ca4a")
	other := notifRegister(t, env, "ca4b")
	notifCleanup(t, db, owner.ID, other.ID)

	base := time.Now().Add(-time.Hour)
	ownID := notifInsert(t, db, owner.ID, "pending", false, base)
	otherID := notifInsert(t, db, other.ID, "pending", false, base)

	var first model.Notification
	if code := notifJSON(t, http.MethodPatch, env.APIURL+notifPath+"/"+ownID+"/read", owner.Token, nil, &first); code != http.StatusOK {
		t.Fatalf("tandai sudah dibaca = %d, want 200", code)
	}
	if first.ID != ownID {
		t.Fatalf("id yang dikembalikan = %q, want %q", first.ID, ownID)
	}
	if !first.IsRead {
		t.Fatal("is_read masih false setelah ditandai")
	}
	if first.ReadAt == nil {
		t.Fatal("read_at kosong setelah ditandai")
	}

	var second model.Notification
	if code := notifJSON(t, http.MethodPatch, env.APIURL+notifPath+"/"+ownID+"/read", owner.Token, nil, &second); code != http.StatusOK {
		t.Fatalf("tandai ulang = %d, want 200", code)
	}
	if second.ReadAt == nil || !second.ReadAt.Equal(*first.ReadAt) {
		t.Fatalf("read_at berubah pada pemanggilan kedua: first=%v second=%v", first.ReadAt, second.ReadAt)
	}

	if code := notifJSON(t, http.MethodPatch, env.APIURL+notifPath+"/"+otherID+"/read", owner.Token, nil, nil); code != http.StatusNotFound {
		t.Fatalf("tandai notifikasi user lain = %d, want 404", code)
	}

	if code := notifJSON(t, http.MethodPatch, env.APIURL+notifPath+"/bukan-object-id/read", owner.Token, nil, nil); code != http.StatusBadRequest {
		t.Fatalf("id tidak valid = %d, want 400", code)
	}

	if code := notifJSON(t, http.MethodPatch, env.APIURL+notifPath+"/"+ownID+"/read", "", nil, nil); code != http.StatusUnauthorized {
		t.Fatalf("tandai tanpa token = %d, want 401", code)
	}
}
