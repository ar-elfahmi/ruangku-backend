package repository

import (
	"context"
	"strconv"
	"strings"

	"ruangku-backend/app/model"
)

const historyColumns = `
	b.id, b.user_id, b.room_id, b.purpose, b.start_time, b.end_time, b.status, b.created_at,
	r.code, r.name, u.name
`

type historyRepository struct {
	db DBTX
}

func NewHistoryRepository(db DBTX) HistoryRepository {
	return &historyRepository{db: db}
}

func (r *historyRepository) FindBookings(ctx context.Context, f model.HistoryFilter) ([]model.BookingDetail, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if f.UserID != nil {
		conditions = append(conditions, "b.user_id = $"+strconv.Itoa(argIdx))
		args = append(args, *f.UserID)
		argIdx++
	}

	if f.RoomID != nil {
		conditions = append(conditions, "b.room_id = $"+strconv.Itoa(argIdx))
		args = append(args, *f.RoomID)
		argIdx++
	}

	if f.Status != "" {
		conditions = append(conditions, "b.status = $"+strconv.Itoa(argIdx))
		args = append(args, f.Status)
		argIdx++
	}

	query := `
		SELECT ` + historyColumns + `
		FROM bookings b
		JOIN rooms r ON r.id = b.room_id
		JOIN users u ON u.id = b.user_id
	`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY b.start_time DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, TranslateError(err)
	}
	defer rows.Close()

	items := make([]model.BookingDetail, 0)
	for rows.Next() {
		var item model.BookingDetail
		err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.RoomID,
			&item.Purpose,
			&item.StartTime,
			&item.EndTime,
			&item.Status,
			&item.CreatedAt,
			&item.RoomCode,
			&item.RoomName,
			&item.UserName,
		)
		if err != nil {
			return nil, TranslateError(err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, TranslateError(err)
	}

	return items, nil
}

var _ HistoryRepository = (*historyRepository)(nil)
