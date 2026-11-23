package helper

import (
	"errors"
	"time"
)

// Error sentinel untuk refresh token. ErrRefreshExpired dan ErrRefreshInvalid
// dipisah agar log dan test bisa membedakan keduanya.
var (
	ErrRefreshExpired = errors.New("refresh token kedaluwarsa")
	ErrRefreshInvalid = errors.New("refresh token tidak valid")
)

// RefreshClaims adalah klaim di dalam refresh JWT. Semua klaim wajib ada.
type RefreshClaims struct {
	UserID    int       // sub
	TokenID   string    // jti
	FamilyID  string    // fid
	ExpiresAt time.Time // exp
}

type RefreshJWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewRefreshJWTManager(secret string, ttl time.Duration) *RefreshJWTManager {
	return &RefreshJWTManager{secret: []byte(secret), ttl: ttl}
}

// Generate membuat refresh JWT HS256 dengan klaim sub, jti, fid, typ, exp.
func (m *RefreshJWTManager) Generate(userID int, familyID, tokenID string) (string, error) {
	return "", errors.New("belum diimplementasi")
}

// Parse memvalidasi signature, exp, dan seluruh klaim wajib.
func (m *RefreshJWTManager) Parse(token string) (RefreshClaims, error) {
	return RefreshClaims{}, ErrRefreshInvalid
}

// ParseAllowExpired dipakai logout: signature harus valid, exp boleh lewat.
func (m *RefreshJWTManager) ParseAllowExpired(token string) (RefreshClaims, error) {
	return RefreshClaims{}, ErrRefreshInvalid
}

// TTLSeconds mengembalikan masa berlaku refresh token dalam detik.
func (m *RefreshJWTManager) TTLSeconds() int {
	return 0
}

// ValidateJWTSecrets menolak secret yang kosong, pendek, atau sama.
func ValidateJWTSecrets(access, refresh string) error {
	return errors.New("belum diimplementasi")
}