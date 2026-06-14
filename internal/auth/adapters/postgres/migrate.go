package postgres

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the tables this module owns and links refresh tokens to
// their user. The FK is added with raw SQL (not a gorm model relation) to
// avoid coupling this model to the user module's gorm model; ON DELETE
// CASCADE means deleting a user cleans up their tokens.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&refreshTokenModel{}); err != nil {
		return fmt.Errorf("migrating refresh_tokens table: %w", err)
	}

	const addFK = `
DO $$ BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname = 'fk_refresh_tokens_user'
  ) THEN
    ALTER TABLE refresh_tokens
      ADD CONSTRAINT fk_refresh_tokens_user
      FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
  END IF;
END $$;`
	if err := db.Exec(addFK).Error; err != nil {
		return fmt.Errorf("adding refresh_tokens user foreign key: %w", err)
	}
	return nil
}
