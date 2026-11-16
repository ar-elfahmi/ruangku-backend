package helper

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("rahasia123")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"password benar", "rahasia123", true},
		{"password salah", "salah", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPassword(hash, tt.password); got != tt.want {
				t.Errorf("CheckPassword(%q) = %v, want %v", tt.password, got, tt.want)
			}
		})
	}
}
