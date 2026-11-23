package service

import (
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type notificationService struct {
	notifications repository.NotificationRepository
}

// NewNotificationService membuat service notifikasi. Semua data selalu
// difilter berdasarkan user yang sedang login.
func NewNotificationService(r repository.NotificationRepository) NotificationService {
	return &notificationService{notifications: r}
}

func (s *notificationService) List(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia") // 501
}

func (s *notificationService) UnreadCount(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia") // 501
}

func (s *notificationService) MarkRead(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia") // 501
}

func (s *notificationService) MarkAllRead(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia") // 501
}

var _ NotificationService = (*notificationService)(nil)
