package service

import (
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type historyService struct {
	history repository.HistoryRepository
}

func NewHistoryService(h repository.HistoryRepository) HistoryService {
	return &historyService{history: h}
}

func (s *historyService) Mine(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *historyService) ListAll(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

var _ HistoryService = (*historyService)(nil)
