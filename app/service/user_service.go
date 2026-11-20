package service

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

type userService struct {
	users repository.UserRepository
}

func NewUserService(u repository.UserRepository) UserService {
	return &userService{users: u}
}

func (s *userService) ChangeRole(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, ok := helper.ParamID(c, "id")
	if !ok {
		return helper.BadRequest("id tidak valid")
	}

	var req model.ChangeRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body request tidak valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	user, err := s.users.UpdateRole(ctx, id, req.Role)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("user tidak ditemukan")
	case err != nil:
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "role user berhasil diubah", user)
}

var _ UserService = (*userService)(nil)
