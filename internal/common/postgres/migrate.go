package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateFunc applies one module's schema. Each module owns its own tables;
// this package owns shared concerns (extensions) and the order of execution.
type MigrateFunc func(db *gorm.DB) error

// requiredExtensions are enabled once, before any module migration runs.
// They are cross-cutting: e.g. citext backs case-insensitive emails used by
// more than one module.
var requiredExtensions = []string{"citext"}

// RunMigrations enables shared extensions and then runs each module's
// migration in the given order. It is a development convenience driven by
// gorm AutoMigrate; production schema changes ship as explicit, versioned
// migrations instead.
func RunMigrations(db *gorm.DB, migrations ...MigrateFunc) error {
	for _, ext := range requiredExtensions {
		if err := db.Exec("CREATE EXTENSION IF NOT EXISTS " + ext).Error; err != nil {
			return fmt.Errorf("enabling extension %q: %w", ext, err)
		}
	}

	for _, migrate := range migrations {
		if err := migrate(db); err != nil {
			return fmt.Errorf("running module migration: %w", err)
		}
	}

	return nil
}
