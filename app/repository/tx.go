package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TxManager menjalankan beberapa operasi repository dalam satu transaksi.
type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(p *pgxpool.Pool) *TxManager {
	return &TxManager{pool: p}
}

// Do menjalankan fn di dalam transaksi. Commit bila fn mengembalikan nil,
// rollback bila sebaliknya.
func (t *TxManager) Do(ctx context.Context, fn func(tx DBTX) error) error {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return err
	}
	// Rollback setelah Commit berhasil adalah no-op, jadi aman dipanggil selalu.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
