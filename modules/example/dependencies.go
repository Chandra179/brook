package example

import (
	"brook/middleware"
	"brook/zlogger"
)

type Dependencies struct {
	Config     Config
	Logger     *zlogger.Logger
	Middleware *middleware.Dependencies
}

func NewDependencies(cfg Config) *Dependencies {
	log := zlogger.New(cfg.Middleware.Logger.Level)
	return &Dependencies{
		Config:     cfg,
		Logger:     log,
		Middleware: middleware.NewDependencies(log),
	}
}
