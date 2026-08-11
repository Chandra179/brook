package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logger     LoggerConfig     `yaml:"logger"`
	Profiling  ProfilingConfig  `yaml:"profiling"`
	Tracing    TracingConfig    `yaml:"tracing"`
	Postgres   PostgresConfig   `yaml:"postgres"`
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

type ProfilingConfig struct {
	ServerAddress   string `yaml:"server_address"`
	ApplicationName string `yaml:"application_name"`
	Enabled         bool   `yaml:"enabled"`
}

type TracingConfig struct {
	OTLPEndpoint    string `yaml:"otlp_endpoint"`
	ApplicationName string `yaml:"application_name"`
	Enabled         bool   `yaml:"enabled"`
}

type PostgresConfig struct {
	DSN      string `yaml:"dsn"`
	MaxConns int32  `yaml:"max_conns"`
	MinConns int32  `yaml:"min_conns"`
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

	if addr := os.Getenv("PYROSCOPE_SERVER_ADDRESS"); addr != "" {
		cfg.Profiling.ServerAddress = addr
	}

	if addr := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); addr != "" {
		cfg.Tracing.OTLPEndpoint = addr
	}

	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		cfg.Postgres.DSN = dsn
	}

	return &cfg, nil
}
