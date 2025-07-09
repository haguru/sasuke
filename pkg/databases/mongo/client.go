package mongo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/pkg/helper"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	MAXPOOLSIZE = 20
	IDFIELD     = "_id"
)

// MongoDBClient implements the interfaces.DBClient interface for MongoDB operations.
type MongoDBClient struct {
	ServerOpts       *options.ServerAPIOptions
	client           *mongo.Client
	db               *mongo.Database
	timeout          time.Duration
	validCollections map[string]bool // A map to validate collection names
	validFields      map[string]bool // A map to validate field names
	Logger           interfaces.Logger
}

// NewMongoDB returns a interface for db client and error if it occurs
func NewMongoDB(dbConfig *config.MongoDBConfig, logger interfaces.Logger) (interfaces.DBClient, error) {
	funcName := helper.GetFuncName()
	logger.Debug("Entering", "func", funcName)
	db := &MongoDBClient{
		timeout:          dbConfig.Timeout,
		ServerOpts:       config.BuildServerAPIOptions(dbConfig.Options),
		validCollections: config.ListToMap(dbConfig.ValidCollections),
		validFields:      config.ListToMap(dbConfig.ValidFields),
		Logger:           logger,
	}
	logger.Info("MongoDBClient created", "func", funcName)
	return db, nil
}

// Connect establishes a connection to the MongoDB database using the provided DSN (Data Source Name).
func (m *MongoDBClient) Connect(ctx context.Context, dsn string) error {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "dsn", dsn)
	m.Logger.Info("MongoDBClient Connecting", "func", funcName, "dsn", dsn)

	// Validate the DSN format
	if dsn == "" {
		m.Logger.Debug("DSN is empty", "func", funcName)
		return fmt.Errorf("MongoDBClient: DSN is empty")
	}
	if !strings.HasPrefix(dsn, "mongodb://") && !strings.HasPrefix(dsn, "mongodb+srv://") {
		m.Logger.Debug("Invalid DSN format", "func", funcName)
		return fmt.Errorf("MongoDBClient: Invalid DSN format, expected 'mongodb://' or 'mongodb+srv://'")
	}

	// Set a timeout for the connection
	if m.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
		defer cancel()
		m.Logger.Debug("Set connection timeout", "func", funcName, "timeout", m.timeout)
	}
	clientOptions := options.Client().ApplyURI(dsn)

	// Set the server API options if provided
	if m.ServerOpts != nil {
		clientOptions.SetServerAPIOptions(m.ServerOpts)
		m.Logger.Debug("Set ServerAPIOptions", "func", funcName)
	}
	// Set the maximum pool size
	clientOptions.SetMaxPoolSize(MAXPOOLSIZE)
	m.Logger.Debug("Set MaxPoolSize", "func", funcName, "maxPoolSize", MAXPOOLSIZE)

	// Set read preference to primaryPreferred
	clientOptions.SetReadPreference(readpref.PrimaryPreferred())
	m.Logger.Debug("Set ReadPreference", "func", funcName, "readPreference", "PrimaryPreferred")

	// Connect to the MongoDB server
	var err error
	m.client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Check if the connection is successful by pinging the server
	m.Logger.Info("MongoDBClient Pinging MongoDB server...", "func", funcName)
	if err = m.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("MongoDBClient: Failed to connect to MongoDB server: %v", err)
	}
	m.Logger.Info("MongoDBClient Connected to MongoDB server successfully.", "func", funcName)

	// Extract the database name from the DSN
	databaseName, err := m.getDBNameFromMongoDSN(dsn)
	if err != nil {
		return fmt.Errorf("MongoDBClient: Failed to extract database name from datasource name(dsn): %v", err)
	}

	m.db = m.client.Database(databaseName)
	m.Logger.Debug("Set database", "func", funcName, "database", databaseName)
	return nil
}

