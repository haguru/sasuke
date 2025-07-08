package postgres

import (
	"context"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/lib/pq" // PostgreSQL driver for database/sql

	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userrepo/constants"
	"github.com/haguru/sasuke/pkg/databases/postgres"
)

const(
	// Unique_ErrorCode is the PostgreSQL error code for unique constraint violations.
	Unique_ErrorCode = "23505" // PostgreSQL unique violation error code
)

var ensureSchemaSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
	`


type PostgresUserRepository struct {
	dbClient interfaces.DBClient // Now depends on the concrete postgres.PostgresDatabaseClient
}

// NewPostgresUserRepository returns a new PostgresUserRepository using the provided dbClient.
func NewPostgresUserRepository(dbClient interfaces.DBClient) (interfaces.UserRepository, error) {
	if dbClient == nil {
		return nil, fmt.Errorf("dbClient cannot be nil")
	}
	// Ensure the dbClient is of type PostgresDatabaseClient
	if _, ok := dbClient.(*postgres.PostgresDatabaseClient); !ok {
		return nil, fmt.Errorf("dbClient must be a PostgreSQL client")
	}
	// Return a new instance of PostgresUserRepository with the provided dbClient
	return &PostgresUserRepository{dbClient: dbClient}, nil
}

// AddUser inserts a user and returns the new user's ID.
func (r *PostgresUserRepository) AddUser(ctx context.Context, user models.User) (string, error) {
	// Convert models.User struct to map[string]interface{} for the generic client
	doc := make(map[string]interface{})
	err := mapstructure.Decode(user, &doc)
	if err != nil {
		return "", fmt.Errorf("failed to decode user model: %w", err)
	}

	// The client's InsertOne will generate the ID if not present
	insertedID, err := r.dbClient.InsertOne(ctx, constants.UsersCollection, doc)
	if err != nil {
		// PostgreSQL specific duplicate key error check (example for `pq` driver)
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == Unique_ErrorCode { // 23505 is unique_violation
			return "", fmt.Errorf("username '%s' already exists", user.Username)
		}
		return "", fmt.Errorf("failed to add user to PostgreSQL: %w", err)
	}
	strID, ok := insertedID.(string)
	if !ok {
		return "", fmt.Errorf("failed to assert inserted ID to string (expected UUID)")
	}
	return strID, nil
}

// GetUserByUsername retrieves a user and returns nil if the user is not found.
func (r *PostgresUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	filter := map[string]interface{}{"username": username}
	err := r.dbClient.FindOne(ctx, constants.UsersCollection, filter, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username from PostgreSQL: %w", err)
	}

	return &user, nil
}

// EnsureIndices creates a table and unique index and returns an error if the table creation fails.
func (r *PostgresUserRepository) EnsureIndices(ctx context.Context) error {
	return r.dbClient.EnsureSchema(ctx, constants.UsersCollection, ensureSchemaSQL)
}

// Close closes database connection and returns an error if the disconnection fails.
func (r *PostgresUserRepository) Close(ctx context.Context) error {
	return r.dbClient.Disconnect(ctx)
}
