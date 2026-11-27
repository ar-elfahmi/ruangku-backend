//go:build integration

// Integration test booking (Task B4: matriks B-I1 s.d. B-I5 dari bagian 11.1
// dokumen Phase 2). Semua test menembak API lewat HTTP dan memeriksa hasilnya
// lewat database test (PostgreSQL dan MongoDB). B-I6 s.d. B-I8 ada di
// queue/worker_integration_test.go dan queue/producer_unreachable_test.go.
//
// Menjalankan (lihat bagian 8.6):
//
//	terminal 1 (API) : APP_PORT=8081, DB_URL ke ruangku_test, REDIS_DB=15,
//	                   MONGO_DB=ruangku_test, JWT_SECRET, JWT_REFRESH_SECRET,
//	                   ADMIN_EMAIL/ADMIN_PASSWORD, lalu `go run .`
//	terminal 2 (worker, hanya untuk B-I5) : REDIS_DB=15, MONGO_DB=ruangku_test,
//	                   lalu `go run ./cmd/worker`
//	terminal 3 (test) : set TEST_*, lalu
//	                   `go test -tags integration ./app/service -run BI -count=1`
//
// Tanpa variabel TEST_* test ini otomatis di-skip (testutil.Require). Data
// test unik (awalan it-) dan tidak pernah dihapus.
//
// Helper umum (probeJSON, apiJSON, dbOne) dipakai bersama dari
// session_integration_test.go; helper khusus booking berawalan bk.
package service_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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
	bkAPI        = "/api/v1"
	bkNotifColl  = "notifications"
	bkMongoWait  = 10 * time.Second
	bkRaceRounds = 5
)

var bkRoomSeq atomic.Int64

func init() {
	bkRoomSeq.Store(time.Now().UnixNano() % 1_000_000_000)
}

// ---------------------------------------------------------------------------
// Bantuan
// ---------------------------------------------------------------------------

// bkEnv membaca TEST_* lalu menjalankan canary: test dihentikan bila API tidak
// berjalan dengan database test.
func bkEnv(t *testing.T) testutil.Env {
	t.Helper()
	env := testutil.Require(t)
	testutil.Canary(t, env)
	return env
}

type bkUser struct {
	ID    int
	Token string
}

// bkRegister mendaftarkan user unik (peminjam) lalu login.
func bkRegister(t *testing.T, env testutil.Env, label string) bkUser {
	t.Helper()

	email := testutil.Unique("bk-"+label) + "@example.test"
	var user model.User
	if code := apiJSON(t, http.MethodPost, env.APIURL+bkAPI+"/auth/register", "", map[string]string{
		"name": "IT " + label, "email": email, "password": "rahasia123",
	}, &user); code != http.StatusCreated {
		t.Fatalf("register %s = %d, want 201", label, code)
	}
	return bkUser{ID: user.ID, Token: bkLogin(t, env, email, "rahasia123")}
}

func bkLogin(t *testing.T, env testutil.Env, email, password string) string {
	t.Helper()

	var login model.TokenResponse
	if code := apiJSON(t, http.MethodPost, env.APIURL+bkAPI+"/auth/login", "", map[string]string{
		"email": email, "password": password,
	}, &login); code != http.StatusOK {
		t.Fatalf("login %s = %d, want 200", email, code)
	}
	return login.AccessToken
}

// bkAdminToken login sebagai admin memakai TEST_ADMIN_*.
func bkAdminToken(t *testing.T, env testutil.Env) string {
	t.Helper()
	return bkLogin(t, env, env.AdminEmail, env.AdminPassword)
}

// bkPengelolaToken membuat user baru, menjadikannya pengelola lewat API admin,
// lalu login ulang agar token membawa role pengelola.
func bkPengelolaToken(t *testing.T, env testutil.Env, adminToken string) string {
	t.Helper()

	email := testutil.Unique("bk-pengelola") + "@example.test"
	var user model.User
	if code := apiJSON(t, http.MethodPost, env.APIURL+bkAPI+"/auth/register", "", map[string]string{
		"name": "IT pengelola", "email": email, "password": "rahasia123",
	}, &user); code != http.StatusCreated {
		t.Fatalf("register pengelola = %d, want 201", code)
	}
	url := fmt.Sprintf("%s%s/users/%d/role", env.APIURL, bkAPI, user.ID)
	if code := apiJSON(t, http.MethodPatch, url, adminToken, map[string]string{"role": model.RolePengelola}, nil); code != http.StatusOK {
		t.Fatalf("jadikan pengelola = %d, want 200", code)
	}
	return bkLogin(t, env, email, "rahasia123")
}