// Disconnect closes the connection to the MongoDB database.
func (m *MongoDBClient) Disconnect(ctx context.Context) error {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName)
	m.Logger.Info("MongoDBClient Disconnecting...", "func", funcName)
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}
	return nil
}

// InsertOne inserts a document and returns its ID.
func (m *MongoDBClient) InsertOne(ctx context.Context, collectionName string, document interfaces.Document) (interface{}, error) {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName)
	m.Logger.Info("MongoDBClient Inserting one", "func", funcName, "collection", collectionName)

	if !m.validCollections[collectionName] {
		m.Logger.Debug("Invalid collection name", "func", funcName, "collection", collectionName)
		return nil, fmt.Errorf("MongoDBClient: Invalid collection name: %s", collectionName)
	}

	if collectionName == "" {
		m.Logger.Debug("Collection name is empty", "func", funcName)
		return nil, fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}

	// Sanitize document
	sanitizedDocument := m.sanitizeDocument(document)
	m.Logger.Debug("Sanitized document", "func", funcName, "document", sanitizedDocument)

	res, err := m.db.Collection(collectionName).InsertOne(ctx, sanitizedDocument)
	if err != nil {
		return nil, fmt.Errorf("MongoDBClient: Failed to insert one into %s: %v", collectionName, err)
	}

	m.Logger.Debug("InsertOne successful", "func", funcName, "insertedID", res.InsertedID)
	return res.InsertedID, nil
}

// FindOne retrieves a single document from the specified collection using a filter.
func (m *MongoDBClient) FindOne(ctx context.Context, collectionName string, filter interfaces.Document, result interfaces.Document) error {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName, "filter", filter)
	m.Logger.Info("MongoDBClient Finding one", "func", funcName, "collection", collectionName, "filter", filter)

	if !m.validCollections[collectionName] {
		m.Logger.Debug("Invalid collection name", "func", funcName, "collection", collectionName)
		return fmt.Errorf("MongoDBClient: Invalid collection name: %s", collectionName)
	}

	if collectionName == "" {
		m.Logger.Debug("Collection name is empty", "func", funcName)
		return fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}

	// Sanitize filter
	sanitizedFilter := m.sanitizeDocument(filter)
	m.Logger.Debug("Sanitized filter", "func", funcName, "filter", sanitizedFilter)

	err := m.db.Collection(collectionName).FindOne(ctx, sanitizedFilter).Decode(result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("MongoDBClient: No document found in %s with filter: %v", collectionName, filter)
		}
		return fmt.Errorf("MongoDBClient: Failed to find one in %s with filter: %v: %v", collectionName, filter, err)
	}

	m.Logger.Debug("FindOne successful", "func", funcName)
	return nil
}

// FindMany retrieves multiple documents from the specified collection.
func (m *MongoDBClient) FindMany(ctx context.Context, collectionName string, filter interfaces.Document) ([]interfaces.Document, error) {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName, "filter", filter)
	m.Logger.Info("MongoDBClient Finding many", "func", funcName, "collection", collectionName, "filter", filter)

	if !m.validCollections[collectionName] {
		m.Logger.Debug("Invalid collection name", "func", funcName, "collection", collectionName)
		return nil, fmt.Errorf("MongoDBClient: Invalid collection name: %s", collectionName)
	}

	if collectionName == "" {
		m.Logger.Debug("Collection name is empty", "func", funcName)
		return nil, fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}

	// Sanitize filter
	sanitizedFilter := m.sanitizeDocument(filter)
	m.Logger.Debug("Sanitized filter", "func", funcName, "filter", sanitizedFilter)

	cursor, err := m.db.Collection(collectionName).Find(ctx, sanitizedFilter)
	if err != nil {
		return nil, fmt.Errorf("MongoDBClient: Finding many in %s with filter: %v failed: %v", collectionName, sanitizedFilter, err)
	}

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			m.Logger.Error("MongoDBClient: Failed to close cursor", "func", funcName, "error", err)
		}
	}()

	var results []interfaces.Document
	for cursor.Next(ctx) {
		var doc map[string]interface{}
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("MongoDBClient: Failed to decode cursor: %v", err)
		}
		results = append(results, doc)
	}

	m.Logger.Debug("FindMany successful", "func", funcName, "count", len(results))
	return results, nil
}

