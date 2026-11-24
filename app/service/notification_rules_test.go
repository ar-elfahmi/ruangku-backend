package service

import (
	"testing"

	"ruangku-backend/helper"
)

func TestParseUnreadFilter(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantUnread bool
		wantStatus int
	}{
		{name: "kosong berarti semua", raw: "", wantUnread: false},
		{name: "false berarti belum dibaca", raw: "false", wantUnread: true},
		{name: "true tidak didukung", raw: "true", wantStatus: 400},
		{name: "nilai acak", raw: "abc", wantStatus: 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUnreadFilter(tt.raw)
			if tt.wantStatus != 0 {
				if err == nil {
					t.Fatalf("ParseUnreadFilter(%q) tidak error", tt.raw)
				}
				if appErr, ok := err.(*helper.AppError); !ok || appErr.Status != tt.wantStatus {
					t.Fatalf("status error = %v, mau %d", err, tt.wantStatus)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseUnreadFilter(%q) error: %v", tt.raw, err)
			}
			if got != tt.wantUnread {
				t.Fatalf("ParseUnreadFilter(%q) = %v, mau %v", tt.raw, got, tt.wantUnread)
			}
		})
	}
}

func TestIsNotificationID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{name: "object id valid", id: "64b7f0c2a1b2c3d4e5f60718", want: true},
		{name: "huruf besar", id: "64B7F0C2A1B2C3D4E5F60718", want: true},
		{name: "terlalu pendek", id: "64b7f0c2", want: false},
		{name: "bukan hex", id: "zzzzzzzzzzzzzzzzzzzzzzzz", want: false},
		{name: "kosong", id: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotificationID(tt.id); got != tt.want {
				t.Fatalf("IsNotificationID(%q) = %v, mau %v", tt.id, got, tt.want)
			}
		})
	}
}