// bkRoom membuat ruang unik (setiap test punya ruang sendiri sehingga jadwal
// antar-test tidak pernah bentrok).
func bkRoom(t *testing.T, env testutil.Env, adminToken string) int {
	t.Helper()

	var room model.Room
	code := apiJSON(t, http.MethodPost, env.APIURL+bkAPI+"/rooms", adminToken, map[string]any{
		"code":     fmt.Sprintf("IT%d", bkRoomSeq.Add(1)), // maks 20 karakter
		"name":     "Ruang uji booking",
		"building": "Gedung IT",
		"capacity": 10,
	}, &room)
	if code != http.StatusCreated {
		t.Fatalf("buat ruang = %d, want 201", code)
	}
	return room.ID
}

// bkSlot adalah jadwal 10:00-12:00 pada hari ke-day (Maret 2035, selalu di
// masa depan).
func bkSlot(day int) (start, end string) {
	return fmt.Sprintf("2035-03-%02dT10:00:00+07:00", day), fmt.Sprintf("2035-03-%02dT12:00:00+07:00", day)
}

// bkTryCreate membuat booking dan aman dipanggil dari goroutine.
func bkTryCreate(env testutil.Env, token string, roomID int, start, end string) (int, model.Booking, error) {
	var booking model.Booking
	code, err := probeJSON(http.MethodPost, env.APIURL+bkAPI+"/bookings", token, map[string]any{
		"room_id": roomID, "purpose": "Uji integrasi", "start_time": start, "end_time": end,
	}, &booking)
	return code, booking, err
}

func bkCreate(t *testing.T, env testutil.Env, token string, roomID int, start, end string) model.Booking {
	t.Helper()

	code, booking, err := bkTryCreate(env, token, roomID, start, end)
	if err != nil {
		t.Fatalf("POST /bookings: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("POST /bookings = %d, want 201", code)
	}
	return booking
}

// bkTryTransition memanggil PATCH /bookings/:id/<action> dan aman dipanggil
// dari goroutine.
func bkTryTransition(env testutil.Env, token string, bookingID int, action string) (int, error) {
	url := fmt.Sprintf("%s%s/bookings/%d/%s", env.APIURL, bkAPI, bookingID, action)
	return probeJSON(http.MethodPatch, url, token, nil, nil)
}

func bkTransition(t *testing.T, env testutil.Env, token string, bookingID int, action string, want int) {
	t.Helper()

	code, err := bkTryTransition(env, token, bookingID, action)
	if err != nil {
		t.Fatalf("%s booking %d: %v", action, bookingID, err)
	}
	if code != want {
		t.Fatalf("%s booking %d = %d, want %d", action, bookingID, code, want)
	}
}

func bkLogs(t *testing.T, env testutil.Env, token string, bookingID int) []model.BookingLog {
	t.Helper()

	var logs []model.BookingLog
	url := fmt.Sprintf("%s%s/bookings/%d/logs", env.APIURL, bkAPI, bookingID)
	if code := apiJSON(t, http.MethodGet, url, token, nil, &logs); code != http.StatusOK {
		t.Fatalf("GET logs booking %d = %d, want 200", bookingID, code)
	}
	return logs
}

// bkStatus membaca status booking langsung dari database test.
func bkStatus(t *testing.T, env testutil.Env, bookingID int) string {
	t.Helper()

	status, ok := dbOne(t, env, "SELECT status FROM bookings WHERE id = $1", bookingID)
	if !ok {
		t.Fatalf("booking %d tidak ada di database test", bookingID)
	}
	return status
}

// bkParallel menjalankan call sebanyak count kali secara bersamaan: semua
// goroutine siap di gerbang, lalu dilepas serentak. Mengembalikan status
// HTTP tiap panggilan.
func bkParallel(t *testing.T, count int, call func(i int) (int, error)) []int {
	t.Helper()

	codes := make([]int, count)
	errs := make([]error, count)
	var ready, done sync.WaitGroup
	ready.Add(count)
	done.Add(count)
	gate := make(chan struct{})

	for i := 0; i < count; i++ {
		go func(i int) {
			defer done.Done()
			ready.Done()
			<-gate
			codes[i], errs[i] = call(i)
		}(i)
	}
	ready.Wait()
	close(gate)
	done.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("request paralel ke-%d gagal: %v", i, err)
		}
	}
	return codes
}

