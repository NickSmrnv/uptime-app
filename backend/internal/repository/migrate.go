package repository

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/uptime-app/backend/migrations"
	"gorm.io/gorm"
)

const migrationTable = "schema_migrations"

type migration struct {
	version int64
	name    string
	path    string
}

// Migrate applies each embedded SQL migration once before the API starts.
// A transaction and PostgreSQL advisory lock keep concurrent API instances from
// applying the same schema change at the same time.
func Migrate(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext('uptime_app_schema_migrations'))`).Error; err != nil {
			return fmt.Errorf("lock migrations: %w", err)
		}
		if err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version bigint PRIMARY KEY,
				name text NOT NULL,
				applied_at timestamptz NOT NULL DEFAULT now()
			)
		`).Error; err != nil {
			return fmt.Errorf("create %s table: %w", migrationTable, err)
		}

		migrationFiles, err := loadMigrations()
		if err != nil {
			return err
		}
		for _, file := range migrationFiles {
			var applied bool
			if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = ?)`, file.version).Scan(&applied).Error; err != nil {
				return fmt.Errorf("check migration %d: %w", file.version, err)
			}
			if applied {
				continue
			}

			sql, err := migrations.FS.ReadFile(file.path)
			if err != nil {
				return fmt.Errorf("read migration %s: %w", file.path, err)
			}
			if err := tx.Exec(string(sql)).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", file.path, err)
			}
			if err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, file.version, file.name).Error; err != nil {
				return fmt.Errorf("record migration %d: %w", file.version, err)
			}
		}
		return nil
	})
}

func loadMigrations() ([]migration, error) {
	paths, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	result := make([]migration, 0, len(paths))
	for _, path := range paths {
		base := filepath.Base(path)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 || !strings.HasSuffix(parts[1], ".sql") {
			return nil, fmt.Errorf("invalid migration filename %q: expected <version>_<name>.sql", base)
		}
		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", base)
		}
		result = append(result, migration{version: version, name: strings.TrimSuffix(parts[1], ".sql"), path: path})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].version < result[j].version })
	for i := 1; i < len(result); i++ {
		if result[i-1].version == result[i].version {
			return nil, fmt.Errorf("duplicate migration version %d", result[i].version)
		}
	}
	return result, nil
}
