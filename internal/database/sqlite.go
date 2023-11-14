package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	Conn *sql.DB
}

//go:embed migration_001.sql
var migration1 string

//go:embed migration_002.sql
var migration2 string

//go:embed migration_003.sql
var migration3 string

func NewMemory() (*SQLite, error) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	// 008-JP1-
	if _, err := conn.Exec(migration1); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(migration2); err != nil {
		return nil, err
	}
	if _, err := conn.Exec(migration3); err != nil {
		return nil, err
	}

	return &SQLite{Conn: conn}, nil
}

func NewLocal(pathToDatabase string) (*SQLite, error) {
	uri := fmt.Sprintf("sqlite3://%s?journaled_mode=WAL", pathToDatabase)

	conn, err := sql.Open("sqlite3", uri)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &SQLite{Conn: conn}, nil
}

func (db *SQLite) Queries() (*Queries, error) {
	return Prepare(context.Background(), db.Conn)
}
