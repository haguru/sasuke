package interfaces

import (
	"context"

	"github.com/haguru/sasuke/internal/models"
)

// UserRepository defines the contract for storing and retrieving User data.
// This interface remains the same as it's database-agnostic.
type UserRepository interface {
	AddUser(ctx context.Context, user models.User) (string, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	EnsureIndices(ctx context.Context) error
	Close(ctx context.Context) error
}
