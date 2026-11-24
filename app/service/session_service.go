package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"

	"ruangku-backend/app/model"
	"ruangku-backend/app/repository"
	"ruangku-backend/helper"
)

const (
	messageBodyInvalid   = "body request tidak valid"
	messageRefreshFailed = "refresh token tidak valid atau sudah kedaluwarsa"
)

// revokeTimeout membatasi pencabutan keluarga di luar transaksi. Context-nya
// baru agar pencabutan tidak ikut batal bila klien memutus koneksi.
const revokeTimeout = 3 * time.Second

// sentinel error klasifikasi refresh; dikirim dari dalam tx.Do agar
// transaksi rollback, lalu diterjemahkan di luar.
var (
	errSessionUnknown = errors.New("sesi tidak dikenal")
	errSessionRevoked = errors.New("keluarga token sudah dicabut")
	errSessionForeign = errors.New("keluarga token milik user lain")
	errSessionReuse   = errors.New("pemakaian ulang refresh token")
	errSessionExpired = errors.New("refresh token kedaluwarsa")
)

type sessionService struct {
	repo    repository.RefreshTokenRepository
	refresh *helper.RefreshJWTManager
	access  *helper.JWTManager
	users   repository.UserRepository
	tx      *repository.TxManager
}

func NewSessionService(r repository.RefreshTokenRepository, refresh *helper.RefreshJWTManager, access *helper.JWTManager, u repository.UserRepository, tx *repository.TxManager) SessionService {
	return &sessionService{repo: r, refresh: refresh, access: access, users: u, tx: tx}
}

// Refresh menangani POST /auth/refresh: rotasi single-use dengan deteksi reuse.
func (s *sessionService) Refresh(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest(messageBodyInvalid)
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// 1. Parse refresh JWT.
	claims, err := s.refresh.Parse(req.RefreshToken)
	if err != nil {
		return helper.Unauthorized(messageRefreshFailed)
	}

	// 2. Muat user dari DB: email dan role access token baru selalu terbaru.
	user, err := s.users.FindByID(ctx, claims.UserID)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Unauthorized(messageRefreshFailed)
	case err != nil:
		return helper.Internal(err)
	}

	// 3. Transaksi: kunci keluarga, pakai token lama, terbitkan metadata baru.
	reuse := false
	var newTokenID string
	err = s.tx.Do(ctx, func(tx repository.DBTX) error {
		r := s.repo.WithTx(tx)

		family, err := r.FindFamilyForShare(ctx, claims.FamilyID)
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return errSessionUnknown
		case err != nil:
			return err
		}
		if family.RevokedAt != nil {
			return errSessionRevoked
		}
		if family.UserID != claims.UserID {
			return errSessionForeign
		}

		_, err = r.UpdateTokenUsed(ctx, claims.TokenID, claims.FamilyID, claims.UserID)
		if errors.Is(err, repository.ErrConflict) {
			// Jangan langsung menyimpulkan reuse; klasifikasi dulu.
			token, findErr := r.FindToken(ctx, claims.TokenID)
			switch {
			case errors.Is(findErr, repository.ErrNotFound):
				return errSessionUnknown
			case findErr != nil:
				return findErr
			case token.UsedAt != nil:
				return errSessionReuse
			default:
				return errSessionExpired
			}
		} else if err != nil {
			return err
		}

		id, err := helper.NewTokenID()
		if err != nil {
			return err
		}
		newTokenID = id
		if err := r.InsertToken(ctx, model.RefreshToken{
			JTI:       id,
			FamilyID:  claims.FamilyID,
			UserID:    claims.UserID,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(s.refreshTTL()),
		}); err != nil {
			return err
		}
		return r.UpdateTokenReplacedBy(ctx, claims.TokenID, id)
	})
	switch {
	case errors.Is(err, errSessionReuse):
		reuse = true
	case errors.Is(err, errSessionUnknown), errors.Is(err, errSessionRevoked),
		errors.Is(err, errSessionForeign), errors.Is(err, errSessionExpired):
		return helper.Unauthorized(messageRefreshFailed)
	case err != nil:
		return helper.Internal(err)
	}

	// 4. Reuse: cabut keluarga DI LUAR transaksi dengan context baru.
	if reuse {
		if err := s.revokeFamily(claims.FamilyID, claims.UserID, "reuse"); err != nil {
			return helper.Internal(err)
		}
		return helper.Unauthorized(messageRefreshFailed)
	}

	// 5. Terbitkan pasangan token baru.
	access, err := s.access.Generate(user.ID, user.Email, user.Role)
	if err != nil {
		return helper.Internal(err)
	}
	refresh, err := s.refresh.Generate(user.ID, claims.FamilyID, newTokenID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "token berhasil diperbarui", model.TokenResponse{
		AccessToken:      access,
		ExpiresIn:        s.access.TTLSeconds(),
		RefreshToken:     refresh,
		RefreshExpiresIn: s.refresh.TTLSeconds(),
	})
}

