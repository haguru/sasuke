package mongo

import (
	"context"
	"fmt"
	"strings"

	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userrepo/constants"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/go-viper/mapstructure/v2"
	mongoClient "github.com/haguru/sasuke/pkg/databases/mongo"
	mongosdk "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	MaxLengthUserName     = 64
	DuplicateKeyErrorCode = "E11000 duplicate key error"
)

type MongoUserRepository struct {
	dbClient interfaces.DBClient // Here we use the concrete Mongo implementation of DBClient
}

// NewMongoUserRepository returns a new MongoUserRepository.
func NewMongoUserRepository(dbClient interfaces.DBClient) (interfaces.UserRepository, error) {
	if dbClient == nil {
		return nil, fmt.Errorf("dbClient cannot be nil")
	}
	// Ensure the dbClient is of type MongoDBClient
	if _, ok := dbClient.(*mongoClient.MongoDBClient); !ok {
		return nil, fmt.Errorf("dbClient must be a MongoDB client")
	}
	return &MongoUserRepository{dbClient: dbClient}, nil
}

// AddUser saves a new user to MongoDB via DBClient.
func (r *MongoUserRepository) AddUser(ctx context.Context, user models.User) (string, error) {
	usermap := make(map[string]interface{})
	err := mapstructure.Decode(user, &usermap)
	if err != nil {
		return "", fmt.Errorf("failed to decode user model: %w", err)
	}

	insertedID, err := r.dbClient.InsertOne(ctx, constants.UsersCollection, usermap)
	if err != nil {
		if strings.Contains(err.Error(), DuplicateKeyErrorCode) { // MongoDB specific duplicate key error check
			return "", fmt.Errorf("username '%s' already exists", user.Username)
		}
		return "", fmt.Errorf("failed to add user to MongoDB: %w", err)
	}

	objID, ok := insertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("failed to assert inserted ID to ObjectID")
	}
	return objID.Hex(), nil
}

// GetUserByUsername fetches a user by username, returns nil if not found.
func (r *MongoUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	filter := map[string]any{"username": username}
	err := r.dbClient.FindOne(ctx, constants.UsersCollection, filter, &user)
	if err != nil {
		// If FindOne returns non-nil error, it's a database issue. If no document, it returns nil error.
		if err.Error() == mongosdk.ErrNoDocuments.Error() { // Check for specific no documents error if FindOne doesn't convert it to nil
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by username from MongoDB: %w", err)
	}

	return &user, nil
}

// EnsureIndices creates a unique index for username in MongoDB.
func (r *MongoUserRepository) EnsureIndices(ctx context.Context) error {
	indexModel := mongosdk.IndexModel{
		Keys:    bson.M{"username": 1},
		Options: options.Index().SetUnique(true),
	}
	// Call MongoDB-specific method for index creation.
	return r.dbClient.EnsureSchema(ctx, constants.UsersCollection, indexModel)
}

// Close disconnects the MongoDB client.
func (r *MongoUserRepository) Close(ctx context.Context) error {
	return r.dbClient.Disconnect(ctx)
}
