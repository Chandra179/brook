package example

import (
	"go.uber.org/zap"
)

type dependencies struct {
	logger *zap.Logger
	store  Store
}

type DependenciesConfig struct {
	Logger *zap.Logger
	Store  Store
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{logger: deps.Logger, store: deps.Store}
}
