package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Note struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			body TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	return err
}

func ListNotes(ctx context.Context, pool *pgxpool.Pool) ([]Note, error) {
	rows, err := pool.Query(ctx, `SELECT id, body, created_at FROM notes ORDER BY id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func GetNote(ctx context.Context, pool *pgxpool.Pool, id int64) (*Note, error) {
	var n Note
	err := pool.QueryRow(ctx, `SELECT id, body, created_at FROM notes WHERE id = $1`, id).Scan(&n.ID, &n.Body, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func CreateNote(ctx context.Context, pool *pgxpool.Pool, body string) (*Note, error) {
	var n Note
	err := pool.QueryRow(ctx,
		`INSERT INTO notes (body) VALUES ($1) RETURNING id, body, created_at`, body,
	).Scan(&n.ID, &n.Body, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &n, nil
}