// UpdateOne modifies a single document in the specified collection using a filter and update document.
func (m *MongoDBClient) UpdateOne(ctx context.Context, collectionName string, filter interfaces.Document, update interfaces.Document) (int64, error) {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName, "filter", filter, "update", update)
	m.Logger.Info("MongoDBClient Updating one", "func", funcName, "collection", collectionName, "filter", filter, "update", update)

	if !m.validCollections[collectionName] {
		m.Logger.Debug("Invalid collection name", "func", funcName, "collection", collectionName)
		return 0, fmt.Errorf("MongoDBClient: Invalid collection name: %s", collectionName)
	}

	if collectionName == "" {
		m.Logger.Debug("Collection name is empty", "func", funcName)
		return 0, fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}

	// Sanitize filter and update
	sanitizedFilter := m.sanitizeDocument(filter)
	sanitizedUpdate := m.sanitizeDocument(update)
	m.Logger.Debug("Sanitized filter and update", "func", funcName, "filter", sanitizedFilter, "update", sanitizedUpdate)

	res, err := m.db.Collection(collectionName).UpdateOne(ctx, sanitizedFilter, sanitizedUpdate)
	if err != nil {
		return 0, fmt.Errorf("MongoDBClient: Failed updating one in %s with filter %v, update %v: %v", collectionName, sanitizedFilter, sanitizedUpdate, err)
	}

	m.Logger.Debug("UpdateOne successful", "func", funcName, "modifiedCount", res.ModifiedCount)
	return res.ModifiedCount, nil
}

// DeleteOne removes a single document from the specified collection using a filter.
func (m *MongoDBClient) DeleteOne(ctx context.Context, collectionName string, filter interfaces.Document) (int64, error) {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName, "filter", filter)
	m.Logger.Info("MongoDBClient Deleting one", "func", funcName, "collection", collectionName, "filter", filter)

	if !m.validCollections[collectionName] {
		m.Logger.Debug("Invalid collection name", "func", funcName, "collection", collectionName)
		return 0, fmt.Errorf("MongoDBClient: Invalid collection name: %s", collectionName)
	}

	if collectionName == "" {
		m.Logger.Debug("Collection name is empty", "func", funcName)
		return 0, fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}

	// Sanitize filter
	sanitizedFilter := m.sanitizeDocument(filter)
	m.Logger.Debug("Sanitized filter", "func", funcName, "filter", sanitizedFilter)

	res, err := m.db.Collection(collectionName).DeleteOne(ctx, sanitizedFilter)
	if err != nil {
		return 0, fmt.Errorf("MongoDBClient: Failed deleting one from %s with filter %v: %v", collectionName, sanitizedFilter, err)
	}

	m.Logger.Debug("DeleteOne successful", "func", funcName, "deletedCount", res.DeletedCount)
	return res.DeletedCount, nil
}

// DeleteMany removes multiple documents from a collection using a filter.
func (m *MongoDBClient) DeleteMany(ctx context.Context, collectionName string, filter interfaces.Document) (int64, error) {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName, "filter", filter)
	m.Logger.Info("MongoDBClient Deleting many", "func", funcName, "collection", collectionName, "filter", filter)

	if !m.validCollections[collectionName] {
		m.Logger.Debug("Invalid collection name", "func", funcName, "collection", collectionName)
		return 0, fmt.Errorf("MongoDBClient: Invalid collection name: %s", collectionName)
	}

	if collectionName == "" {
		m.Logger.Debug("Collection name is empty", "func", funcName)
		return 0, fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}

	// Sanitize filter
	sanitizedFilter := m.sanitizeDocument(filter)
	m.Logger.Debug("Sanitized filter", "func", funcName, "filter", sanitizedFilter)

	res, err := m.db.Collection(collectionName).DeleteMany(ctx, sanitizedFilter)
	if err != nil {
		return 0, fmt.Errorf("MongoDBClient: Failed Deleting many from %s with filter %v: %v", collectionName, sanitizedFilter, err)
	}

	m.Logger.Debug("DeleteMany successful", "func", funcName, "deletedCount", res.DeletedCount)
	return res.DeletedCount, nil
}

