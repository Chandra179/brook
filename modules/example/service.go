package example

import (
	"context"
	"fmt"
)

func (d *dependencies) CreateExample(ctx context.Context, name string) (*Example, error) {
	ex, err := d.store.CreateExample(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("create example %q: %w", name, err)
	}

	return ex, nil
}
