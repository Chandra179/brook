package example

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (d *dependencies) CreateExample(ctx context.Context, name string) (*Example, error) {
	if name == reservedName {
		return nil, fmt.Errorf("create example: %w", ErrReservedName)
	}

	ex := &Example{ID: uuid.NewString(), Name: name}

	err := d.db.QueryRowContext(ctx,
		`INSERT INTO examples (id, name) VALUES (?, ?) RETURNING id, name, created_at`,
		ex.ID, ex.Name,
	).Scan(&ex.ID, &ex.Name, &ex.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create example: %w", err)
	}

	return ex, nil
}
