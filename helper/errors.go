package helper

import "github.com/gofiber/fiber/v2"

const messageInternal = "terjadi kesalahan pada server"

// AppError adalah error HTTP yang diterjemahkan menjadi response error oleh config.NewApp.
type AppError struct {
	Status  int
	Message string
	Fields  map[string]string
	Cause   error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func BadRequest(message string) *AppError {
	return &AppError{Status: fiber.StatusBadRequest, Message: message}
}

func Unauthorized(message string) *AppError {
	return &AppError{Status: fiber.StatusUnauthorized, Message: message}
}

func Forbidden(message string) *AppError {
	return &AppError{Status: fiber.StatusForbidden, Message: message}
}

func NotFound(message string) *AppError {
	return &AppError{Status: fiber.StatusNotFound, Message: message}
}

func Conflict(message string) *AppError {
	return &AppError{Status: fiber.StatusConflict, Message: message}
}

func NotImplemented(message string) *AppError {
	return &AppError{Status: fiber.StatusNotImplemented, Message: message}
}

// Validation membuat error 422 dengan detail per field.
func Validation(fields map[string]string) *AppError {
	return &AppError{
		Status:  fiber.StatusUnprocessableEntity,
		Message: "data tidak valid",
		Fields:  fields,
	}
}

// Internal membungkus error tak terduga. Cause hanya masuk log, tidak dikirim ke client.
func Internal(cause error) *AppError {
	return &AppError{
		Status:  fiber.StatusInternalServerError,
		Message: messageInternal,
		Cause:   cause,
	}
}
