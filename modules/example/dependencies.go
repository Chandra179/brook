package example

import (
	"database/sql"

	"go.uber.org/zap"
)

type DependenciesConfig struct {
	Logger *zap.Logger
	DB     *sql.DB
}

type dependencies struct {
	logger *zap.Logger
	store  store
}

type storeDependencies struct {
	db *sql.DB
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{logger: deps.Logger, store: &storeDependencies{db: deps.DB}}
}
