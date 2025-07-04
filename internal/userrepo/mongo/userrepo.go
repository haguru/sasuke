package mongo

import (
	"context"
	"fmt"
	"strings"

	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userrepo/constants"
	"github.com/haguru/sasuke/pkg/databases/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	mongosdk "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoUserRepository implements UserRepository using the generic DBClient.
type MongoUserRepository struct {
	dbClient *mongo.MongoDBClient // Here we use the concrete Mongo implementation of DBClient
}

// NewMongoUserRepository creates a new MongoDB repository instance.
// It takes a concrete mongo.MongoDBClient.
func NewMongoUserRepository(dbClient *mongo.MongoDBClient) *MongoUserRepository {
	return &MongoUserRepository{dbClient: dbClient}
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
	return r.dbClient.EnsureMongoIndex(ctx, constants.UsersCollection, indexModel)
}

// Close disconnects the MongoDB client.
func (r *MongoUserRepository) Close(ctx context.Context) error {
	return r.dbClient.Disconnect(ctx)
}
