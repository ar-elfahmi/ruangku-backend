package testutil

import (
	"errors"
	"testing"
)

func validVars() map[string]string {
	return map[string]string{
		"TEST_API_URL":        "http://localhost:8081",
		"TEST_DB_URL":         "postgres://ruangku:ruangku@localhost:5432/ruangku_test?sslmode=disable",
		"TEST_REDIS_ADDR":     "127.0.0.1:6379",
		"TEST_REDIS_DB":       "15",
		"TEST_MONGO_URI":      "mongodb://127.0.0.1:27017",
		"TEST_MONGO_DB":       "ruangku_test",
		"TEST_ADMIN_EMAIL":    "admin@ruangku.local",
		"TEST_ADMIN_PASSWORD": "admin12345",
	}
}

func lookup(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestLoadEnvValid(t *testing.T) {
	env, err := LoadEnv(lookup(validVars()))
	if err != nil {
		t.Fatalf("env valid ditolak: %v", err)
	}
	if env.RedisDB != 15 || env.MongoDB != "ruangku_test" || env.APIURL != "http://localhost:8081" {
		t.Fatalf("hasil parse salah: %+v", env)
	}
}

func TestLoadEnvNotConfigured(t *testing.T) {
	for key := range validVars() {
		t.Run("kosong "+key, func(t *testing.T) {
			vars := validVars()
			vars[key] = ""
			if _, err := LoadEnv(lookup(vars)); !errors.Is(err, ErrNotConfigured) {
				t.Fatalf("variabel %s kosong harus ErrNotConfigured, dapat %v", key, err)
			}
		})
	}
}

func TestLoadEnvRefusesUnsafeTarget(t *testing.T) {
	tests := []struct {
		name, key, value string
	}{
		{"database postgres bukan _test", "TEST_DB_URL", "postgres://ruangku:ruangku@localhost:5432/ruangku?sslmode=disable"},
		{"database postgres tanpa nama", "TEST_DB_URL", "postgres://ruangku:ruangku@localhost:5432/"},
		{"url postgres tidak valid", "TEST_DB_URL", "://bukan-url"},
		{"database mongo bukan _test", "TEST_MONGO_DB", "ruangku"},
		{"redis db 0", "TEST_REDIS_DB", "0"},
		{"redis db bukan angka", "TEST_REDIS_DB", "lima-belas"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := validVars()
			vars[tt.key] = tt.value
			_, err := LoadEnv(lookup(vars))
			if !errors.Is(err, ErrUnsafeTarget) {
				t.Fatalf("harus ErrUnsafeTarget, dapat %v", err)
			}
		})
	}
}
