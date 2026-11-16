package service

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
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
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	hash, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	// Role tidak pernah dibaca dari body: registrasi selalu menjadi peminjam.
	user := model.User{
		Name:         req.Name,
		Email:        NormalizeEmail(req.Email),
		PasswordHash: hash,
		Role:         model.RolePeminjam,
	}
	created, err := s.users.Create(ctx, user)
	switch {
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("email sudah terdaftar")
	case err != nil:
		return helper.Internal(err)
	}

	return helper.Created(c, "registrasi berhasil", created, "")
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
