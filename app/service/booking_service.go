package service

import (
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type bookingService struct {
	bookings repository.BookingRepository
	rooms    repository.RoomRepository
	tx       *repository.TxManager
}

func NewBookingService(b repository.BookingRepository, r repository.RoomRepository, tx *repository.TxManager) BookingService {
	return &bookingService{bookings: b, rooms: r, tx: tx}
}

func (s *bookingService) Create(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *bookingService) Cancel(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *bookingService) Approve(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *bookingService) Reject(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *bookingService) Logs(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

var _ BookingService = (*bookingService)(nil)
