package service

import (
	"github.com/gofiber/fiber/v2"

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
	return helper.NotImplemented("fitur belum tersedia")
}

var _ UserService = (*userService)(nil)
