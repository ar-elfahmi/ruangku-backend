package service

import (
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type sessionService struct {
	repo    repository.RefreshTokenRepository
	refresh *helper.RefreshJWTManager
	access  *helper.JWTManager
	users   repository.UserRepository
}

func NewSessionService(r repository.RefreshTokenRepository, refresh *helper.RefreshJWTManager, access *helper.JWTManager, u repository.UserRepository) SessionService {
	return &sessionService{repo: r, refresh: refresh, access: access, users: u}
}

// Refresh menangani POST /auth/refresh.
func (s *sessionService) Refresh(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

// Logout menangani POST /auth/logout.
func (s *sessionService) Logout(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

var _ SessionService = (*sessionService)(nil)