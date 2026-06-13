package example

import (
	"go.uber.org/zap"

	"brook/middleware"
)

type Dependencies struct {
	Config     Config
	Logger     *zap.Logger
	Middleware *middleware.Dependencies
}

func NewDependencies(cfg Config) *Dependencies {
	var log *zap.Logger
	if cfg.Middleware.Logger.Level == "dev" {
		log, _ = zap.NewDevelopment()
	} else {
		log, _ = zap.NewProduction()
	}
	return &Dependencies{
		Config:     cfg,
		Logger:     log,
		Middleware: middleware.NewDependencies(log),
	}
}
