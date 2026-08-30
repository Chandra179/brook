package store

import (
	"context"
	"fmt"

	"github.com/arangodb/go-driver/v2/arangodb"
	"github.com/arangodb/go-driver/v2/connection"

	"brook/config"
)

// NewArangoClient creates an ArangoDB client connection and verifies
// connectivity with the server version before returning, so a bad
// endpoint or auth fails fast at startup instead of surfacing on the
// first request. It also ensures the configured database exists.
func NewArangoClient(ctx context.Context, cfg config.ArangoDBConfig) (arangodb.Client, error) {
	endpoint := connection.NewRoundRobinEndpoints(cfg.Endpoints)
	conn := connection.NewHttp2Connection(connection.DefaultHTTP2ConfigurationWrapper(endpoint, true))

	auth := connection.NewBasicAuth(cfg.Username, cfg.Password)
	if err := conn.SetAuthentication(auth); err != nil {
		return nil, fmt.Errorf("set arangodb auth: %w", err)
	}

	client := arangodb.NewClient(conn)

	if _, err := client.Version(ctx); err != nil {
		return nil, fmt.Errorf("ping arangodb: %w", err)
	}

	exists, err := client.DatabaseExists(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("check arangodb database %q: %w", cfg.Database, err)
	}
	if !exists {
		if _, err := client.CreateDatabase(ctx, cfg.Database, nil); err != nil {
			return nil, fmt.Errorf("create arangodb database %q: %w", cfg.Database, err)
		}
	}

	return client, nil
}