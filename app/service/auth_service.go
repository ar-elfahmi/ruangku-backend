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
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	user, err := s.users.FindByEmail(ctx, NormalizeEmail(req.Email))
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Unauthorized(messageWrongCredentials)
	case err != nil:
		return helper.Internal(err)
	}

	if !helper.CheckPassword(user.PasswordHash, req.Password) {
		return helper.Unauthorized(messageWrongCredentials)
	}

	token, err := s.jwt.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "login berhasil", model.TokenResponse{
		AccessToken: token,
		ExpiresIn:   s.jwt.TTLSeconds(),
	})
}

func (s *authService) Me(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	authUser, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	user, err := s.users.FindByID(ctx, authUser.ID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Unauthorized(messageInvalidToken)
	case err != nil:
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "profil berhasil diambil", user)
}

func (s *authService) EnsureAdmin(ctx context.Context, name, email, password string) error {
	email = NormalizeEmail(email)
	if email == "" || password == "" {
		return nil
	}

	_, err := s.users.FindByEmail(ctx, email)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, repository.ErrNotFound):
		return err
	}

	hash, err := helper.HashPassword(password)
	if err != nil {
		return err
	}

	_, err = s.users.Create(ctx, model.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         model.RoleAdmin,
	})
	if errors.Is(err, repository.ErrDuplicate) {
		return nil
	}
	return err
}

var _ AuthService = (*authService)(nil)
