package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	App      AppConfig      `mapstructure:"app"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
	Port         string `mapstructure:"port"`
	ReadTimeout  string `mapstructure:"read_timeout"`
	WriteTimeout string `mapstructure:"write_timeout"`
	IdleTimeout  string `mapstructure:"idle_timeout"`
}

type DatabaseConfig struct {
	Type     string         `mapstructure:"type"` // memory, sqlite, postgres
	SQLite   SQLiteConfig   `mapstructure:"sqlite"`
	Postgres PostgresConfig `mapstructure:"postgres"`
}

type SQLiteConfig struct {
	Path string `mapstructure:"path"`
}

type PostgresConfig struct {
	URL string `mapstructure:"url"`
}

type AppConfig struct {
	BaseURL         string `mapstructure:"base_url"`
	ShortCodeLength int    `mapstructure:"short_code_length"`
}

type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/dove/")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")

	viper.SetDefault("database.type", "memory")
	viper.SetDefault("database.sqlite.path", "./data/dove.db")
	viper.SetDefault("database.postgres.url", "")

	viper.SetDefault("app.base_url", "http://localhost:8080")
	viper.SetDefault("app.short_code_length", 6)

	viper.SetDefault("logging.level", "info")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &config, nil
}

func (c *Config) GetDatabaseURL() string {
	switch c.Database.Type {
	case "sqlite":
		return c.Database.SQLite.Path
	case "postgres":
		return c.Database.Postgres.URL
	default:
		return ""
	}
}
