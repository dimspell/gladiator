package database

import (
	"context"
	"database/sql"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLite struct {
	Writer *sql.DB
	Reader *sql.DB
	Read   *Queries
	Write  *Queries
}

func NewMemory() (*SQLite, error) {
	slog.Debug("Connecting to in-memory SQLite database")

	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}

	// Set max open connections to 1 to prevent concurrent writes
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if err := Migrate(conn); err != nil {
		return nil, err
	}

	queriesRead, err := Prepare(context.Background(), conn)
	if err != nil {
		return nil, err
	}

	queriesWrite, err := Prepare(context.Background(), conn)
	if err != nil {
		return nil, err
	}

	return &SQLite{
		Reader: conn,
		Read:   queriesRead,
		Writer: conn,
		Write:  queriesWrite,
	}, nil
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

	writer, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}

	writer.SetMaxOpenConns(1)

	if err := writer.Ping(); err != nil {
		return nil, err
	}
	if err := Migrate(writer); err != nil {
		return nil, err
	}

	reader, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}

	reader.SetMaxOpenConns(min(runtime.NumCPU(), 4))
	if err := reader.Ping(); err != nil {
		return nil, err
	}

	queriesRead, err := Prepare(context.Background(), reader)
	if err != nil {
		return nil, err
	}

	queriesWrite, err := Prepare(context.Background(), writer)
	if err != nil {
		return nil, err
	}

	return &SQLite{
		Reader: reader,
		Read:   queriesRead,
		Writer: writer,
		Write:  queriesWrite,
	}, nil
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

	{
		version, dirty, err := m.Version()
		slog.Info("Migration status", "version", version, "dirty", dirty, logging.Error(err))
	}
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("migration: %w", err)
	}
	{
		version, dirty, err := m.Version()
		slog.Info("Migration complete", "version", version, "dirty", dirty, "error", err)
	}

	return nil
}

func (db *SQLite) Ping() error {
	if err := db.Reader.Ping(); err != nil {
		return err
	}
	return nil
}

func (db *SQLite) WithTx(ctx context.Context) (*sql.Tx, *Queries, error) {
	tx, err := db.Writer.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, nil, err
	}
	return tx, db.Write.WithTx(tx), nil
}

func (db *SQLite) Close() error {
	return errors.Join(
		db.Write.Close(),
		db.Read.Close(),
		db.Writer.Close(),
		db.Reader.Close(),
	)
}
