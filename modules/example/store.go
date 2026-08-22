package example

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type store interface {
	CreateExample(ctx context.Context, name string) (*Example, error)
}

var _ store = (*postgresStore)(nil)

type postgresStore struct {
	pool *pgxpool.Pool
}

func newPostgresStore(pool *pgxpool.Pool) *postgresStore {
	return &postgresStore{pool: pool}
}

func (s *postgresStore) CreateExample(ctx context.Context, name string) (*Example, error) {
	rows, err := s.pool.Query(ctx,
		`INSERT INTO examples (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("insert example: %w", err)
	}

	ex, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Example])
	if err != nil {
		return nil, fmt.Errorf("scan example: %w", err)
	}

	return ex, nil
}
