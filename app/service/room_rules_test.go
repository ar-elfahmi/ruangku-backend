package service

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeRoomCode(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"spasi dan huruf kecil", " lab-01 ", "LAB-01"},
		{"sudah huruf besar", "AULA", "AULA"},
		{"tanpa perubahan", "lab-01", "LAB-01"},
		{"kode dengan spasi di tengah", " LAB- 01 ", "LAB- 01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeRoomCode(tt.input); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseDay(t *testing.T) {
	// FixedZone agar test tidak bergantung pada data tzdata sistem.
	loc := time.FixedZone("WIB", 7*60*60)

	tests := []struct {
		name      string
		input     string
		wantStart time.Time
		wantEnd   time.Time
		wantErr   error
	}{
		{
			"tanggal valid",
			"2026-12-01",
			time.Date(2026, 12, 1, 0, 0, 0, 0, loc),
			time.Date(2026, 12, 2, 0, 0, 0, 0, loc),
			nil,
		},
		{"kosong", "", time.Time{}, time.Time{}, ErrInvalidDate},
		{"format dd-mm-yyyy", "01-12-2026", time.Time{}, time.Time{}, ErrInvalidDate},
		{"bulan tidak ada", "2026-13-01", time.Time{}, time.Time{}, ErrInvalidDate},
		{"bukan angka", "xyz", time.Time{}, time.Time{}, ErrInvalidDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := ParseDay(tt.input, loc)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if !start.Equal(tt.wantStart) || !end.Equal(tt.wantEnd) {
				t.Fatalf("got [%v, %v), want [%v, %v)", start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
