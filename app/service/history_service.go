package service

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type historyService struct {
	history repository.HistoryRepository
}

func NewHistoryService(h repository.HistoryRepository) HistoryService {
	return &historyService{history: h}
}

// Mine menangani GET /api/v1/bookings/mine?status=
// Menampilkan riwayat peminjaman milik peminjam yang sedang login.
//
//	@Summary		Riwayat peminjaman milik peminjam
//	@Tags			bookings
//	@Security		BearerAuth
//	@Param			status	query	string	false	"Filter status: pending, approved, rejected, cancelled"
//	@Success		200		{object}	model.WebResponse{data=[]model.Booking}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Failure		403		{object}	model.WebResponse
//	@Router			/bookings/mine [get]
func (s *historyService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("token tidak valid atau sudah kedaluwarsa")
	}

	status, err := ParseStatusFilter(c.Query("status"))
	if err != nil {
		return helper.BadRequest("status tidak dikenal")
	}

	filter := model.HistoryFilter{
		UserID: &user.ID,
		Status: status,
	}

	bookings, err := s.history.FindBookings(ctx, filter)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "riwayat peminjaman berhasil diambil", bookings)
}

// ListAll menangani GET /api/v1/bookings?status=&room_id=
// Menampilkan seluruh riwayat peminjaman untuk pengelola dengan filter opsional status dan room_id.
func (s *historyService) ListAll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	status, err := ParseStatusFilter(c.Query("status"))
	if err != nil {
		return helper.BadRequest("status tidak dikenal")
	}

	var roomIDPtr *int
	if roomIDStr := c.Query("room_id"); roomIDStr != "" {
		roomID, err := strconv.Atoi(roomIDStr)
		if err != nil || roomID <= 0 {
			return helper.BadRequest("room_id tidak valid")
		}
		roomIDPtr = &roomID
	}

	filter := model.HistoryFilter{
		RoomID: roomIDPtr,
		Status: status,
	}

	bookings, err := s.history.FindBookings(ctx, filter)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "daftar peminjaman berhasil diambil", bookings)
}

var _ HistoryService = (*historyService)(nil)
