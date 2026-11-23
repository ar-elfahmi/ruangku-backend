package repository

import (
	"context"

	"ruangku-backend/app/model"
)

type refreshRepository struct {
	db DBTX
}

func NewRefreshRepository(db DBTX) RefreshTokenRepository {
	return &refreshRepository{db: db}
}

// WithTx mengembalikan salinan repository yang berjalan di dalam transaksi tx.
func (r *refreshRepository) WithTx(tx DBTX) RefreshTokenRepository {
	return &refreshRepository{db: tx}
}

func (r *refreshRepository) InsertFamily(ctx context.Context, f model.RefreshFamily) error {
	return ErrNotImplemented
}

func (r *refreshRepository) InsertToken(ctx context.Context, t model.RefreshToken) error {
	return ErrNotImplemented
}

func (r *refreshRepository) FindFamily(ctx context.Context, id string) (model.RefreshFamily, error) {
	return model.RefreshFamily{}, ErrNotImplemented
}

func (r *refreshRepository) FindFamilyForShare(ctx context.Context, id string) (model.RefreshFamily, error) {
	return model.RefreshFamily{}, ErrNotImplemented
}

func (r *refreshRepository) FindToken(ctx context.Context, jti string) (model.RefreshToken, error) {
	return model.RefreshToken{}, ErrNotImplemented
}

func (r *refreshRepository) UpdateTokenUsed(ctx context.Context, jti, familyID string, userID int) (model.RefreshToken, error) {
	return model.RefreshToken{}, ErrNotImplemented
}

func (r *refreshRepository) UpdateTokenReplacedBy(ctx context.Context, oldJTI, newJTI string) error {
	return ErrNotImplemented
}

func (r *refreshRepository) UpdateFamilyRevoked(ctx context.Context, familyID string, userID int, reason string) (bool, error) {
	return false, ErrNotImplemented
}

var _ RefreshTokenRepository = (*refreshRepository)(nil)