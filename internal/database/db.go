package database

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"investo/internal/config"
)

func Connect(cfg *config.Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("database connect: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping: %w", err)
	}

	return db, nil
}

func RunMigrations(db *sqlx.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS migrations (
		filename VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, f := range files {
		var applied bool
		if err := db.Get(&applied, "SELECT COUNT(*) > 0 FROM migrations WHERE filename = ?", f); err != nil {
			return fmt.Errorf("check migration %s: %w", f, err)
		}
		if applied {
			continue
		}

		data, err := fs.ReadFile(migrationsFS, "migrations/"+f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		content := string(data)
		statements := strings.Split(content, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s: %w", f, err)
			}
		}

		if _, err := db.Exec("INSERT INTO migrations (filename) VALUES (?)", f); err != nil {
			return fmt.Errorf("record migration %s: %w", f, err)
		}
	}

	return nil
}
