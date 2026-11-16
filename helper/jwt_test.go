package helper

import (
	"testing"
	"time"
)

const testSecret = "secret-untuk-test-minimal-32-karakter!!"

func TestJWTRoundTrip(t *testing.T) {
	manager := NewJWTManager(testSecret, time.Hour)

	token, err := manager.Generate(7, "a@b.c", "admin")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	user, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if user.ID != 7 || user.Email != "a@b.c" || user.Role != "admin" {
		t.Errorf("Parse = %+v, want ID=7 Email=a@b.c Role=admin", user)
	}
}

func TestJWTParseInvalid(t *testing.T) {
	manager := NewJWTManager(testSecret, time.Hour)
	other := NewJWTManager("secret-lain-yang-juga-32-karakter!!!!", time.Hour)
	expired := NewJWTManager(testSecret, -time.Minute)

	foreign, _ := other.Generate(1, "a@b.c", "peminjam")
	old, _ := expired.Generate(1, "a@b.c", "peminjam")

	tests := []struct {
		name  string
		token string
	}{
		{"secret berbeda", foreign},
		{"sudah kedaluwarsa", old},
		{"bukan token", "bukan-token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := manager.Parse(tt.token); err == nil {
				t.Error("Parse harus mengembalikan error")
			}
		})
	}
}

func TestJWTTTLSeconds(t *testing.T) {
	if got := NewJWTManager(testSecret, 90*time.Minute).TTLSeconds(); got != 5400 {
		t.Errorf("TTLSeconds = %d, want 5400", got)
	}
}
