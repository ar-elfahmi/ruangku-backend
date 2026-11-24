package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"ruangku-backend/app/model"
)

const (
	refreshFamilyColumns = "id, user_id, created_at, revoked_at, revoked_reason"
	refreshTokenColumns  = "jti, family_id, user_id, issued_at, expires_at, used_at, replaced_by"
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

func scanFamily(row pgx.Row) (model.RefreshFamily, error) {
	var f model.RefreshFamily
	err := row.Scan(&f.ID, &f.UserID, &f.CreatedAt, &f.RevokedAt, &f.RevokedReason)
	return f, err
}

func scanToken(row pgx.Row) (model.RefreshToken, error) {
	var t model.RefreshToken
	err := row.Scan(&t.JTI, &t.FamilyID, &t.UserID, &t.IssuedAt, &t.ExpiresAt, &t.UsedAt, &t.ReplacedBy)
	return t, err
}

// InsertFamily membuat keluarga token baru.
func (r *refreshRepository) InsertFamily(ctx context.Context, f model.RefreshFamily) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO refresh_families (id, user_id, created_at) VALUES ($1, $2, $3)",
		f.ID, f.UserID, f.CreatedAt)
	return TranslateError(err)
}

// InsertToken menyimpan metadata satu refresh token yang baru diterbitkan.
func (r *refreshRepository) InsertToken(ctx context.Context, t model.RefreshToken) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO refresh_tokens (jti, family_id, user_id, issued_at, expires_at) VALUES ($1, $2, $3, $4, $5)",
		t.JTI, t.FamilyID, t.UserID, t.IssuedAt, t.ExpiresAt)
	return TranslateError(err)
}

// FindFamily membaca satu keluarga tanpa lock.
func (r *refreshRepository) FindFamily(ctx context.Context, id string) (model.RefreshFamily, error) {
	row := r.db.QueryRow(ctx, "SELECT "+refreshFamilyColumns+" FROM refresh_families WHERE id = $1", id)
	f, err := scanFamily(row)
	if err != nil {
		return model.RefreshFamily{}, TranslateError(err)
	}
	return f, nil
}

// FindFamilyForShare membaca keluarga dengan SELECT ... FOR SHARE: barisnya
// terkunci bersama sampai transaksi selesai, sehingga logout yang butuh lock
// eksklusif menunggu sampai refresh commit (lihat dokumen Phase 2, 3.5).
func (r *refreshRepository) FindFamilyForShare(ctx context.Context, id string) (model.RefreshFamily, error) {
	row := r.db.QueryRow(ctx, "SELECT "+refreshFamilyColumns+" FROM refresh_families WHERE id = $1 FOR SHARE", id)
	f, err := scanFamily(row)
	if err != nil {
		return model.RefreshFamily{}, TranslateError(err)
	}
	return f, nil
}

// FindToken membaca metadata satu token berdasarkan jti.
func (r *refreshRepository) FindToken(ctx context.Context, jti string) (model.RefreshToken, error) {
	row := r.db.QueryRow(ctx, "SELECT "+refreshTokenColumns+" FROM refresh_tokens WHERE jti = $1", jti)
	t, err := scanToken(row)
	if err != nil {
		return model.RefreshToken{}, TranslateError(err)
	}
	return t, nil
}

// UpdateTokenUsed bersifat guarded: hanya menandai token yang belum pernah
// dipakai dan belum kedaluwarsa. 0 baris berarti pemakaian tidak sah
// (sudah dipakai / kedaluwarsa / tidak milik keluarga-user ini) → ErrConflict.
func (r *refreshRepository) UpdateTokenUsed(ctx context.Context, jti, familyID string, userID int) (model.RefreshToken, error) {
	row := r.db.QueryRow(ctx,
		"UPDATE refresh_tokens SET used_at = now() WHERE jti = $1 AND family_id = $2 AND user_id = $3 AND used_at IS NULL AND expires_at > now() RETURNING "+refreshTokenColumns,
		jti, familyID, userID)
	t, err := scanToken(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.RefreshToken{}, ErrConflict
	}
	if err != nil {
		return model.RefreshToken{}, TranslateError(err)
	}
	return t, nil
}

// UpdateTokenReplacedBy mencatat token pengganti pada token lama yang baru saja dipakai.
func (r *refreshRepository) UpdateTokenReplacedBy(ctx context.Context, oldJTI, newJTI string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE refresh_tokens SET replaced_by = $2 WHERE jti = $1",
		oldJTI, newJTI)
	return TranslateError(err)
}

// UpdateFamilyRevoked mencabut keluarga milik userID dengan alasan tertentu.
// Hanya keluarga yang belum dicabut yang berubah; true berarti ada baris berubah.
func (r *refreshRepository) UpdateFamilyRevoked(ctx context.Context, familyID string, userID int, reason string) (bool, error) {
	tag, err := r.db.Exec(ctx,
		"UPDATE refresh_families SET revoked_at = now(), revoked_reason = $3 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL",
		familyID, userID, reason)
	if err != nil {
		return false, TranslateError(err)
	}
	return tag.RowsAffected() > 0, nil
}

var _ RefreshTokenRepository = (*refreshRepository)(nil)
