package helper

import (
	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
)

// Success mengirim response envelope sukses dengan status yang diberikan.
func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Created mengirim 201 dan header Location bila location tidak kosong.
func Created(c *fiber.Ctx, message string, data any, location string) error {
	if location != "" {
		c.Set(fiber.HeaderLocation, location)
	}
	return Success(c, fiber.StatusCreated, message, data)
}

// NoContent mengirim 204 tanpa body.
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}
