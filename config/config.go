package config

import (
	"os"

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
	Database       Database `yaml:"database" validate:"required"`
	PrivateKeyPath string   `yaml:"privateKeyPath" validate:"required"`
}

// Database holds the database configuration.
type Database struct {
	Type         string `yaml:"type" validate:"required"`
	Host         string `yaml:"host" validate:"required"`
	DatabaseName string `yaml:"database_name" validate:"required"`
	Port         int    `yaml:"port" validate:"required"`
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