// Ping verifies the MongoDB connection health using a ping command.
func (m *MongoDBClient) Ping(ctx context.Context) error {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName)
	m.Logger.Info("Pinging...", "func", funcName)
	return m.client.Ping(ctx, nil)
}

// getDBNameFromMongoDSN extracts the database name from a MongoDB DSN.
func (m *MongoDBClient) getDBNameFromMongoDSN(dsn string) (string, error) {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "dsn", dsn)
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to parse MongoDB DSN: %w", err)
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return "", fmt.Errorf("no database name found in MongoDB DSN path: %s", dsn)
	}

	if idx := strings.Index(dbName, "/"); idx != -1 {
		dbName = dbName[:idx]
	}

	m.Logger.Debug("Extracted database name", "func", funcName, "database", dbName)
	return dbName, nil
}

// EnsureSchema creates the required index on the specified collection using the provided mongo.IndexModel.
func (m *MongoDBClient) EnsureSchema(ctx context.Context, collectionName string, schema interfaces.Document) error {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName, "collection", collectionName)
	// verify m.db is not nil
	if m.db == nil {
		return fmt.Errorf("MongoDBClient is not connected to a database")
	}

	// Ensure schema is a mongo.IndexModel
	if schema == nil {
		return fmt.Errorf("EnsureSchema expects schema to be a mongo.IndexModel")
	}

	// Type assertion to mongo.IndexModel
	model, ok := schema.(mongo.IndexModel)
	if !ok {
		return fmt.Errorf("EnsureSchema: expected mongo.IndexModel for MongoDB")
	}
	// Create the index on the specified collection
	collection := m.db.Collection(collectionName)
	_, err := collection.Indexes().CreateOne(ctx, model)
	if err != nil {
		return fmt.Errorf("failed to create index: %w",err)
	}

	m.Logger.Debug("index created successfully", "func", funcName)
	return nil
}

// SanitizeDocument ensures that the document does not contain any malicious content.
func (m *MongoDBClient) sanitizeDocument(document interfaces.Document) interfaces.Document {
	funcName := helper.GetFuncName()
	m.Logger.Debug("Entering", "func", funcName)
	m.Logger.Info("MongoDBClient: Sanitizing document...", "func", funcName)

	// Ensure the document is not nil
	if document == nil {
		m.Logger.Debug("Document is nil", "func", funcName)
		return nil
	}

	// Create a sanitized copy of the document
	sanitized := make(map[string]interface{})
	// Assert that document is a map[string]interface{} before iterating
	docMap, ok := document.(map[string]interface{}) // bson.M is a type alias for map[string]interface{}
	if !ok {
		m.Logger.Error("Document is not of type map[string]interface{}, cannot sanitize", "func", funcName)
		return nil
	}

	for key, value := range docMap {
		// Skip the ID field to prevent overwriting or exposing it
		if key == IDFIELD {
			m.Logger.Debug("Skipping ID field", "func", funcName, "field", key)
			continue
		}

		// Ensure the key is a valid field name and does not contain special characters
		if _, ok := m.validFields[key]; !ok || strings.ContainsAny(key, "$.") {
			m.Logger.Info("Skipping invalid or unsafe field name", "func", funcName, "field", key)
			continue
		}

		// Add the sanitized key-value pair to the sanitized document
		sanitized[key] = value
	}

	m.Logger.Debug("Sanitization complete", "func", funcName, "sanitized", sanitized)
	return sanitized
}
