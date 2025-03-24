package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

const (
	dbName = "app"

	defaultMaxOpenConns    = 25
	defaultMaxIdleConns    = 25
	defaultConnMaxLifetime = 5 * time.Minute
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewDB(url, host string) (*bun.DB, error) {
	if url == "" {
		url = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbName, dbName, host, dbName)
	}
	slog.Info("postgres connection string", "url", url)

	sqlDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(url)))
	sqlDB.SetMaxOpenConns(defaultMaxOpenConns)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConns)
	sqlDB.SetConnMaxLifetime(defaultConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	if err := runMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	bunDB := bun.NewDB(sqlDB, pgdialect.New())
	bunDB.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.FromEnv("BUNDEBUG"),
	))

	return bunDB, nil
}

func runMigrations(db *sql.DB) error {
	source := &migrate.EmbedFileSystemMigrationSource{
		FileSystem: migrationsFS,
		Root:       "migrations",
	}
	if _, err := migrate.Exec(db, "postgres", source, migrate.Up); err != nil {
		return err
	}
	return nil
}
