package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"finish-line/internal/user/domain"
	"finish-line/internal/user/ports"
)

type Repository struct {
	db *gorm.DB
}

var _ ports.UserRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, u *domain.User) error {
	m := toModel(u)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (r *Repository) ByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("selecting user by id: %w", err)
	}
	return toDomain(m), nil
}

func (r *Repository) ByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m userModel
	if err := r.db.WithContext(ctx).First(&m, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("selecting user by email: %w", err)
	}
	return toDomain(m), nil
}

func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	var models []userModel
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("selecting users: %w", err)
	}

	users := make([]domain.User, 0, len(models))
	for _, m := range models {
		users = append(users, *toDomain(m))
	}
	return users, nil
}
