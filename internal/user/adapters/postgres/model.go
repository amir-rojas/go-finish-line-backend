package postgres

import (
	"time"

	"github.com/google/uuid"

	"finish-line/internal/user/domain"
)

// userModel is the persistence shape of a User. Column names follow the
// database schema (Spanish), while the domain speaks English — this mapping
// is exactly what the adapter is for.
type userModel struct {
	ID           uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Name         string    `gorm:"column:nombre;type:text;not null"`
	Email        string    `gorm:"column:email;type:citext;not null;uniqueIndex"`
	PasswordHash string    `gorm:"column:password_hash;type:text;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;type:timestamptz;not null"`
}

func (userModel) TableName() string { return "users" }

func toModel(u *domain.User) userModel {
	return userModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
	}
}

func toDomain(m userModel) *domain.User {
	return &domain.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
	}
}
