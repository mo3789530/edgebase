package migration

import (
	"fmt"
	"time"

	"github.com/edgebase/platform/control-plane/internal/logger"
	"gorm.io/gorm"
)

type Migration struct {
	Version     int
	Description string
	Up          func(*gorm.DB) error
	Down        func(*gorm.DB) error
}

type MigrationRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Version   int       `gorm:"unique;not null"`
	Name      string    `gorm:"not null"`
	AppliedAt time.Time `gorm:"not null;default:now()"`
}

type Manager struct {
	db         *gorm.DB
	migrations []Migration
}

func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db:         db,
		migrations: []Migration{},
	}
}

func (m *Manager) Register(migration Migration) {
	m.migrations = append(m.migrations, migration)
}

func (m *Manager) Migrate() error {
	// Create migrations table if not exists
	if err := m.db.AutoMigrate(&MigrationRecord{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	for _, migration := range m.migrations {
		var record MigrationRecord
		result := m.db.Where("version = ?", migration.Version).First(&record)

		if result.Error == gorm.ErrRecordNotFound {
			logger.Info("", "applying_migration", map[string]interface{}{
				"version":     migration.Version,
				"description": migration.Description,
			})

			if err := migration.Up(m.db); err != nil {
				logger.Error("", "migration_failed", err)
				return fmt.Errorf("migration %d failed: %w", migration.Version, err)
			}

			if err := m.db.Create(&MigrationRecord{
				Version: migration.Version,
				Name:    migration.Description,
			}).Error; err != nil {
				return fmt.Errorf("failed to record migration: %w", err)
			}

			logger.Info("", "migration_applied", map[string]interface{}{
				"version": migration.Version,
			})
		}
	}

	return nil
}

func (m *Manager) Rollback(version int) error {
	var migration *Migration
	for i := range m.migrations {
		if m.migrations[i].Version == version {
			migration = &m.migrations[i]
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration version %d not found", version)
	}

	logger.Info("", "rolling_back_migration", map[string]interface{}{
		"version": version,
	})

	if err := migration.Down(m.db); err != nil {
		logger.Error("", "rollback_failed", err)
		return fmt.Errorf("rollback failed: %w", err)
	}

	if err := m.db.Where("version = ?", version).Delete(&MigrationRecord{}).Error; err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	logger.Info("", "migration_rolled_back", map[string]interface{}{
		"version": version,
	})

	return nil
}
