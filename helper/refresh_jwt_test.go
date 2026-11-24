package helper

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testRefreshSecret = "secret-refresh-untuk-test-minimal-32-karakter!"

func newTestRefreshManager(ttl time.Duration) *RefreshJWTManager {
	return NewRefreshJWTManager(testRefreshSecret, ttl)
}

// D-U1: round trip Generate lalu Parse mengembalikan klaim yang sama.
func TestRefreshJWTRoundTrip(t *testing.T) {
	m := newTestRefreshManager(time.Hour)

	token, err := m.Generate(7, "familyhex32karaktermap1234567890ab", "tokhex32karaktermap1234567890abcd")
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	claims, err := m.Parse(token)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if claims.UserID != 7 || claims.FamilyID != "familyhex32karaktermap1234567890ab" || claims.TokenID != "tokhex32karaktermap1234567890abcd" {
		t.Errorf("Parse = %+v, want UserID=7 dengan fid dan jti yang sama", claims)
	}
	if claims.ExpiresAt.Before(time.Now()) {
		t.Errorf("ExpiresAt harus di masa depan, dapat %v", claims.ExpiresAt)
	}
}

// D-U1: token rusak, secret salah, dan kedaluwarsa dibedakan errornya.
func TestRefreshJWTParseInvalid(t *testing.T) {
	m := newTestRefreshManager(time.Hour)
	other := NewRefreshJWTManager("secret-lain-yang-juga-minimal-32-karakter!!", time.Hour)
	expired := newTestRefreshManager(-time.Minute)

	foreign, _ := other.Generate(1, "f", "t")
	old, _ := expired.Generate(1, "f", "t")

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{"token acak", "bukan-token", ErrRefreshInvalid},
		{"secret berbeda", foreign, ErrRefreshInvalid},
		{"kedaluwarsa", old, ErrRefreshExpired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.Parse(tt.token)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Parse error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// D-U1: klaim wajib tidak boleh hilang dan typ harus "refresh".
func TestRefreshJWTMissingOrWrongClaims(t *testing.T) {
	m := newTestRefreshManager(time.Hour)

	makeToken := func(claims jwt.MapClaims) string {
		tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testRefreshSecret))
		return tok
	}

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{"typ bukan refresh", jwt.MapClaims{"sub": "1", "jti": "j", "fid": "f", "typ": "access", "exp": time.Now().Add(time.Hour).Unix()}},
		{"typ hilang", jwt.MapClaims{"sub": "1", "jti": "j", "fid": "f", "exp": time.Now().Add(time.Hour).Unix()}},
		{"jti hilang", jwt.MapClaims{"sub": "1", "fid": "f", "typ": "refresh", "exp": time.Now().Add(time.Hour).Unix()}},
		{"fid hilang", jwt.MapClaims{"sub": "1", "jti": "j", "typ": "refresh", "exp": time.Now().Add(time.Hour).Unix()}},
		{"sub hilang", jwt.MapClaims{"jti": "j", "fid": "f", "typ": "refresh", "exp": time.Now().Add(time.Hour).Unix()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := m.Parse(makeToken(tt.claims)); !errors.Is(err, ErrRefreshInvalid) {
				t.Errorf("Parse harus ditolak dengan ErrRefreshInvalid, dapat %v", err)
			}
		})
	}
}

// D-U1: access token (typ access, tanpa klaim refresh) ditolak oleh Parse refresh.
func TestRefreshJWTRejectsAccessToken(t *testing.T) {
	access := NewJWTManager(testRefreshSecret, time.Hour) // sengaja secret sama: yang menolak harus klaim, bukan sekadar secret
	token, _ := access.Generate(7, "a@b.c", "peminjam")

	m := newTestRefreshManager(time.Hour)
	if _, err := m.Parse(token); !errors.Is(err, ErrRefreshInvalid) {
		t.Errorf("access token harus ditolak Parse refresh, dapat %v", err)
	}
}

// D-U1: refresh token juga harus ditolak oleh JWTManager access.
func TestAccessJWTRejectsRefreshToken(t *testing.T) {
	rm := newTestRefreshManager(time.Hour)
	token, _ := rm.Generate(7, "f", "t")

	am := NewJWTManager("secret-access-berbeda-minimal-32-karakter!", time.Hour)
	if _, err := am.Parse(token); err == nil {
		t.Errorf("refresh token harus ditolak Parse access")
	}
}

// D-U1: ParseAllowExpired menerima token kedaluwarsa yang signature-nya valid.
func TestRefreshJWTParseAllowExpired(t *testing.T) {
	m := newTestRefreshManager(-time.Minute)
	token, _ := m.Generate(7, "f", "t")

	claims, err := m.ParseAllowExpired(token)
	if err != nil {
		t.Fatalf("ParseAllowExpired error: %v", err)
	}
	if claims.UserID != 7 || claims.TokenID != "t" || claims.FamilyID != "f" {
		t.Errorf("ParseAllowExpired = %+v, want UserID=7", claims)
	}
}

// D-U2: validasi secret aplikasi saat startup.
func TestValidateJWTSecrets(t *testing.T) {
	tests := []struct {
		name    string
		access  string
		refresh string
		wantErr bool
	}{
		{"valid", "access-secret-minimal-32-karakter!!!", "refresh-secret-minimal-32-karakter!", false},
		{"access kosong", "", "refresh-secret-minimal-32-karakter!", true},
		{"refresh kosong", "access-secret-minimal-32-karakter!!!", "", true},
		{"terlalu pendek", "pendek", "refresh-secret-minimal-32-karakter!", true},
		{"sama persis", "sama-sama-minimal-32-karakter!!!!!!!", "sama-sama-minimal-32-karakter!!!!!!!", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJWTSecrets(tt.access, tt.refresh)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWTSecrets(%q, %q) error = %v, wantErr %v", tt.access, tt.refresh, err, tt.wantErr)
			}
		})
	}
}
