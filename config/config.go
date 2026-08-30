package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logger     LoggerConfig     `yaml:"logger"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	ArangoDB   ArangoDBConfig   `yaml:"arangodb"`
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
	RealIP     RealIPConfig     `yaml:"real_ip"`
	RequestLog RequestLogConfig `yaml:"request_log"`
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

type PostgresConfig struct {
	DSN      string `yaml:"dsn"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	MaxConns int32  `yaml:"max_conns"`
	MinConns int32  `yaml:"min_conns"`
}

type ArangoDBConfig struct {
	Endpoints []string `yaml:"endpoints"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	Database  string   `yaml:"database"`
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

	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		cfg.Postgres.DSN = dsn
	}
	cfg.Postgres.Username = os.Getenv("POSTGRES_USERNAME")
	cfg.Postgres.Password = os.Getenv("POSTGRES_PASSWORD")

	cfg.ArangoDB.Database = os.Getenv("ARANGODB_DATABASE")
	cfg.ArangoDB.Username = os.Getenv("ARANGODB_USERNAME")
	cfg.ArangoDB.Password = os.Getenv("ARANGODB_PASSWORD")

	return &cfg, nil
}
