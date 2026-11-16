package service

import (
	"context"
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type authService struct {
	users repository.UserRepository
	jwt   *helper.JWTManager
}

func NewAuthService(u repository.UserRepository, jwt *helper.JWTManager) AuthService {
	return &authService{users: u, jwt: jwt}
}

func (s *authService) Register(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *authService) Login(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *authService) Me(c *fiber.Ctx) error {
	return helper.NotImplemented("fitur belum tersedia")
}

func (s *authService) EnsureAdmin(ctx context.Context, name, email, password string) error {
	return nil
}

var _ AuthService = (*authService)(nil)
