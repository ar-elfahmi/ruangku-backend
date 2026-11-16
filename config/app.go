package config

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
	"ruangku-backend/helper"
)

// NewApp membuat aplikasi Fiber dengan error handler yang menulis envelope error.
func NewApp() *fiber.App {
	return fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})
}

func errorHandler(c *fiber.Ctx, err error) error {
	var appErr *helper.AppError
	if !errors.As(err, &appErr) {
		appErr = toAppError(err)
	}

	if appErr.Cause != nil {
		log.Printf("error: %s %s: %v", c.Method(), c.Path(), appErr.Cause)
	}

	resp := model.WebResponse{Success: false, Message: appErr.Message}
	if len(appErr.Fields) > 0 {
		resp.Errors = appErr.Fields
	}
	return c.Status(appErr.Status).JSON(resp)
}

// toAppError menerjemahkan error non-AppError. Route yang tidak ada menjadi 404,
// sisanya 500 agar detail internal tidak bocor ke client.
func toAppError(err error) *helper.AppError {
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Code == fiber.StatusNotFound {
		return helper.NotFound("endpoint tidak ditemukan")
	}
	return helper.Internal(err)
}
