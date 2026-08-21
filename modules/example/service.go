package example

import (
	"context"
	"fmt"
)

type Service interface {
	CreateExample(ctx context.Context, name string) (*Example, error)
}

var _ Service = (*dependencies)(nil)

func (d *dependencies) CreateExample(ctx context.Context, name string) (*Example, error) {
	if name == reservedName {
		return nil, fmt.Errorf("create example %q: %w", name, ErrReservedName)
	}

	ex, err := d.store.CreateExample(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create example %q: %w", name, err)
	}

	return ex, nil
}
