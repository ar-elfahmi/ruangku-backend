package service

import (
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type roomService struct {
	rooms repository.RoomRepository
}

func NewRoomService(r repository.RoomRepository) RoomService {
	return &roomService{rooms: r}
}

func (s *roomService) List(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *roomService) Get(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *roomService) Create(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *roomService) Update(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *roomService) Delete(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *roomService) Availability(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

var _ RoomService = (*roomService)(nil)
