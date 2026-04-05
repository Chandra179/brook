package internal

import (
	"brook/internal/middleware"

	"github.com/Chandra179/gosdk/logger"
)

func Server() {
	logger := logger.NewLogger("dev")
	deps := middleware.NewDependencies(logger)

}