// revokeFamily mencabut keluarga dengan context baru (bukan context request)
// agar pencabutan tidak ikut rollback dan tidak hilang bila klien memutus.
func (s *sessionService) revokeFamily(familyID string, userID int, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), revokeTimeout)
	defer cancel()
	_, err := s.repo.UpdateFamilyRevoked(ctx, familyID, userID, reason)
	if err != nil {
		log.Printf("sesi: gagal mencabut keluarga %s (user %d, alasan %s): %v", familyID, userID, reason, err)
	}
	return err
}

func (s *sessionService) refreshTTL() time.Duration {
	return time.Duration(s.refresh.TTLSeconds()) * time.Second
}

func (s *sessionService) Logout(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest(messageBodyInvalid)
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	// Signature dan klaim harus valid; exp boleh sudah lewat.
	claims, err := s.refresh.ParseAllowExpired(req.RefreshToken)
	if err != nil {
		return helper.Unauthorized(messageRefreshFailed)
	}

	user, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized(messageInvalidToken)
	}

	changed, err := s.repo.UpdateFamilyRevoked(ctx, claims.FamilyID, user.ID, "logout")
	if err != nil {
		return helper.Internal(err)
	}
	if !changed {
		// Bedakan: keluarga tidak ada (204) vs milik user lain (403) vs sudah revoked (204).
		family, findErr := s.repo.FindFamily(ctx, claims.FamilyID)
		switch {
		case errors.Is(findErr, repository.ErrNotFound):
			return helper.NoContent(c)
		case findErr != nil:
			return helper.Internal(findErr)
		case family.UserID != user.ID:
			return helper.Forbidden("akses ditolak")
		default:
			return helper.NoContent(c) // sudah pernah dicabut: tetap 204
		}
	}
	return helper.NoContent(c)
}

type sessionIssuer struct {
	repo    repository.RefreshTokenRepository
	tx      *repository.TxManager
	refresh *helper.RefreshJWTManager
}

func NewRefreshIssuer(r repository.RefreshTokenRepository, tx *repository.TxManager, refresh *helper.RefreshJWTManager) RefreshIssuer {
	return &sessionIssuer{repo: r, tx: tx, refresh: refresh}
}

// Issue membuat keluarga baru plus token pertamanya. Dipanggil oleh Login
// SETELAH user ditemukan dan password cocok.
func (i *sessionIssuer) Issue(ctx context.Context, userID int) (string, int, error) {
	familyID, err := helper.NewTokenID()
	if err != nil {
		return "", 0, err
	}
	tokenID, err := helper.NewTokenID()
	if err != nil {
		return "", 0, err
	}

	err = i.tx.Do(ctx, func(tx repository.DBTX) error {
		r := i.repo.WithTx(tx)
		if err := r.InsertFamily(ctx, model.RefreshFamily{
			ID:        familyID,
			UserID:    userID,
			CreatedAt: time.Now(),
		}); err != nil {
			return err
		}
		return r.InsertToken(ctx, model.RefreshToken{
			JTI:       tokenID,
			FamilyID:  familyID,
			UserID:    userID,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Duration(i.refresh.TTLSeconds()) * time.Second),
		})
	})
	if err != nil {
		return "", 0, err
	}

	signed, err := i.refresh.Generate(userID, familyID, tokenID)
	if err != nil {
		return "", 0, err
	}
	return signed, i.refresh.TTLSeconds(), nil
}