func bkCount(codes []int, status int) int {
	n := 0
	for _, code := range codes {
		if code == status {
			n++
		}
	}
	return n
}

// bkAssertLogChain memeriksa jejak status: log pertama null -> pending, tiap
// log berikutnya dimulai dari status tujuan log sebelumnya, dan jumlahnya
// sesuai.
func bkAssertLogChain(t *testing.T, logs []model.BookingLog, wantLen int) {
	t.Helper()

	if len(logs) != wantLen {
		t.Fatalf("jumlah log = %d, want %d: %+v", len(logs), wantLen, logs)
	}
	if logs[0].FromStatus != nil || logs[0].ToStatus != model.StatusPending {
		t.Fatalf("log pertama harus null -> pending: %+v", logs[0])
	}
	for i := 1; i < len(logs); i++ {
		if logs[i].FromStatus == nil || *logs[i].FromStatus != logs[i-1].ToStatus {
			t.Fatalf("rantai log putus di indeks %d: %+v", i, logs)
		}
	}
}

// ---------------------------------------------------------------------------
// B-I1 s.d. B-I4: konkurensi booking (hanya API + PostgreSQL)
// ---------------------------------------------------------------------------

// B-I1 (slot race): 10 peminjam memesan slot yang sama serentak. Tepat satu
// 201 dan sembilan 409; hanya satu booking aktif untuk slot itu dan tepat
// satu log pending. Pengamannya constraint database, bukan kode aplikasi.
func TestBI1_SlotRaceExactlyOneWins(t *testing.T) {
	env := bkEnv(t)
	adminToken := bkAdminToken(t, env)
	roomID := bkRoom(t, env, adminToken)
	start, end := bkSlot(1)

	const borrowers = 10
	users := make([]bkUser, borrowers)
	for i := range users {
		users[i] = bkRegister(t, env, fmt.Sprintf("racer%d", i))
	}

	codes := bkParallel(t, borrowers, func(i int) (int, error) {
		code, _, err := bkTryCreate(env, users[i].Token, roomID, start, end)
		return code, err
	})

	if got := bkCount(codes, http.StatusCreated); got != 1 {
		t.Fatalf("jumlah 201 = %d, want 1 (kode: %v)", got, codes)
	}
	if got := bkCount(codes, http.StatusConflict); got != borrowers-1 {
		t.Fatalf("jumlah 409 = %d, want %d (kode: %v)", got, borrowers-1, codes)
	}

	active, _ := dbOne(t, env, "SELECT count(*)::text FROM bookings WHERE room_id = $1 AND status IN ('pending', 'approved')", roomID)
	if active != "1" {
		t.Fatalf("booking aktif untuk slot = %s, want 1", active)
	}
	pendingLogs, _ := dbOne(t, env, `SELECT count(*)::text FROM booking_logs l JOIN bookings b ON b.id = l.booking_id
		WHERE b.room_id = $1 AND l.to_status = 'pending' AND l.from_status IS NULL`, roomID)
	if pendingLogs != "1" {
		t.Fatalf("log pending untuk ruang = %s, want 1", pendingLogs)
	}
	allLogs, _ := dbOne(t, env, `SELECT count(*)::text FROM booking_logs l JOIN bookings b ON b.id = l.booking_id WHERE b.room_id = $1`, roomID)
	if allLogs != "1" {
		t.Fatalf("booking yang kalah tidak boleh meninggalkan log; total log = %s, want 1", allLogs)
	}
}

