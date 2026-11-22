package service

import (
	"errors"
	"strings"

	"ruangku-backend/app/model"
)

var ErrInvalidStatus = errors.New("status tidak dikenal")

func ParseStatusFilter(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", nil
	}

	lower := strings.ToLower(trimmed)
	switch lower {
	case model.StatusPending, model.StatusApproved, model.StatusRejected, model.StatusCancelled:
		return lower, nil
	default:
		return "", ErrInvalidStatus
	}
}
