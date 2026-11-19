package service

import (
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

func (s *historyService) ListAll(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

var _ HistoryService = (*historyService)(nil)