// B-I2 (approve vs cancel serentak): hasil sah ada tiga (approve menang dan
// cancel 409; cancel menang dan approve 409; keduanya 200 dengan status akhir
// cancelled). Yang dijaga hanya invarian: tidak ada 5xx, minimal satu 200,
// rantai log konsisten, jumlah log = 1 + jumlah keberhasilan, dan status akhir
// sama dengan tujuan log terakhir. Diulang beberapa kali karena balapan.
func TestBI2_ApproveVsCancelInvariants(t *testing.T) {
	env := bkEnv(t)
	adminToken := bkAdminToken(t, env)
	pengelola := bkPengelolaToken(t, env, adminToken)
	owner := bkRegister(t, env, "owner")
	roomID := bkRoom(t, env, adminToken)

	for round := 1; round <= bkRaceRounds; round++ {
		start, end := bkSlot(round)
		booking := bkCreate(t, env, owner.Token, roomID, start, end)

		codes := bkParallel(t, 2, func(i int) (int, error) {
			if i == 0 {
				return bkTryTransition(env, pengelola, booking.ID, "approve")
			}
			return bkTryTransition(env, owner.Token, booking.ID, "cancel")
		})
		approveCode, cancelCode := codes[0], codes[1]

		var wantFinal string
		switch {
		case approveCode == http.StatusOK && cancelCode == http.StatusConflict:
			wantFinal = model.StatusApproved
		case approveCode == http.StatusConflict && cancelCode == http.StatusOK:
			wantFinal = model.StatusCancelled
		case approveCode == http.StatusOK && cancelCode == http.StatusOK:
			wantFinal = model.StatusCancelled
		default:
			t.Fatalf("ronde %d: kombinasi kode tidak sah approve=%d cancel=%d", round, approveCode, cancelCode)
		}

		successes := bkCount(codes, http.StatusOK)
		logs := bkLogs(t, env, owner.Token, booking.ID)
		bkAssertLogChain(t, logs, 1+successes)
		if last := logs[len(logs)-1].ToStatus; last != wantFinal {
			t.Fatalf("ronde %d: status akhir pada log = %s, want %s", round, last, wantFinal)
		}
		if got := bkStatus(t, env, booking.ID); got != wantFinal {
			t.Fatalf("ronde %d: status booking = %s, want %s", round, got, wantFinal)
		}
	}
}

// B-I3: approve lalu cancel berurutan: 200 dan 200, jejak
// null -> pending -> approved -> cancelled.
func TestBI3_ApproveThenCancelInSequence(t *testing.T) {
	env := bkEnv(t)
	adminToken := bkAdminToken(t, env)
	pengelola := bkPengelolaToken(t, env, adminToken)
	owner := bkRegister(t, env, "owner")
	roomID := bkRoom(t, env, adminToken)
	start, end := bkSlot(1)

	booking := bkCreate(t, env, owner.Token, roomID, start, end)
	bkTransition(t, env, pengelola, booking.ID, "approve", http.StatusOK)
	bkTransition(t, env, owner.Token, booking.ID, "cancel", http.StatusOK)

	logs := bkLogs(t, env, owner.Token, booking.ID)
	bkAssertLogChain(t, logs, 3)
	if logs[1].ToStatus != model.StatusApproved || logs[2].ToStatus != model.StatusCancelled {
		t.Fatalf("jejak harus pending -> approved -> cancelled: %+v", logs)
	}
	if got := bkStatus(t, env, booking.ID); got != model.StatusCancelled {
		t.Fatalf("status akhir = %s, want cancelled", got)
	}
}

// B-I4: approve vs reject serentak: tepat satu 200 dan satu 409, tepat satu log
// tambahan. Approve ganda serentak: tepat satu 200.
func TestBI4_ApproveVsRejectAndDoubleApprove(t *testing.T) {
	env := bkEnv(t)
	adminToken := bkAdminToken(t, env)
	pengelola := bkPengelolaToken(t, env, adminToken)
	owner := bkRegister(t, env, "owner")
	roomID := bkRoom(t, env, adminToken)

	t.Run("approve vs reject", func(t *testing.T) {
		for round := 1; round <= bkRaceRounds; round++ {
			start, end := bkSlot(round)
			booking := bkCreate(t, env, owner.Token, roomID, start, end)

			codes := bkParallel(t, 2, func(i int) (int, error) {
				action := "approve"
				if i == 1 {
					action = "reject"
				}
				return bkTryTransition(env, pengelola, booking.ID, action)
			})

			if bkCount(codes, http.StatusOK) != 1 || bkCount(codes, http.StatusConflict) != 1 {
				t.Fatalf("ronde %d: harus tepat satu 200 dan satu 409, dapat %v", round, codes)
			}
			wantFinal := model.StatusApproved
			if codes[1] == http.StatusOK {
				wantFinal = model.StatusRejected
			}
			bkAssertLogChain(t, bkLogs(t, env, owner.Token, booking.ID), 2)
			if got := bkStatus(t, env, booking.ID); got != wantFinal {
				t.Fatalf("ronde %d: status = %s, want %s", round, got, wantFinal)
			}
		}
	})

	t.Run("double approve", func(t *testing.T) {
		for round := 1; round <= bkRaceRounds; round++ {
			start, end := bkSlot(10 + round) // hari berbeda dari subtest sebelumnya
			booking := bkCreate(t, env, owner.Token, roomID, start, end)

			codes := bkParallel(t, 2, func(int) (int, error) {
				return bkTryTransition(env, pengelola, booking.ID, "approve")
			})

			if bkCount(codes, http.StatusOK) != 1 || bkCount(codes, http.StatusConflict) != 1 {
				t.Fatalf("ronde %d: approve ganda harus tepat satu 200 dan satu 409, dapat %v", round, codes)
			}
			bkAssertLogChain(t, bkLogs(t, env, owner.Token, booking.ID), 2)
			if got := bkStatus(t, env, booking.ID); got != model.StatusApproved {
				t.Fatalf("ronde %d: status = %s, want approved", round, got)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// B-I5: end to end (API + worker + MongoDB)
// ---------------------------------------------------------------------------

func bkMongo(t *testing.T, env testutil.Env) *mongo.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), bkMongoWait)
	defer cancel()

	db, err := database.NewMongo(ctx, env.MongoURI, env.MongoDB)
	if err != nil {
		t.Fatalf("konek mongodb test: %v", err)
	}
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), bkMongoWait)
		defer closeCancel()
		_ = db.Client().Disconnect(closeCtx)
	})
	return db
}

