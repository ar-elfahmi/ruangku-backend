package repository

import (
	"context"

	"ruangku-backend/app/model"
)

type userRepository struct {
	db DBTX
}

func NewUserRepository(db DBTX) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	return model.User{}, ErrNotImplemented
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	return model.User{}, ErrNotImplemented
}

func (r *userRepository) FindByID(ctx context.Context, id int) (model.User, error) {
	return model.User{}, ErrNotImplemented
}

func (r *userRepository) UpdateRole(ctx context.Context, id int, role string) (model.User, error) {
	return model.User{}, ErrNotImplemented
}

var _ UserRepository = (*userRepository)(nil)
