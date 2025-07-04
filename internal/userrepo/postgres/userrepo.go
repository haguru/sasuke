package postgres

import (
	"context"
	"fmt"

	"github.com/lib/pq" // PostgreSQL driver for database/sql

	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userrepo/constants"
	"github.com/haguru/sasuke/pkg/databases/postgres"
)

var (
	ensureSchemaSQL = `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
	`
)

// PostgresUserRepository implements UserRepository for PostgreSQL databases.
type PostgresUserRepository struct {
	dbClient interfaces.DBClient // Now depends on the concrete postgres.PostgresDatabaseClient
}

// NewPostgresUserRepository creates a new PostgreSQL repository instance.
// It takes a postgres.PostgresDatabaseClient as a parameter to interact with the database.
// This function is used to initialize the repository with a specific database client.
// It returns a pointer to PostgresUserRepository, which implements the UserRepository interface.
// This allows the repository to perform operations like adding and retrieving users from a PostgreSQL database.
// The dbClient is expected to be already configured with the necessary connection details.
// This function is typically called during the application setup phase, where the database client is created and passed to the repository.
// It ensures that the repository has a valid database client to perform operations on the PostgreSQL database.
// Example usage:
// dbClient := postgres.NewPostgresDatabaseClient("your_connection_string")
// userRepo := NewPostgresUserRepository(dbClient)
// This function is part of the user repository package and is used to create a new instance of
// PostgresUserRepository, which is responsible for handling user-related operations in a PostgreSQL database.
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

// AddUser saves a new user to PostgreSQL via DBClient.
// It returns the user's ID as a string if successful, or an error if the operation fails.
func (r *PostgresUserRepository) AddUser(ctx context.Context, user models.User) (string, error) {
	// Convert models.User struct to map[string]interface{} for the generic client
	doc := map[string]interface{}{
		"username": user.Username,
		"password": user.Password,
	}
	// The client's InsertOne will generate the ID if not present

	insertedID, err := r.dbClient.InsertOne(ctx, constants.UsersCollection, doc)
	if err != nil {
		// PostgreSQL specific duplicate key error check (example for `pq` driver)
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" { // 23505 is unique_violation
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

// GetUserByUsername retrieves a user from PostgreSQL via DBClient.
// It returns nil if the user is not found.
func (r *PostgresUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	filter := map[string]interface{}{"username": username}
	err := r.dbClient.FindOne(ctx, constants.UsersCollection, filter, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username from PostgreSQL: %w", err)
	}
	if user.ID == "" { // If ID is empty after FindOne, it means no user was found.
		return nil, nil
	}
	return &user, nil
}

// EnsureIndices creates a table and unique index for username in PostgreSQL.
// This method is called to ensure the necessary indices are in place.
// It returns an error if the table creation fails.
func (r *PostgresUserRepository) EnsureIndices(ctx context.Context) error {
	return r.dbClient.EnsureSchema(ctx, constants.UsersCollection, ensureSchemaSQL)
}

// Close closes the PostgreSQL database connection.
// It is important to call this method when the repository is no longer needed.
// It returns an error if the disconnection fails.
func (r *PostgresUserRepository) Close(ctx context.Context) error {
	return r.dbClient.Disconnect(ctx)
}
