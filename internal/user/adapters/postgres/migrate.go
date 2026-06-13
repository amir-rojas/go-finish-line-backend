package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the tables this module owns. Shared extensions are handled
// centrally by common/postgres.RunMigrations before this runs.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&userModel{}); err != nil {
		return fmt.Errorf("migrating users table: %w", err)
	}
	return nil
}
