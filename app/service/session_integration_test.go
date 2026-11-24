//go:build integration

// Integration test fitur refresh JWT (Task D4 — matriks D-I1 s.d. D-I10 dari
// bagian 11.3 dokumen Phase 2). Semua test menembak API lewat HTTP dan — bila
// perlu — memeriksa baris langsung di database test lewat TEST_DB_URL.
//
// Menjalankan (lihat juga bagian 8.6 dokumen Phase 2):
//
//	1 terminal : set APP_PORT=8081, DB_URL (ruangku_test), REDIS_DB=15,
//	             MONGO_DB=ruangku_test, JWT_SECRET & JWT_REFRESH_SECRET,
//	             lalu `go run .`
//	2 terminal : set TEST_API_URL, TEST_DB_URL, TEST_REDIS_ADDR,
//	             TEST_REDIS_DB=15, TEST_MONGO_URI, TEST_MONGO_DB,
//	             TEST_ADMIN_EMAIL, TEST_ADMIN_PASSWORD, lalu
//	             `go test -tags integration ./... -count=1`
//
// Tanpa variabel TEST_* test ini otomatis di-skip (testutil.Require).
package service_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"ruangku-backend/app/model"
	"ruangku-backend/testutil"
)

// ---------------------------------------------------------------------------
// Bantuan HTTP + SQL
// ---------------------------------------------------------------------------

// probeJSON mengirim request JSON dan men-decode envelope {data:...} ke out
// (bila out != nil). Aman dipanggil dari goroutine: tidak memanggil t.Fatal,
// melainkan mengembalikan error.
func probeJSON(method, url, bearer string, body any, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = strings.NewReader(string(raw))
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if out != nil {
		var envelope struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil && err != io.EOF {
			return resp.StatusCode, err
		}
		if len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, out); err != nil {
				return resp.StatusCode, err
			}
		}
	}
	return resp.StatusCode, nil
}

// apiJSON seperti probeJSON, tapi langsung t.Fatal bila request gagal.
func apiJSON(t *testing.T, method, url, bearer string, body any, out any) int {
	t.Helper()

	code, err := probeJSON(method, url, bearer, body, out)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return code
}

// dbOne menjalankan query satu kolom ke database test.
// ok=false berarti tidak ada baris.
func dbOne(t *testing.T, env testutil.Env, query string, args ...any) (value string, ok bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, env.DBURL)
	if err != nil {
		t.Fatalf("koneksi TEST_DB_URL: %v", err)
	}
	defer conn.Close(ctx)

	err = conn.QueryRow(ctx, query, args...).Scan(&value)
	if err == pgx.ErrNoRows {
		return "", false
	}
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value, true
}

// dbExec menjalankan UPDATE/DELETE ke database test, mengembalikan jumlah
// baris yang terpengaruh.
func dbExec(t *testing.T, env testutil.Env, query string, args ...any) int64 {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, env.DBURL)
	if err != nil {
		t.Fatalf("koneksi TEST_DB_URL: %v", err)
	}
	defer conn.Close(ctx)

	tag, err := conn.Exec(ctx, query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	return tag.RowsAffected()
}

// jwtField membaca satu klaim string dari payload JWT tanpa memverifikasi
// signature. Hanya untuk keperluan test (menemukan fid/jti), bukan mekanisme
// keamanan.
func jwtField(t *testing.T, token, field string) string {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatal("token tidak berformat JWT")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload JWT: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload JWT: %v", err)
	}
	v, _ := payload[field].(string)
	return v
}

// registerAndLogin membuat user unik lalu login, mengembalikan pasangan token.
func registerAndLogin(t *testing.T, env testutil.Env, label string) model.TokenResponse {
	t.Helper()

	email := testutil.Unique(label) + "@example.test"
	api := env.APIURL + "/api/v1"

	var registered model.User
	if code := apiJSON(t, http.MethodPost, api+"/auth/register", "", map[string]string{
		"name": "IT " + label, "email": email, "password": "rahasia123",
	}, &registered); code != http.StatusCreated {
		t.Fatalf("register %s = %d, want 201", label, code)
	}

	var login model.TokenResponse
	if code := apiJSON(t, http.MethodPost, api+"/auth/login", "", map[string]string{
		"email": email, "password": "rahasia123",
	}, &login); code != http.StatusOK {
		t.Fatalf("login %s = %d, want 200", label, code)
	}
	return login
}

