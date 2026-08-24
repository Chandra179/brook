package logger

import "go.uber.org/zap"

func NewLogger(appEnvironment string) (*zap.Logger, error) {
	if appEnvironment == "dev" {
		logger, err := zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
		return logger, nil
	}
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
