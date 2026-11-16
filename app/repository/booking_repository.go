package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ruangku-backend/app/model"
)

const (
	bookingColumns    = "id, user_id, room_id, purpose, start_time, end_time, status, created_at"
	bookingLogColumns = "id, booking_id, actor_id, from_status, to_status, note, created_at"
)

type bookingRepository struct {
	db DBTX
}

func NewBookingRepository(db DBTX) BookingRepository {
	return &bookingRepository{db: db}
}

// WithTx mengembalikan salinan repository yang berjalan di dalam transaksi tx.
func (r *bookingRepository) WithTx(tx DBTX) BookingRepository {
	return &bookingRepository{db: tx}
}

func scanBooking(row pgx.Row) (model.Booking, error) {
	var b model.Booking
	err := row.Scan(&b.ID, &b.UserID, &b.RoomID, &b.Purpose, &b.StartTime, &b.EndTime, &b.Status, &b.CreatedAt)
	return b, err
}

// Create menyimpan booking baru. Status memakai default 'pending' dari tabel;
// tabrakan jadwal ditolak oleh exclusion constraint (ErrOverlap).
func (r *bookingRepository) Create(ctx context.Context, b model.Booking) (model.Booking, error) {
	row := r.db.QueryRow(ctx,
		"INSERT INTO bookings (user_id, room_id, purpose, start_time, end_time) VALUES ($1, $2, $3, $4, $5) RETURNING "+bookingColumns,
		b.UserID, b.RoomID, b.Purpose, b.StartTime, b.EndTime)
	created, err := scanBooking(row)
	if err != nil {
		return model.Booking{}, TranslateError(err)
	}
	return created, nil
}

func (r *bookingRepository) FindByID(ctx context.Context, id int) (model.Booking, error) {
	row := r.db.QueryRow(ctx, "SELECT "+bookingColumns+" FROM bookings WHERE id = $1", id)
	b, err := scanBooking(row)
	if err != nil {
		return model.Booking{}, TranslateError(err)
	}
	return b, nil
}

// UpdateStatus bersifat guarded: hanya mengubah bila status saat ini masih
// from. 0 baris berarti status sudah berubah oleh proses lain (ErrConflict).
func (r *bookingRepository) UpdateStatus(ctx context.Context, id int, from, to string) (model.Booking, error) {
	row := r.db.QueryRow(ctx,
		"UPDATE bookings SET status = $3 WHERE id = $1 AND status = $2 RETURNING "+bookingColumns,
		id, from, to)
	b, err := scanBooking(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Booking{}, ErrConflict
	}
	if err != nil {
		return model.Booking{}, TranslateError(err)
	}
	return b, nil
}

func (r *bookingRepository) InsertLog(ctx context.Context, l model.BookingLog) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO booking_logs (booking_id, actor_id, from_status, to_status, note) VALUES ($1, $2, $3, $4, $5)",
		l.BookingID, l.ActorID, l.FromStatus, l.ToStatus, l.Note)
	return TranslateError(err)
}

func (r *bookingRepository) FindLogs(ctx context.Context, bookingID int) ([]model.BookingLog, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+bookingLogColumns+" FROM booking_logs WHERE booking_id = $1 ORDER BY id", bookingID)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()

	logs := []model.BookingLog{}
	for rows.Next() {
		var l model.BookingLog
		if err := rows.Scan(&l.ID, &l.BookingID, &l.ActorID, &l.FromStatus, &l.ToStatus, &l.Note, &l.CreatedAt); err != nil {
			return nil, TranslateError(err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, TranslateError(err)
	}
	return logs, nil
}

var _ BookingRepository = (*bookingRepository)(nil)
