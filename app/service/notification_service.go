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
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	onlyUnread, err := ParseUnreadFilter(c.Query("is_read"))
	if err != nil {
		return err
	}

	notifications, err := s.notifications.FindByUser(ctx, user.ID, onlyUnread, notificationListLimit)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, messageNotificationList, notifications)
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
