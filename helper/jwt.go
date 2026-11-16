package helper

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ruangku-backend/app/model"
)

type claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{secret: []byte(secret), ttl: ttl}
}

// Generate membuat token HS256 dengan sub (id user), email, role, dan exp.
func (m *JWTManager) Generate(id int, email, role string) (string, error) {
	c := claims{
		Email: email,
		Role:  role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(id),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Parse memvalidasi signature dan masa berlaku, lalu mengembalikan identitas user.
func (m *JWTManager) Parse(t string) (model.AuthUser, error) {
	var c claims
	_, err := jwt.ParseWithClaims(t, &c, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return model.AuthUser{}, err
	}

	id, err := strconv.Atoi(c.Subject)
	if err != nil || id < 1 {
		return model.AuthUser{}, errors.New("subject token tidak valid")
	}
	return model.AuthUser{ID: id, Email: c.Email, Role: c.Role}, nil
}

func (m *JWTManager) TTLSeconds() int {
	return int(m.ttl.Seconds())
}
