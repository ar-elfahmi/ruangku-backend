package repository

import (
	"context"

	"ruangku-backend/app/model"
)

const userColumns = "id, name, email, password_hash, role, created_at"

type userRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	const query = `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + userColumns

	var created model.User
	err := r.db.QueryRow(ctx, query, u.Name, u.Email, u.PasswordHash, u.Role).Scan(
		&created.ID,
		&created.Name,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.CreatedAt,
	)
	return created, TranslateError(err)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE lower(email) = lower($1)`

	var user model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	return user, TranslateError(err)
}

func (r *userRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	const query = `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	return user, TranslateError(err)
}

func (r *userRepository) UpdateRole(ctx context.Context, id int, role string) (model.User, error) {
	const query = `UPDATE users SET role = $2 WHERE id = $1 RETURNING ` + userColumns

	var user model.User
	err := r.db.QueryRow(ctx, query, id, role).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	return user, TranslateError(err)
}

var _ UserRepository = (*userRepository)(nil)
