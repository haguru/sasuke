package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMain(m *testing.M) {
	invalidYamlPath := "./invalid_config.yaml"
	invalidContent := []byte("invalid: [unclosed_list\nanother: value")

	// Create invalid YAML file
	if err := os.WriteFile(invalidYamlPath, invalidContent, 0600); err != nil {
		panic("failed to create invalid YAML file: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Clean up
	os.Remove(invalidYamlPath)

	os.Exit(code)
}

func TestReadLocalConfig(t *testing.T) {
	type args struct {
		configPath string
	}
	tests := []struct {
		name    string
		args    args
		want    *ServiceConfig
		wantErr bool
	}{
		{
			name: "successful",
			args: args{
				configPath: "../res/config.yaml",
			},
			want: &ServiceConfig{
				ServiceName:    "sasuke",
				Host:           "localhost",
				Port:           "50051",
				LogLevel:       "DEBUG",
				PrivateKeyPath: "./res/sharingan_key.pem",
				// Assuming the database configuration is also part of the config file
				Database: Database{
					Type: "mongo",
					MongoDB: MongoDBConfig{
						DSN:              "mongodb://localhost:27017/sasukeDB",
						DatabaseName:     "sasukeDB",
						Timeout:          10 * time.Second,
						ValidCollections: []string{"users"},
						ValidFields:      []string{"username", "hashed_password"},
						Options: MongoServerOptions{
							APIVersion:           "1",
							SetStrict:            true,
							SetDeprecationErrors: true,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "file does not exist",
			args: args{
				configPath: "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid YAML file",
			args: args{
				configPath: "./invalid_config.yaml",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadLocalConfig(tt.args.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadLocalConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadLocalConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildServerAPIOptions(t *testing.T) {
	type args struct {
		cfg MongoServerOptions
	}
	tests := []struct {
		name string
		args args
		want *options.ServerAPIOptions
	}{
		{
			name: "default options",
			args: args{
				cfg: MongoServerOptions{
					APIVersion:           "1",
					SetStrict:            true,
					SetDeprecationErrors: true,
				},
			},
			want: options.ServerAPI(options.ServerAPIVersion("1")).
				SetStrict(true).
				SetDeprecationErrors(true),
		},
		{
			name: "empty options",
			args: args{
				cfg: MongoServerOptions{
					APIVersion:           "",
					SetStrict:            false,
					SetDeprecationErrors: false,
				},
			},
			want: options.ServerAPI(options.ServerAPIVersion("")).
				SetStrict(false).
				SetDeprecationErrors(false),
		},
		{
			name: "custom options",
			args: args{
				cfg: MongoServerOptions{
					APIVersion:           "2",
					SetStrict:            true,
					SetDeprecationErrors: false,
				},
			},
			want: options.ServerAPI(options.ServerAPIVersion("2")).
				SetStrict(true).
				SetDeprecationErrors(false),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildServerAPIOptions(tt.args.cfg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildServerAPIOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
