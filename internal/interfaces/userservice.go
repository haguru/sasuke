package interfaces

import "context"

type UserService interface {
	RegisterUser(ctx context.Context, username, password string) (string, error)
	AuthenticateUser(ctx context.Context, username, password string) (bool, error)
}