// familyReason mengembalikan revoked_reason sebuah keluarga ("" = aktif).
func familyReason(t *testing.T, env testutil.Env, familyID string) (string, bool) {
	t.Helper()
	return dbOne(t, env, "SELECT COALESCE(revoked_reason, '') FROM refresh_families WHERE id = $1", familyID)
}

// ---------------------------------------------------------------------------
// D-I1 s.d. D-I10
// ---------------------------------------------------------------------------

// D-I1: login mengembalikan refresh_token; refresh memutar pasangan; logout
// 204; refresh sesudahnya 401.
func TestDI1_LoginRefreshLogoutFlow(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di1")
	if login.RefreshToken == "" {
		t.Fatal("login tidak mengembalikan refresh_token")
	}

	var rotated model.TokenResponse
	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.RefreshToken,
	}, &rotated); code != http.StatusOK {
		t.Fatalf("refresh = %d, want 200", code)
	}
	if rotated.RefreshToken == "" || rotated.RefreshToken == login.RefreshToken {
		t.Fatal("refresh harus memberikan refresh token BARU yang berbeda")
	}

	if code := apiJSON(t, http.MethodPost, api+"/auth/logout", rotated.AccessToken, map[string]string{
		"refresh_token": rotated.RefreshToken,
	}, nil); code != http.StatusNoContent {
		t.Fatalf("logout = %d, want 204", code)
	}

	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": rotated.RefreshToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("refresh setelah logout = %d, want 401", code)
	}
}

// D-I2: reuse — token lama dipakai lagi setelah rotasi → 401, keluarga
// dicabut dengan revoked_reason='reuse' (dicek via SQL), dan token rotasi
// juga ikut 401. Pencabutan tetap ada (tidak ikut rollback).
func TestDI2_ReuseRevokesFamily(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di2")
	familyID := jwtField(t, login.RefreshToken, "fid")

	// Rotasi A → B.
	var rotated model.TokenResponse
	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.RefreshToken,
	}, &rotated); code != http.StatusOK {
		t.Fatalf("refresh pertama = %d, want 200", code)
	}

	// Pakai A lagi → 401.
	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.RefreshToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("reuse token lama = %d, want 401", code)
	}

	// Keluarga tercatat dicabut karena reuse.
	reason, ok := familyReason(t, env, familyID)
	if !ok {
		t.Fatal("keluarga tidak ditemukan di database test")
	}
	if reason != "reuse" {
		t.Fatalf("revoked_reason = %q, want 'reuse'", reason)
	}

	// Token rotasi (B) juga mati.
	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": rotated.RefreshToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("refresh dengan token rotasi = %d, want 401", code)
	}
}

// D-I3: 10 request refresh paralel dengan token yang sama → tepat satu 200,
// sisanya 401; keluarga dicabut; tabel refresh_tokens berisi 2 baris.
func TestDI3_ConcurrentRefreshExactlyOneSuccess(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di3")
	familyID := jwtField(t, login.RefreshToken, "fid")

	const n = 10
	results := make([]int, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = probeJSON(http.MethodPost, api+"/auth/refresh", "", map[string]string{
				"refresh_token": login.RefreshToken,
			}, nil)
		}(i)
	}
	wg.Wait()

	success := 0
	for i, code := range results {
		if errs[i] != nil {
			t.Fatalf("request paralel ke-%d gagal terkirim: %v", i, errs[i])
		}
		switch {
		case code == http.StatusOK:
			success++
		case code == http.StatusUnauthorized:
			// diharapkan
		default:
			t.Fatalf("request paralel ke-%d = %d, want 200 atau 401 (tanpa 5xx)", i, code)
		}
	}
	if success != 1 {
		t.Fatalf("jumlah refresh sukses = %d, want tepat 1", success)
	}

	// Keluarga dicabut karena reuse.
	if reason, ok := familyReason(t, env, familyID); !ok || reason != "reuse" {
		t.Fatalf("revoked_reason = %q (found=%v), want 'reuse'", reason, ok)
	}

	// Dua baris token: token login + satu hasil rotasi yang menang.
	count, ok := dbOne(t, env, "SELECT COUNT(*)::text FROM refresh_tokens WHERE family_id = $1", familyID)
	if !ok {
		t.Fatal("keluarga tidak ditemukan")
	}
	if count != "2" {
		t.Fatalf("jumlah baris refresh_tokens = %s, want 2", count)
	}
}

