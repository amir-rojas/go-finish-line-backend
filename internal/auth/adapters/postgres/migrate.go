package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the tables this module owns. The user_id column is indexed
// but not a hard FK, to avoid coupling this gorm model to the user module's
// model; the relationship is enforced by the application.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&refreshTokenModel{}); err != nil {
		return fmt.Errorf("migrating refresh_tokens table: %w", err)
	}
	return nil
}
