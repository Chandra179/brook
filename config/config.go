package config

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP       HTTPConfig       `yaml:"http"`
	Logger     LoggerConfig     `yaml:"logger"`
	Middleware MiddlewareConfig `yaml:"middleware"`
}

type HTTPConfig struct {
	Port              string `yaml:"port"`
	ReadTimeoutInSec  int    `yaml:"read_timeout_in_second"`
	WriteTimeoutInSec int    `yaml:"write_timeout_in_second"`
	IdleTimeoutInSec  int    `yaml:"idle_timeout_in_second"`
}

type MiddlewareConfig struct {
	TimeoutInSec int              `yaml:"timeout_in_second"`
	RequestLog   RequestLogConfig `yaml:"request_log"`
	RealIP       RealIPConfig     `yaml:"real_ip"`
}

type RequestLogConfig struct {
	SkipPaths []string `yaml:"skip_paths"`
	LogQuery  bool     `yaml:"log_query"`
}

type RealIPConfig struct {
	TrustedProxies []string `yaml:"trusted_proxies"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
}

func (l LoggerConfig) New() *zap.Logger {
	if l.Level == "dev" {
		logger, _ := zap.NewDevelopment()
		return logger
	}
	logger, _ := zap.NewProduction()
	return logger
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
