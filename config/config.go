package config

import (
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v3"
)

const (
	CONFIG_PATH = "./res/config.yaml"
)

// ServiceConfig holds the configuration for the service.
type ServiceConfig struct {
	ServiceName    string   `yaml:"service_name" validate:"required"`
	LogLevel       string   `yaml:"loglevel" validate:"required"`
	Host           string   `yaml:"host" validate:"required"`
	Port           string   `yaml:"port" validate:"required"`
	PrivateKeyPath string   `yaml:"private_key_path" validate:"required"`
	Database       Database `yaml:"database" validate:"required"`
}

type Database struct {
	Type string `yaml:"type" validate:"required"`
	// For MongoDB
	MongoDB MongoDBConfig `yaml:"mongodb_config" validate:"omitempty"`
	// For PostgreSQL
	Postgres PostgresConfig `yaml:"postgres_config" validate:"omitempty"`
}

// Database holds the database configuration.
type MongoDBConfig struct {
	Host             string             `yaml:"host" validate:"required"`
	DatabaseName     string             `yaml:"database_name" validate:"required"`
	Port             int                `yaml:"port" validate:"required"`
	Timeout          time.Duration      `yaml:"timeout"`
	Options          MongoServerOptions `yaml:"mongo_server_options"`
	ValidCollections []string           `yaml:"valid_collections" validate:"required"`
	ValidFields      []string           `yaml:"valid_fields" validate:"required"`
}

type PostgresConfig struct {
	Host         string                `yaml:"host" validate:"required"`
	Port         int                   `yaml:"port" validate:"required"`
	DatabaseName string                `yaml:"database_name" validate:"required"`
	Options      PostgresServerOptions `yaml:"postgres_server_options"`
	ValidTables  []string              `yaml:"valid_tables" validate:"required"`
	ValidFields  []string              `yaml:"valid_fields" validate:"required"`
}

type MongoServerOptions struct {
	APIVersion           string `yaml:"api_version"`
	SetStrict            bool   `yaml:"set_strict"`
	SetDeprecationErrors bool   `yaml:"set_deprecation_errors"`
}

type PostgresServerOptions struct {
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type ValidFields struct {
	Fields []string `yaml:"fields" validate:"required"`
}

// ReadLocalConfig reads the service configuration from a YAML file at the specified path.
// It unmarshals the YAML content into a ServiceConfig struct and returns it.
// If there is an error reading the file or unmarshaling the content, it returns an error.
func ReadLocalConfig(configPath string) (*ServiceConfig, error) {
	config := &ServiceConfig{}

	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func BuildServerAPIOptions(cfg MongoServerOptions) *options.ServerAPIOptions {
	opts := options.ServerAPI(options.ServerAPIVersion(cfg.APIVersion))
	opts.SetStrict(cfg.SetStrict)
	opts.SetDeprecationErrors(cfg.SetDeprecationErrors)

	return opts
}

func ListToMap(list []string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range list {
		result[item] = true
	}
	return result
}
