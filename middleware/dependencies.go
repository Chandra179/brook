package middleware

import (
	"go.uber.org/zap"
)

type dependencies struct {
	logger *zap.Logger
}

func NewDependencies(logger *zap.Logger) *dependencies {
	return &dependencies{
		logger: logger,
	}
}