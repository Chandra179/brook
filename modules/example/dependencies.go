package example

import (
	"go.uber.org/zap"
)

type dependencies struct {
	logger *zap.Logger
}

type DependenciesConfig struct {
	Logger *zap.Logger
}

func NewDependencies(deps *DependenciesConfig) *dependencies {
	return &dependencies{logger: deps.Logger}
}
