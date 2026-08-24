package example

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type dependencies struct {
	logger *zap.Logger
	store  store
}

type DependenciesConfig struct {
	Logger *zap.Logger
	Pool   *pgxpool.Pool
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{logger: deps.Logger, store: newPostgresStore(deps.Pool)}
}