// bkNotifCount menghitung dokumen notifikasi yang cocok dengan filter.
func bkNotifCount(t *testing.T, db *mongo.Database, filter bson.M) int64 {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), bkMongoWait)
	defer cancel()

	n, err := db.Collection(bkNotifColl).CountDocuments(ctx, filter)
	if err != nil {
		t.Fatalf("hitung notifikasi %v: %v", filter, err)
	}
	return n
}

// B-I5: dengan worker test berjalan, POST /bookings menghasilkan satu dokumen
// Mongo status=pending milik pemilik dalam 10 detik; approve menghasilkan
// dokumen kedua approved; tiap (booking_id, status) tepat satu dokumen; dan
// peminjam yang ditolak karena bentrok (409) tidak menerima notifikasi apa pun.
func TestBI5_EndToEndNotifications(t *testing.T) {
	env := bkEnv(t)
	db := bkMongo(t, env)
	adminToken := bkAdminToken(t, env)
	pengelola := bkPengelolaToken(t, env, adminToken)
	owner := bkRegister(t, env, "owner")
	rival := bkRegister(t, env, "rival")
	roomID := bkRoom(t, env, adminToken)
	start, end := bkSlot(1)
	const workerHint = "pastikan worker test berjalan (go run ./cmd/worker dengan REDIS_DB=15 dan MONGO_DB=ruangku_test)"

	booking := bkCreate(t, env, owner.Token, roomID, start, end)
	pendingFilter := bson.M{"booking_id": booking.ID, "user_id": owner.ID, "status": model.StatusPending}
	testutil.WaitFor(t, testutil.DefaultWait, testutil.DefaultInterval, func() bool {
		return bkNotifCount(t, db, pendingFilter) == 1
	}, "notifikasi pending milik pemilik; "+workerHint)

	// Pesaing memesan slot yang sama: 409, dan tidak boleh ada notifikasi.
	if code, _, err := bkTryCreate(env, rival.Token, roomID, start, end); err != nil || code != http.StatusConflict {
		t.Fatalf("booking bentrok = %d (err %v), want 409", code, err)
	}

	bkTransition(t, env, pengelola, booking.ID, "approve", http.StatusOK)
	approvedFilter := bson.M{"booking_id": booking.ID, "user_id": owner.ID, "status": model.StatusApproved}
	testutil.WaitFor(t, testutil.DefaultWait, testutil.DefaultInterval, func() bool {
		return bkNotifCount(t, db, approvedFilter) == 1
	}, "notifikasi approved milik pemilik; "+workerHint)

	// Event pending pesaing (kalau bug) sudah lebih dulu dikirim daripada event
	// approved di atas, jadi pada titik ini ia seharusnya sudah muncul.
	if got := bkNotifCount(t, db, bson.M{"user_id": rival.ID}); got != 0 {
		t.Fatalf("peminjam yang ditolak karena bentrok menerima %d notifikasi, want 0", got)
	}
	if got := bkNotifCount(t, db, bson.M{"booking_id": booking.ID}); got != 2 {
		t.Fatalf("dokumen untuk booking %d = %d, want 2 (pending dan approved)", booking.ID, got)
	}
	for _, status := range []string{model.StatusPending, model.StatusApproved} {
		if got := bkNotifCount(t, db, bson.M{"booking_id": booking.ID, "status": status}); got != 1 {
			t.Fatalf("dokumen (booking %d, %s) = %d, want 1", booking.ID, status, got)
		}
	}
}
