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
	db     *sql.DB
}

var _ Service = (*dependencies)(nil)

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{logger: deps.Logger, db: deps.DB}
}
