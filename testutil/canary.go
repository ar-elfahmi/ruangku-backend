package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// Canary memastikan API yang sedang berjalan memakai database test: ia
// mendaftarkan satu user unik lewat API lalu mencarinya di TEST_DB_URL. Bila
// user tidak ada di sana, API berjalan dengan konfigurasi lain dan test
// dihentikan sebelum menulis data ke database yang salah.
//
// Batasan: satu user canary tetap bisa tertulis ke database yang salah bila
// API salah konfigurasi. Canary hanya mengurangi risiko, bukan menghapusnya.
func Canary(t testing.TB, env Env) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	email := Unique("canary") + "@example.test"
	if err := registerCanary(ctx, env.APIURL, email, "rahasia123"); err != nil {
		t.Fatalf("canary: API test tidak siap di %s: %v", env.APIURL, err)
	}

	conn, err := pgx.Connect(ctx, env.DBURL)
	if err != nil {
		t.Fatalf("canary: tidak bisa terhubung ke TEST_DB_URL: %v", err)
	}
	defer conn.Close(ctx)

	var found bool
	err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users WHERE lower(email) = lower($1))", email).Scan(&found)
	if err != nil {
		t.Fatalf("canary: query ke TEST_DB_URL gagal: %v", err)
	}
	if !found {
		t.Fatalf("canary: user %s tidak ada di database test; API tidak berjalan dengan konfigurasi test (DB_URL, REDIS_DB, MONGO_DB)", email)
	}
}

func registerCanary(ctx context.Context, apiURL, email, password string) error {
	body, err := json.Marshal(map[string]string{"name": "Canary", "email": email, "password": password})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL+"/api/v1/auth/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("register mengembalikan status %d, bukan 201", resp.StatusCode)
	}
	return nil
}