// D-I4: refresh vs logout paralel, diulang 20 kali — tidak ada 5xx, dan
// setelah keduanya selesai, refresh dengan token asli selalu 401.
func TestDI4_RefreshVsLogoutParallel(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	for round := 0; round < 20; round++ {
		login := registerAndLogin(t, env, "di4")

		codes := make([]int, 2)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			codes[0], errs[0] = probeJSON(http.MethodPost, api+"/auth/refresh", "", map[string]string{
				"refresh_token": login.RefreshToken,
			}, nil)
		}()
		go func() {
			defer wg.Done()
			codes[1], errs[1] = probeJSON(http.MethodPost, api+"/auth/logout", login.AccessToken, map[string]string{
				"refresh_token": login.RefreshToken,
			}, nil)
		}()
		wg.Wait()

		for i, code := range codes {
			if errs[i] != nil {
				t.Fatalf("ronde %d: request %d gagal: %v", round, i, errs[i])
			}
			if code >= 500 {
				t.Fatalf("ronde %d: request %d = %d, tidak boleh 5xx", round, i, code)
			}
		}

		// Apa pun pemenangnya (refresh sukses atau logout mencabut keluarga),
		// refresh dengan token asli setelahnya selalu 401.
		if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
			"refresh_token": login.RefreshToken,
		}, nil); code != http.StatusUnauthorized {
			t.Fatalf("ronde %d: refresh terakhir = %d, want 401", round, code)
		}
	}
}

// D-I5: logout ownership — refresh token milik orang lain + Bearer sendiri
// → 403 dan keluarga tetap aktif; keluarga tak dikenal → 204; tanpa Bearer
// → 401; body kosong → 422.
func TestDI5_LogoutOwnership(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	rina := registerAndLogin(t, env, "di5a")
	other := registerAndLogin(t, env, "di5b")
	familyRina := jwtField(t, rina.RefreshToken, "fid")

	// Bearer milik orang lain, refresh token milik Rina → 403.
	if code := apiJSON(t, http.MethodPost, api+"/auth/logout", other.AccessToken, map[string]string{
		"refresh_token": rina.RefreshToken,
	}, nil); code != http.StatusForbidden {
		t.Fatalf("logout token orang lain = %d, want 403", code)
	}

	// Keluarga Rina TIDAK dicabut.
	if reason, ok := familyReason(t, env, familyRina); ok && reason != "" {
		t.Fatalf("keluarga Rina tercabut (reason=%q), harus tetap aktif", reason)
	}

	// Keluarga tak dikenal (baris keluarga dihapus via SQL) → 204.
	third := registerAndLogin(t, env, "di5c")
	familyThird := jwtField(t, third.RefreshToken, "fid")
	if n := dbExec(t, env, "DELETE FROM refresh_families WHERE id = $1", familyThird); n != 1 {
		t.Fatalf("baris keluarga terhapus = %d, want 1", n)
	}
	if code := apiJSON(t, http.MethodPost, api+"/auth/logout", third.AccessToken, map[string]string{
		"refresh_token": third.RefreshToken,
	}, nil); code != http.StatusNoContent {
		t.Fatalf("logout keluarga tak dikenal = %d, want 204", code)
	}

	// Tanpa Bearer → 401.
	if code := apiJSON(t, http.MethodPost, api+"/auth/logout", "", map[string]string{
		"refresh_token": rina.RefreshToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("logout tanpa Bearer = %d, want 401", code)
	}

	// Body kosong → 422.
	if code := apiJSON(t, http.MethodPost, api+"/auth/logout", other.AccessToken, map[string]string{}, nil); code != http.StatusUnprocessableEntity {
		t.Fatalf("logout tanpa refresh_token = %d, want 422", code)
	}
}

// D-I6: cross use — refresh token sebagai Bearer di /auth/me → 401;
// access token di /auth/refresh → 401.
func TestDI6_CrossUseRejected(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di6")

	if code := apiJSON(t, http.MethodGet, api+"/auth/me", login.RefreshToken, nil, nil); code != http.StatusUnauthorized {
		t.Fatalf("me dengan refresh token = %d, want 401", code)
	}
	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.AccessToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("refresh dengan access token = %d, want 401", code)
	}
}

// D-I7: access token masih 200 di /auth/me setelah logout, sampai exp
// (dokumentasi limitasi: tidak ada denylist).
func TestDI7_AccessTokenValidUntilExpiryAfterLogout(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di7")

	if code := apiJSON(t, http.MethodPost, api+"/auth/logout", login.AccessToken, map[string]string{
		"refresh_token": login.RefreshToken,
	}, nil); code != http.StatusNoContent {
		t.Fatalf("logout = %d, want 204", code)
	}

	// Access token lama masih diterima meskipun keluarga sudah dicabut.
	var me model.User
	if code := apiJSON(t, http.MethodGet, api+"/auth/me", login.AccessToken, nil, &me); code != http.StatusOK {
		t.Fatalf("me setelah logout = %d, want 200 (limitasi terdokumentasi)", code)
	}
}

