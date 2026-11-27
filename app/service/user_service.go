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

// ChangeRole mengubah role user; hanya admin.
//
//	@Summary		Ubah role user
//	@Tags			users
//	@Security		BearerAuth
//	@Param			id		path	int						true	"ID user"
//	@Param			request	body	model.ChangeRoleRequest	true	"Role baru"
//	@Success		200		{object}	model.WebResponse{data=model.User}
//	@Failure		400		{object}	model.WebResponse
//	@Failure		401		{object}	model.WebResponse
//	@Failure		403		{object}	model.WebResponse
//	@Failure		404		{object}	model.WebResponse
//	@Failure		422		{object}	model.WebResponse
//	@Router			/users/{id}/role [patch]
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
