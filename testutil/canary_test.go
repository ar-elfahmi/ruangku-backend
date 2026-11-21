package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterCanarySendsRegisterRequest(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	if err := registerCanary(context.Background(), srv.URL, "it-canary@example.test", "rahasia123"); err != nil {
		t.Fatalf("registerCanary gagal: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/auth/register" {
		t.Fatalf("request salah: %s %s", gotMethod, gotPath)
	}
	if gotBody["email"] != "it-canary@example.test" || gotBody["password"] != "rahasia123" || gotBody["name"] == "" {
		t.Fatalf("body salah: %v", gotBody)
	}
}

func TestRegisterCanaryRejectsUnexpectedStatus(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusConflict, http.StatusInternalServerError} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		if err := registerCanary(context.Background(), srv.URL, "x@example.test", "rahasia123"); err == nil {
			t.Fatalf("status %d harus dianggap gagal", status)
		}
		srv.Close()
	}
}

func TestRegisterCanaryAPIUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	if err := registerCanary(context.Background(), url, "x@example.test", "rahasia123"); err == nil {
		t.Fatal("API yang mati harus menghasilkan error")
	}
}
