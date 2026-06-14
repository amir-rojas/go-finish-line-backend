package postgres

import (
	"time"

	"github.com/google/uuid"

	"finish-line/internal/auth/domain"
)

// refreshTokenModel is the persistence shape of a refresh token. Columns are
// in English: this is an auth infrastructure table, not part of the business
// schema.
type refreshTokenModel struct {
	ID        uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"column:user_id;type:uuid;not null;index"`
	FamilyID  uuid.UUID  `gorm:"column:family_id;type:uuid;not null;index"`
	TokenHash string     `gorm:"column:token_hash;type:text;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"column:expires_at;type:timestamptz;not null"`
	RotatedAt *time.Time `gorm:"column:rotated_at;type:timestamptz"`
	RevokedAt *time.Time `gorm:"column:revoked_at;type:timestamptz"`
	CreatedAt time.Time  `gorm:"column:created_at;type:timestamptz;not null"`
}

func (refreshTokenModel) TableName() string { return "refresh_tokens" }

func toModel(t *domain.RefreshToken) refreshTokenModel {
	return refreshTokenModel{
		ID:        t.ID,
		UserID:    t.UserID,
		FamilyID:  t.FamilyID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		RotatedAt: t.RotatedAt,
		RevokedAt: t.RevokedAt,
		CreatedAt: t.CreatedAt,
	}
}

func toDomain(m refreshTokenModel) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        m.ID,
		UserID:    m.UserID,
		FamilyID:  m.FamilyID,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		RotatedAt: m.RotatedAt,
		RevokedAt: m.RevokedAt,
		CreatedAt: m.CreatedAt,
	}
}
