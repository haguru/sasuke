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
	MAXLENGTH_USERNAME = 64 // Maximum length for username
)

// MongoUserRepository implements UserRepository using the generic DBClient.
type MongoUserRepository struct {
	dbClient interfaces.DBClient // Here we use the concrete Mongo implementation of DBClient
}

// NewMongoUserRepository creates a new MongoDB repository instance.
// It takes a concrete mongo.MongoDBClient.
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
	// convert models.User struct to a MongoDB BSON document
	usermap := make(map[string]interface{})
	err := mapstructure.Decode(user, &usermap)
	if err != nil {
		return "", fmt.Errorf("failed to decode user model: %w", err)
	}

	insertedID, err := r.dbClient.InsertOne(ctx, constants.UsersCollection, usermap)
	if err != nil {
		if strings.Contains(err.Error(), "E11000 duplicate key error") { // MongoDB specific duplicate key error check
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

// GetUserByUsername retrieves a user from MongoDB via DBClient.
// validation of username is done before querying.
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

	return &models.User{
		Username: user.Username,
		Password: user.Password,
	}, nil
}

// EnsureIndices creates unique indices for username in MongoDB (uses direct client helper).
// Note: This calls a specific helper on mongo.MongoDBClient, as DBClient doesn't have a generic index method.
func (r *MongoUserRepository) EnsureIndices(ctx context.Context) error {
	indexModel := mongosdk.IndexModel{
		Keys:    bson.M{"username": 1},
		Options: options.Index().SetUnique(true),
	}
	// Here, we have to call a MongoDB-specific method provided by our concrete mongo.MongoDBClient
	// because the generic DBClient interface doesn't expose index creation.
	return r.dbClient.EnsureSchema(ctx, constants.UsersCollection, indexModel)
}

// Close disconnects the MongoDB client.
func (r *MongoUserRepository) Close(ctx context.Context) error {
	return r.dbClient.Disconnect(ctx)
}
