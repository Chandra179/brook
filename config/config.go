package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logger     LoggerConfig     `yaml:"logger"`
	SQLite     SQLiteConfig     `yaml:"sqlite"`
	Badger     BadgerConfig     `yaml:"badger"`
	Middleware MiddlewareConfig `yaml:"middleware"`
	HTTP       HTTPConfig       `yaml:"http"`
}

type HTTPConfig struct {
	Port                 string `yaml:"port"`
	ReadTimeoutInSec     int    `yaml:"read_timeout_in_second"`
	WriteTimeoutInSec    int    `yaml:"write_timeout_in_second"`
	IdleTimeoutInSec     int    `yaml:"idle_timeout_in_second"`
	ShutdownTimeoutInSec int    `yaml:"shutdown_timeout_in_second"`
}

type MiddlewareConfig struct {
	RequestLog RequestLogConfig `yaml:"request_log"`
}

type RequestLogConfig struct {
	SkipPaths []string `yaml:"skip_paths"`
	LogQuery  bool     `yaml:"log_query"`
}

type LoggerConfig struct {
	Level string `yaml:"level"`
}

type SQLiteConfig struct {
	DSN string `yaml:"dsn"`
}

type BadgerConfig struct {
	Dir string `yaml:"dir"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	if dsn := os.Getenv("SQLITE_DSN"); dsn != "" {
		cfg.SQLite.DSN = dsn
	}
	if dir := os.Getenv("BADGER_DIR"); dir != "" {
		cfg.Badger.Dir = dir
	}

	return &cfg, nil
}
