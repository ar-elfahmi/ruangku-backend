package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound       = errors.New("data tidak ditemukan")
	ErrDuplicate      = errors.New("data sudah ada")
	ErrOverlap        = errors.New("jadwal bentrok")
	ErrConflict       = errors.New("data berubah oleh proses lain")
	ErrNotImplemented = errors.New("fitur belum tersedia")
)

const (
	pgUniqueViolation    = "23505"
	pgExclusionViolation = "23P01"
)

// TranslateError mengubah error pgx menjadi error repository. Error yang
// tidak dikenal dikembalikan apa adanya.
func TranslateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation:
			return ErrDuplicate
		case pgExclusionViolation:
			return ErrOverlap
		}
	}
	return err
}
