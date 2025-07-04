package config

import (
	"reflect"
	"testing"
)

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
				ServiceName: "sasuke",
				Host:        "localhost",
				Port:        "50051",
				LogLevel:    "DEBUG",
				PrivateKeyPath: "./res/sharingan_key.pem",
				// Assuming the database configuration is also part of the config file
				Database: Database{
					Host:         "localhost",
					Port:         27017,
					DatabaseName: "sasukeDB",
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
