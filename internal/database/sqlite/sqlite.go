package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	Conn *sql.DB
}

func NewMemory() (*SQLite, error) {
	conn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
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
