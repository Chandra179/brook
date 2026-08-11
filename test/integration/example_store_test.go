//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"brook/config"
	"brook/modules/example"
	"brook/store"
)

func TestExampleStore_CreateExample(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	require.NotEmpty(t, dsn, "POSTGRES_DSN must be set to run integration tests")

	ctx := context.Background()

	pool, err := store.NewPool(ctx, config.PostgresConfig{DSN: dsn, MaxConns: 5, MinConns: 1})
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	s := example.NewPostgresStore(pool)

	got, err := s.CreateExample(ctx, "integration-test")
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)
	require.Equal(t, "integration-test", got.Name)
	require.False(t, got.CreatedAt.IsZero())
}
