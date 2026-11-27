package service

import (
	"errors"

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

// List mengembalikan notifikasi milik user yang sedang login, terbaru dulu.
//
//	@Summary		Daftar notifikasi
//	@Tags			notifications
//	@Security		BearerAuth
//	@Param			is_read	query	string	false	"Isi false untuk hanya menampilkan yang belum dibaca"
//	@Success		200		{object}	model.WebResponse{data=[]model.Notification}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Router			/notifications [get]
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

// UnreadCount mengembalikan jumlah notifikasi belum dibaca milik pemanggil.
//
//	@Summary		Jumlah notifikasi belum dibaca
//	@Tags			notifications
//	@Security		BearerAuth
//	@Success		200	{object}	model.WebResponse{data=map[string]int}
//	@Failure		401	{object}	model.WebResponse
//	@Router			/notifications/unread-count [get]
func (s *notificationService) UnreadCount(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	count, err := s.notifications.CountUnread(ctx, user.ID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, messageNotificationUnread, fiber.Map{"count": count})
}

// MarkRead menandai satu notifikasi milik pemanggil sebagai sudah dibaca.
//
//	@Summary		Tandai notifikasi sudah dibaca
//	@Tags			notifications
//	@Security		BearerAuth
//	@Param			id	path	string	true	"ObjectID notifikasi"
//	@Success		200	{object}	model.WebResponse{data=model.Notification}
//	@Failure		400	{object}	model.WebResponse
//	@Failure		401	{object}	model.WebResponse
//	@Failure		404	{object}	model.WebResponse
//	@Router			/notifications/{id}/read [patch]
func (s *notificationService) MarkRead(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	id := c.Params("id")
	if !IsNotificationID(id) {
		return helper.BadRequest(messageNotificationIDInvalid)
	}

	notification, err := s.notifications.UpdateRead(ctx, user.ID, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound(messageNotificationNotFound)
		}
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, messageNotificationRead, notification)
}

// MarkAllRead menandai semua notifikasi milik pemanggil sebagai sudah dibaca.
//
//	@Summary		Tandai semua notifikasi sudah dibaca
//	@Tags			notifications
//	@Security		BearerAuth
//	@Success		200	{object}	model.WebResponse{data=map[string]int}
//	@Failure		401	{object}	model.WebResponse
//	@Router			/notifications/read-all [patch]
func (s *notificationService) MarkAllRead(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	updated, err := s.notifications.UpdateAllRead(ctx, user.ID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, messageNotificationReadAll, fiber.Map{"updated": updated})
}

var _ NotificationService = (*notificationService)(nil)
