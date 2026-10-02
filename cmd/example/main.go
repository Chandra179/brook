package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"brook/server"
)

// @title           Brook API
// @version         1.0.0
// @description     Modular monolith API server
func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	return server.RunHttpServer(ctx)
}