// D-I8: expiry dari DB — expires_at token dimundurkan lewat SQL → refresh
// 401 dan keluarga TIDAK dicabut.
func TestDI8_ExpiredTokenDoesNotRevokeFamily(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di8")
	familyID := jwtField(t, login.RefreshToken, "fid")
	jti := jwtField(t, login.RefreshToken, "jti")

	// Mundurkan exp token ini saja ke masa lalu.
	if n := dbExec(t, env, "UPDATE refresh_tokens SET expires_at = now() - interval '1 hour' WHERE jti = $1", jti); n != 1 {
		t.Fatalf("baris token ter-update = %d, want 1", n)
	}

	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.RefreshToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("refresh token kedaluwarsa = %d, want 401", code)
	}

	// Keluarga tidak boleh ikut dicabut.
	if reason, ok := familyReason(t, env, familyID); ok && reason != "" {
		t.Fatalf("keluarga tercabut (reason=%q), harus tetap aktif", reason)
	}
}

// D-I9: unknown token — baris token dihapus lewat SQL → refresh 401 dan
// keluarga tidak dicabut.
func TestDI9_UnknownTokenDoesNotRevokeFamily(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	login := registerAndLogin(t, env, "di9")
	familyID := jwtField(t, login.RefreshToken, "fid")
	jti := jwtField(t, login.RefreshToken, "jti")

	if n := dbExec(t, env, "DELETE FROM refresh_tokens WHERE jti = $1", jti); n != 1 {
		t.Fatalf("baris token terhapus = %d, want 1", n)
	}

	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.RefreshToken,
	}, nil); code != http.StatusUnauthorized {
		t.Fatalf("refresh token tak dikenal = %d, want 401", code)
	}

	if reason, ok := familyReason(t, env, familyID); ok && reason != "" {
		t.Fatalf("keluarga tercabut (reason=%q), harus tetap aktif", reason)
	}
}

// D-I10: admin mengubah role → access token baru hasil refresh membawa role baru.
func TestDI10_RoleChangeReflectedOnRefresh(t *testing.T) {
	env := testutil.Require(t)
	testutil.Canary(t, env)
	api := env.APIURL + "/api/v1"

	email := testutil.Unique("di10") + "@example.test"

	var registered model.User
	if code := apiJSON(t, http.MethodPost, api+"/auth/register", "", map[string]string{
		"name": "IT di10", "email": email, "password": "rahasia123",
	}, &registered); code != http.StatusCreated {
		t.Fatalf("register = %d, want 201", code)
	}

	var login model.TokenResponse
	if code := apiJSON(t, http.MethodPost, api+"/auth/login", "", map[string]string{
		"email": email, "password": "rahasia123",
	}, &login); code != http.StatusOK {
		t.Fatalf("login = %d, want 200", code)
	}

	// Admin mempromosikan user menjadi pengelola.
	var adminLogin model.TokenResponse
	if code := apiJSON(t, http.MethodPost, api+"/auth/login", "", map[string]string{
		"email": env.AdminEmail, "password": env.AdminPassword,
	}, &adminLogin); code != http.StatusOK {
		t.Fatalf("login admin = %d, want 200", code)
	}
	if code := apiJSON(t, http.MethodPatch, fmt.Sprintf("%s/users/%d/role", api, registered.ID), adminLogin.AccessToken, map[string]string{
		"role": "pengelola",
	}, nil); code != http.StatusOK {
		t.Fatalf("change role = %d, want 200", code)
	}

	// Access token lama membawa role lama. Bukti via refresh: rotasi tetap
	// 200, lalu access token BARU membawa role terbaru.
	var rotated model.TokenResponse
	if code := apiJSON(t, http.MethodPost, api+"/auth/refresh", "", map[string]string{
		"refresh_token": login.RefreshToken,
	}, &rotated); code != http.StatusOK {
		t.Fatalf("refresh = %d, want 200", code)
	}

	var me model.User
	if code := apiJSON(t, http.MethodGet, api+"/auth/me", rotated.AccessToken, nil, &me); code != http.StatusOK {
		t.Fatalf("me setelah refresh = %d, want 200", code)
	}
	if me.Role != "pengelola" {
		t.Fatalf("role di access token baru = %q, want 'pengelola'", me.Role)
	}
}
