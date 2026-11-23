package helper

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Error sentinel untuk refresh token. ErrRefreshExpired dan ErrRefreshInvalid
// dipisah agar log dan test bisa membedakan keduanya.
var (
	ErrRefreshExpired = errors.New("refresh token kedaluwarsa")
	ErrRefreshInvalid = errors.New("refresh token tidak valid")
)

// refreshClaims adalah klaim di dalam refresh JWT. Semua klaim wajib ada.
type refreshClaims struct {
	Typ      string `json:"typ"`
	FamilyID string `json:"fid"`
	jwt.RegisteredClaims
}

// RefreshClaims adalah hasil Parse yang sudah bersih.
type RefreshClaims struct {
	UserID    int
	TokenID   string // jti
	FamilyID  string // fid
	ExpiresAt time.Time
}

type RefreshJWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewRefreshJWTManager(secret string, ttl time.Duration) *RefreshJWTManager {
	return &RefreshJWTManager{secret: []byte(secret), ttl: ttl}
}

// NewTokenID menghasilkan id acak 128-bit dalam bentuk 32 karakter hex.
// Dipakai untuk jti dan fid; tidak menambah dependency baru.
func NewTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("membuat token id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Generate membuat refresh JWT HS256 dengan klaim sub, jti, fid, typ, exp.
func (m *RefreshJWTManager) Generate(userID int, familyID, tokenID string) (string, error) {
	c := refreshClaims{
		Typ:      "refresh",
		FamilyID: familyID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			ID:        tokenID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// parse memvalidasi signature dan (bila requireExp) masa berlaku, lalu
// mengecek seluruh klaim wajib.
func (m *RefreshJWTManager) parse(token string, requireExp bool) (RefreshClaims, error) {
	var c refreshClaims
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithoutClaimsValidation(), // exp divalidasi manual agar bisa dibedakan expired vs invalid
	}
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return m.secret, nil }, opts...)
	if err != nil {
		return RefreshClaims{}, ErrRefreshInvalid
	}

	id, err := strconv.Atoi(c.Subject)
	if err != nil || id < 1 {
		return RefreshClaims{}, ErrRefreshInvalid
	}
	if c.Typ != "refresh" || c.ID == "" || c.FamilyID == "" {
		return RefreshClaims{}, ErrRefreshInvalid
	}
	if c.ExpiresAt == nil {
		return RefreshClaims{}, ErrRefreshInvalid
	}

	expiresAt := c.ExpiresAt.Time
	if requireExp && time.Now().After(expiresAt) {
		return RefreshClaims{ExpiresAt: expiresAt}, ErrRefreshExpired
	}
	return RefreshClaims{UserID: id, TokenID: c.ID, FamilyID: c.FamilyID, ExpiresAt: expiresAt}, nil
}

// Parse memvalidasi signature, exp, dan seluruh klaim wajib.
func (m *RefreshJWTManager) Parse(token string) (RefreshClaims, error) {
	return m.parse(token, true)
}

// ParseAllowExpired dipakai logout: signature dan klaim harus valid, exp boleh lewat.
func (m *RefreshJWTManager) ParseAllowExpired(token string) (RefreshClaims, error) {
	return m.parse(token, false)
}

// TTLSeconds mengembalikan masa berlaku refresh token dalam detik.
func (m *RefreshJWTManager) TTLSeconds() int {
	return int(m.ttl.Seconds())
}

// ValidateJWTSecrets menolak secret yang kosong, pendek, atau sama.
func ValidateJWTSecrets(access, refresh string) error {
	const minLen = 32
	switch {
	case access == "":
		return errors.New("JWT_SECRET tidak boleh kosong")
	case refresh == "":
		return errors.New("JWT_REFRESH_SECRET tidak boleh kosong")
	case len(access) < minLen || len(refresh) < minLen:
		return errors.New("JWT_SECRET dan JWT_REFRESH_SECRET minimal 32 karakter")
	case access == refresh:
		return errors.New("JWT_REFRESH_SECRET harus berbeda dari JWT_SECRET")
	}
	return nil
}