package postgres

import (
	"context"
	"fmt"

	"github.com/lib/pq" // PostgreSQL driver for database/sql

	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userrepo/constants"
	"github.com/haguru/sasuke/pkg/databases/postgres"
)

// PostgresUserRepository implements UserRepository for PostgreSQL databases.
type PostgresUserRepository struct {
	dbClient *postgres.PostgresDatabaseClient // Now depends on the concrete postgres.PostgresDatabaseClient
}

// NewPostgresUserRepository creates a new PostgreSQL repository instance.
func NewPostgresUserRepository(dbClient *postgres.PostgresDatabaseClient) *PostgresUserRepository {
	return &PostgresUserRepository{dbClient: dbClient}
}

// Addmodels.User saves a new user to PostgreSQL via DBClient.
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

// Getmodels.UserBymodels.Username retrieves a user from PostgreSQL via DBClient.
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
func (r *PostgresUserRepository) EnsureIndices(ctx context.Context) error {
	return r.dbClient.EnsurePostgresTable(ctx, constants.UsersCollection)
}

// Close closes the PostgreSQL database connection.
func (r *PostgresUserRepository) Close(ctx context.Context) error {
	return r.dbClient.Disconnect(ctx)
}
