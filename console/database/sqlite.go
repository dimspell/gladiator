package database

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLite struct {
	Conn *sql.DB
}

func NewMemory() (*SQLite, error) {
	slog.Debug("Connecting to in-memory SQLite database")

	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if err := Migrate(conn); err != nil {
		return nil, err
	}

	return &SQLite{Conn: conn}, nil
}

func NewLocal(pathToDatabase string) (*SQLite, error) {
	pragmas := "_pragma=busy_timeout(10000)&" +
		"_pragma=journal_mode(WAL)&" +
		"_pragma=journal_size_limit(200000000)&" +
		"_pragma=synchronous(NORMAL)&" +
		"_pragma=foreign_keys(ON)&" +
		"_pragma=temp_store(MEMORY)&" +
		"_pragma=cache_size(-16000)"
	uri := fmt.Sprintf("%s?%s", pathToDatabase, pragmas)
	slog.Debug("Connecting to local SQLite database", "uri", uri)

	conn, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if err := Migrate(conn); err != nil {
		return nil, err
	}

	return &SQLite{Conn: conn}, nil
}

func Migrate(conn *sql.DB) error {
	// Prepare resources
	migrationSource, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}
	driver, err := sqlite.WithInstance(conn, &sqlite.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", migrationSource, "sqlite", driver)
	if err != nil {
		return err
	}

	// Migrate
	// _ = m.Down()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("migration: %w", err)
	}
	return nil
}

func (db *SQLite) Queries() (*Queries, error) {
	return Prepare(context.Background(), db.Conn)
}
