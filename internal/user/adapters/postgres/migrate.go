package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the schema this module needs. Development convenience
// only — production schema changes must be explicit, reviewed migrations.
func Migrate(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS citext").Error; err != nil {
		return fmt.Errorf("enabling citext extension: %w", err)
	}
	if err := db.AutoMigrate(&userModel{}); err != nil {
		return fmt.Errorf("migrating users table: %w", err)
	}
	return nil
}
