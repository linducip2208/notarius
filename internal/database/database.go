package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Open connects to MySQL 8.x. Times are UTC in the database;
// presentation converts to Asia/Jakarta (WIB).
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	return db, nil
}

// Migrate applies embedded *.sql files in lexical order, tracking
// applied versions in schema_migrations. Safe to run on every boot.
func Migrate(db *sql.DB, files fs.FS, dir string) (int, error) {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(191) PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return 0, fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := fs.ReadDir(files, dir)
	if err != nil {
		return 0, err
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	applied := 0
	for _, n := range names {
		var v string
		err := db.QueryRow(`SELECT version FROM schema_migrations WHERE version=?`, n).Scan(&v)
		if err == nil {
			continue // already applied
		}
		if err != sql.ErrNoRows {
			return applied, err
		}
		raw, err := fs.ReadFile(files, n)
		if err != nil && dir != "" && dir != "." {
			raw, err = fs.ReadFile(files, dir+"/"+n)
		}
		if err != nil {
			return applied, err
		}
		for _, stmt := range splitStatements(string(raw)) {
			if strings.TrimSpace(stmt) == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				return applied, fmt.Errorf("migration %s: %w", n, err)
			}
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations(version) VALUES(?)`, n); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}

func splitStatements(s string) []string {
	lines := strings.Split(s, "\n")
	var b strings.Builder
	var out []string
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "--") {
			continue
		}
		b.WriteString(ln + "\n")
		if strings.HasSuffix(t, ";") {
			out = append(out, b.String())
			b.Reset()
		}
	}
	if rest := strings.TrimSpace(b.String()); rest != "" {
		out = append(out, b.String())
	}
	return out
}
