package example

import (
	"go.uber.org/zap"
)

type Dependencies struct {
	Logger *zap.Logger
}

func NewDependencies(logger *zap.Logger) *Dependencies {
	return &Dependencies{Logger: logger}
}
