package example

import "context"

type Store interface {
	CreateExample(ctx context.Context, name string) (*Example, error)
}
