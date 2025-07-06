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
	// MongoDB typically generates ObjectID, so we'll let it do that.
	// We'll pass a struct with _id field or let Mongo generate
	mongoUser := struct {
		ID       primitive.ObjectID `bson:"_id,omitempty"`
		Username string             `bson:"username"`
		Password string             `bson:"password"`
	}{
		ID:       primitive.NewObjectID(), // Generate ObjectID here
		Username: user.Username,
		Password: user.Password, // Hashed password
	}

	insertedID, err := r.dbClient.InsertOne(ctx, constants.UsersCollection, mongoUser)
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
func (r *MongoUserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	// Validate the username
	if len(username) == 0 || len(username) > MAXLENGTH_USERNAME {
		return nil, fmt.Errorf("invalid username: must be between 1 and %d characters", MAXLENGTH_USERNAME)
	}

	var mongoUser struct { // Temporary struct to decode MongoDB BSON
		ID       primitive.ObjectID `bson:"_id,omitempty"`
		Username string             `bson:"username"`
		Password string             `bson:"password"`
	}

	filter := bson.M{"username": username}
	err := r.dbClient.FindOne(ctx, constants.UsersCollection, filter, &mongoUser)
	if err != nil {
		// If FindOne returns non-nil error, it's a database issue. If no document, it returns nil error.
		if err.Error() == mongosdk.ErrNoDocuments.Error() { // Check for specific no documents error if FindOne doesn't convert it to nil
			return nil, nil // User not found
		}
		return nil, fmt.Errorf("failed to get user by username from MongoDB: %w", err)
	}
	if mongoUser.ID.IsZero() { // If ID is zero, it means FindOne returned no document but didn't error out explicitly.
		return nil, nil
	}

	return &models.User{
		ID:       mongoUser.ID.Hex(),
		Username: mongoUser.Username,
		Password: mongoUser.Password,
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
