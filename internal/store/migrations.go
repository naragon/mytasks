package store

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type migration struct {
	version int
	name    string
	sql     string
}

func runMigrations(db *sql.DB) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	if err := bootstrapLegacyMigrations(db, migrations); err != nil {
		return err
	}

	applied, err := appliedMigrationVersions(db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}

		if err := applyMigration(db, m); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	seen := make(map[int]struct{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if filepath.Ext(filename) != ".sql" {
			continue
		}

		version, name, err := parseMigrationFilename(filename)
		if err != nil {
			return nil, err
		}

		if _, exists := seen[version]; exists {
			return nil, fmt.Errorf("duplicate migration version: %d", version)
		}
		seen[version] = struct{}{}

		content, err := migrationsFS.ReadFile(filepath.Join("migrations", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		migrations = append(migrations, migration{
			version: version,
			name:    name,
			sql:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func parseMigrationFilename(filename string) (int, string, error) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid migration filename %q: expected '<version>_<name>.sql'", filename)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid migration version in %q: %w", filename, err)
	}

	return version, parts[1], nil
}

func appliedMigrationVersions(db *sql.DB) (map[int]bool, error) {
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	versions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		versions[version] = true
	}

	return versions, rows.Err()
}

func applyMigration(db *sql.DB, m migration) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction for %d_%s: %w", m.version, m.name, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(m.sql); err != nil {
		return fmt.Errorf("failed to apply migration %d_%s: %w", m.version, m.name, err)
	}

	if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
		return fmt.Errorf("failed to record migration %d_%s: %w", m.version, m.name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration %d_%s: %w", m.version, m.name, err)
	}

	return nil
}

func bootstrapLegacyMigrations(db *sql.DB, migrations []migration) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		return fmt.Errorf("failed to count existing migrations: %w", err)
	}
	if count > 0 {
		return nil
	}

	hasProjects, err := tableExists(db, "projects")
	if err != nil {
		return err
	}
	hasTasks, err := tableExists(db, "tasks")
	if err != nil {
		return err
	}

	if !hasProjects && !hasTasks {
		return nil
	}

	baselineVersion := 1
	hasCompletedAt, err := columnExists(db, "tasks", "completed_at")
	if err != nil {
		return err
	}
	if hasCompletedAt {
		baselineVersion = 2
	}

	hasProjectCompleted, err := columnExists(db, "projects", "completed")
	if err != nil {
		return err
	}
	hasProjectCompletedAt, err := columnExists(db, "projects", "completed_at")
	if err != nil {
		return err
	}
	if hasProjectCompleted && hasProjectCompletedAt {
		baselineVersion = 3
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin legacy migration bootstrap transaction: %w", err)
	}
	defer tx.Rollback()

	for _, m := range migrations {
		if m.version > baselineVersion {
			break
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, m.version, m.name); err != nil {
			return fmt.Errorf("failed to bootstrap legacy migration %d_%s: %w", m.version, m.name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit legacy migration bootstrap: %w", err)
	}

	return nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check table %s: %w", table, err)
	}

	return true, nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, fmt.Errorf("failed to query table info for %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			typeName   string
			notNull    int
			defaultV   interface{}
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &defaultV, &primaryKey); err != nil {
			return false, fmt.Errorf("failed to scan table info for %s: %w", table, err)
		}

		if name == column {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("failed iterating table info for %s: %w", table, err)
	}

	return false, nil
}
