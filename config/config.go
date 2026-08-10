package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Profiling  ProfilingConfig  `yaml:"profiling"`
	Logger     LoggerConfig     `yaml:"logger"`
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
	return &cfg, nil
}
