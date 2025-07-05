package models

import (
	"reflect"
	"testing"
)

func TestNewUser(t *testing.T) {
	type args struct {
		username string
		password string
	}
	tests := []struct {
		name string
		args args
		want *User
	}{
		{
			name: "Create new user with valid username and password",
			args: args{
				username: "testuser",
				password: "testpass",
			},
			want: &User{
				ID:	   "",	// ID is left empty for the caller to set or for the database to populate
				Username: "testuser",
				Password: "testpass",
			},
		},
		{
			name: "Create new user with empty username and password",
			args: args{
				username: "",
				password: "",
			},
			want: &User{
				ID:       "",
				Username: "",
				Password: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUser(tt.args.username, tt.args.password); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUser() = %v, want %v", got, tt.want)
			}
		})
	}
}
