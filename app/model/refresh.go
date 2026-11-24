package model

import "time"

type RefreshFamily struct {
	ID            string // fid, 32 karakter hex
	UserID        int
	CreatedAt     time.Time
	RevokedAt     *time.Time
	RevokedReason *string // logout | reuse
}

type RefreshToken struct {
	JTI        string // token id, 32 karakter hex
	FamilyID   string
	UserID     int
	IssuedAt   time.Time
	ExpiresAt  time.Time
	UsedAt     *time.Time
	ReplacedBy *string // jti token pengganti
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}
