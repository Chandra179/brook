package foo

import (
	"go.uber.org/zap"

	"brook/modules/example"
)

type dependencies struct {
	logger  *zap.Logger
	example example.Service
}

type DependenciesConfig struct {
	Logger  *zap.Logger
	Example example.Service
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{logger: deps.Logger, example: deps.Example}
}
