package database

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations automatically applies pending UP SQL migration scripts to the database.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database connection pool is nil")
	}

	// 1. Create schema_migrations table if not exists
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := pool.Exec(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Fetch applied migration versions
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var ver string
		if err := rows.Scan(&ver); err == nil {
			applied[ver] = true
		}
	}

	// 3. Read embedded migration files
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations dir: %w", err)
	}

	var upFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
			upFiles = append(upFiles, entry.Name())
		}
	}
	sort.Strings(upFiles)

	// 4. Apply unapplied migration files sequentially
	appliedCount := 0
	for _, fileName := range upFiles {
		version := strings.TrimSuffix(fileName, ".up.sql")
		if applied[version] {
			continue
		}

		log.Printf("🚀 Applying database migration: %s", fileName)
		content, err := migrationFS.ReadFile("migrations/" + fileName)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fileName, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", fileName, err)
		}

		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("failed to record migration version %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration transaction %s: %w", version, err)
		}

		appliedCount++
		log.Printf("✅ Migration %s applied successfully.", version)
	}

	if appliedCount == 0 {
		log.Println("✨ Database schema is up to date (no pending migrations).")
	} else {
		log.Printf("🎉 Successfully applied %d database migration(s).", appliedCount)
	}

	return nil
}
