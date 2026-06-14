package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"finish-line/internal/auth/domain"
	"finish-line/internal/auth/ports"
)

type Repository struct {
	db *gorm.DB
}

var _ ports.RefreshTokenRepository = (*Repository)(nil)

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, t *domain.RefreshToken) error {
	m := toModel(t)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("inserting refresh token: %w", err)
	}
	return nil
}

func (r *Repository) ByTokenHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var m refreshTokenModel
	if err := r.db.WithContext(ctx).First(&m, "token_hash = ?", hash).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("selecting refresh token: %w", err)
	}
	return toDomain(m), nil
}

func (r *Repository) MarkRotated(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&refreshTokenModel{}).
		Where("id = ?", id).
		Update("rotated_at", time.Now()).Error
	if err != nil {
		return fmt.Errorf("marking refresh token rotated: %w", err)
	}
	return nil
}

// RevokeFamily revokes every still-active token in a family at once — the
// response to a detected token replay.
func (r *Repository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Model(&refreshTokenModel{}).
		Where("family_id = ? AND revoked_at IS NULL", familyID).
		Update("revoked_at", time.Now()).Error
	if err != nil {
		return fmt.Errorf("revoking token family: %w", err)
	}
	return nil
}
