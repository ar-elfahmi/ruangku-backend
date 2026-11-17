package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"ruangku-backend/app/model"
)

const roomColumns = "id, code, name, building, capacity, is_active, created_at"

type roomRepository struct {
	db DBTX
}

func NewRoomRepository(db DBTX) RoomRepository {
	return &roomRepository{db: db}
}

func scanRoom(row pgx.Row) (model.Room, error) {
	var room model.Room
	err := row.Scan(&room.ID, &room.Code, &room.Name, &room.Building, &room.Capacity, &room.IsActive, &room.CreatedAt)
	return room, err
}

// FindAllActive mengembalikan semua ruang aktif diurutkan berdasarkan kode.
func (r *roomRepository) FindAllActive(ctx context.Context) ([]model.Room, error) {
	rows, err := r.db.Query(ctx, "SELECT "+roomColumns+" FROM rooms WHERE is_active = true ORDER BY code")
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()

	rooms := []model.Room{}
	for rows.Next() {
		room, err := scanRoom(rows)
		if err != nil {
			return nil, TranslateError(err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, TranslateError(err)
	}
	return rooms, nil
}

// FindByID mengembalikan ruang berdasarkan id, termasuk yang tidak aktif
// agar riwayat booking ruang nonaktif tetap bisa dibaca.
func (r *roomRepository) FindByID(ctx context.Context, id int) (model.Room, error) {
	row := r.db.QueryRow(ctx, "SELECT "+roomColumns+" FROM rooms WHERE id = $1", id)
	room, err := scanRoom(row)
	if err != nil {
		return model.Room{}, TranslateError(err)
	}
	return room, nil
}

// Create menyimpan ruang baru. is_active dan created_at memakai default dari
// tabel; kode duplikat ditolak oleh unique constraint (ErrDuplicate).
func (r *roomRepository) Create(ctx context.Context, room model.Room) (model.Room, error) {
	row := r.db.QueryRow(ctx,
		"INSERT INTO rooms (code, name, building, capacity) VALUES ($1, $2, $3, $4) RETURNING "+roomColumns,
		room.Code, room.Name, room.Building, room.Capacity)
	created, err := scanRoom(row)
	if err != nil {
		return model.Room{}, TranslateError(err)
	}
	return created, nil
}

// Update mengganti seluruh field ruang. 0 baris berarti id tidak ada
// (pgx.ErrNoRows diterjemahkan menjadi ErrNotFound).
func (r *roomRepository) Update(ctx context.Context, room model.Room) (model.Room, error) {
	row := r.db.QueryRow(ctx,
		"UPDATE rooms SET code = $2, name = $3, building = $4, capacity = $5 WHERE id = $1 RETURNING "+roomColumns,
		room.ID, room.Code, room.Name, room.Building, room.Capacity)
	updated, err := scanRoom(row)
	if err != nil {
		return model.Room{}, TranslateError(err)
	}
	return updated, nil
}

// SoftDelete menonaktifkan ruang tanpa menghapus barisnya agar riwayat
// booking tetap utuh. Kondisi is_active menjadikan query guarded: 0 baris
// berarti id tidak ada atau ruang sudah nonaktif.
func (r *roomRepository) SoftDelete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, "UPDATE rooms SET is_active = false WHERE id = $1 AND is_active", id)
	if err != nil {
		return TranslateError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindBookedSlots mengembalikan slot pending/approved yang menyentuh rentang
// [dayStart, dayEnd), diurutkan berdasarkan waktu mulai.
func (r *roomRepository) FindBookedSlots(ctx context.Context, roomID int, dayStart, dayEnd time.Time) ([]model.TimeSlot, error) {
	rows, err := r.db.Query(ctx,
		"SELECT booking_id, start_time, end_time, status FROM bookings WHERE room_id = $1 AND status IN ('pending', 'approved') AND start_time < $3 AND end_time > $2 ORDER BY start_time",
		roomID, dayStart, dayEnd)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()

	slots := []model.TimeSlot{}
	for rows.Next() {
		var slot model.TimeSlot
		if err := rows.Scan(&slot.BookingID, &slot.StartTime, &slot.EndTime, &slot.Status); err != nil {
			return nil, TranslateError(err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, TranslateError(err)
	}
	return slots, nil
}

var _ RoomRepository = (*roomRepository)(nil)
