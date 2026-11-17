package service

import (
	"errors"
	"strings"
	"time"
)

// ErrInvalidDate terjadi ketika query tanggal tidak berformat YYYY-MM-DD.
var ErrInvalidDate = errors.New("format tanggal harus YYYY-MM-DD")

// NormalizeRoomCode memotong spasi di awal/akhir dan mengubah kode menjadi
// huruf besar agar "lab-01" dan "LAB-01" dianggap kode yang sama.
func NormalizeRoomCode(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// ParseDay mengubah "YYYY-MM-DD" menjadi rentang satu hari penuh
// [00:00, 00:00 besok) pada zona waktu loc.
func ParseDay(s string, loc *time.Location) (time.Time, time.Time, error) {
	start, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, time.Time{}, ErrInvalidDate
	}
	return start, start.AddDate(0, 0, 1), nil
}
