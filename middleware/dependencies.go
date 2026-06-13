package middleware

import (
	"go.uber.org/zap"
)

type Dependencies struct {
	logger *zap.Logger
}

func NewDependencies(logger *zap.Logger) *Dependencies {
	return &Dependencies{logger: logger}
}
