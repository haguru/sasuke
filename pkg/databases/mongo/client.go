package mongo

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/haguru/sasuke/internal/interfaces"

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
	Uri        string
	Host       string
	Port       int
	ServerOpts *options.ServerAPIOptions
	client     *mongo.Client
	db         *mongo.Database
	timeout    time.Duration
}



// NewMongoDB returns a interface for db client and error if it occurs
func NewMongoDB(timeout time.Duration, opts *options.ServerAPIOptions) (interfaces.DBClient, error) {
	db := &MongoDBClient{
		timeout:    timeout,
		ServerOpts: opts,
	}

	return db, nil
}

// Connect establishes a connection to the MongoDB database using the provided DSN (Data Source Name).
// It initializes the MongoDB client and sets the database instance.
// The DSN should be in the format "mongodb://<host>:<port>/<database>".
// The function extracts the database name from the DSN and sets it as the active database for the client.
func (m *MongoDBClient) Connect(ctx context.Context, dsn string) error {
	fmt.Printf("MongoDBClient: Connecting to %s...\n", dsn)

	// Set a timeout for the connection
	if m.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, m.timeout)
		defer cancel()
	}
	clientOptions := options.Client().ApplyURI(dsn)

	// Set the server API options if provided
	if m.ServerOpts != nil {
		clientOptions.SetServerAPIOptions(m.ServerOpts)
	}
	// Set the maximum pool size
	clientOptions.SetMaxPoolSize(MAXPOOLSIZE)

	// Set the read preference to primarypreferred
	// This ensures that read operations are directed to the primary node in a replica set.
	// If the primary is unavailable, it will read from secondary nodes.
	// This is useful for applications that require strong consistency.
	// If you want to read from secondary nodes, you can change this to readpref.SecondaryPreferred()
	// If you want to read from the primary node only, you can use readpref.Primary()
	// If you want to read from the nearest node, you can use readpref.Nearest()
	// For this example, we will use readpref.Primary() to ensure that all
	clientOptions.SetReadPreference(readpref.PrimaryPreferred())

	// Connect to the MongoDB server
	var err error
	m.client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// Check if the connection is successful by pinging the server
	fmt.Println("MongoDBClient: Pinging MongoDB server...")
	if err = m.client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("MongoDBClient: Failed to connect to MongoDB server: %v", err)
	}
	fmt.Println("MongoDBClient: Connected to MongoDB server successfully.")

	// Validate the DSN format
	if dsn == "" {
		return fmt.Errorf("MongoDBClient: DSN is empty")
	}
	if !strings.HasPrefix(dsn, "mongodb://") && !strings.HasPrefix(dsn, "mongodb+srv://") {
		return fmt.Errorf("MongoDBClient: Invalid DSN format, expected 'mongodb://' or 'mongodb+srv://'")
	}
	// Extract the database name from the DSN
	databaseName, err := m.getDBNameFromMongoDSN(dsn)
	if err != nil {
		return fmt.Errorf("MongoDBClient: Failed to extract database name from datasource name(dsn): %v", err)
	}

	m.db = m.client.Database(databaseName)
	return nil
}

// Disconnect closes the connection to the MongoDB database.
// It checks if the client is not nil before attempting to disconnect.
func (m *MongoDBClient) Disconnect(ctx context.Context) error {
	fmt.Println("MongoDBClient: Disconnecting...")
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}

	return nil
}

// InsertOne inserts a single document into the specified collection in the MongoDB database.
// It takes a context, the name of the collection, and the document to be inserted.
// It returns the inserted ID and an error if the operation fails.
func (m *MongoDBClient) InsertOne(ctx context.Context, collectionName string, document interfaces.Document) (interface{}, error) {
	fmt.Printf("MongoDBClient: Inserting one into %s: %v\n", collectionName, document)

	// Validate the collection name
	// Ensure that the collection name is not empty
	if collectionName == "" {
		return nil, fmt.Errorf("MongoDBClient: Collection name cannot be empty")
	}
	res, err := m.db.Collection(collectionName).InsertOne(ctx, document)
	if err != nil {
		return nil, fmt.Errorf("MongoDBClient: Failed to insert one into %s: %v", collectionName, err)
	}

	return res.InsertedID, nil
}

// FindOne retrieves a single document from the specified collection in the MongoDB database.
// It takes a context, the name of the collection, a filter to match the document,
// and a result variable to decode the found document into.
// It returns an error if the operation fails or if no document is found.
func (m *MongoDBClient) FindOne(ctx context.Context, collectionName string, filter interfaces.Document, result interfaces.Document) error {
	fmt.Printf("MongoDBClient: Finding one in %s with filter: %v\n", collectionName, filter)

	err := m.db.Collection(collectionName).FindOne(ctx, filter).Decode(result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("MongoDBClient: No document found in %s with filter: %v", collectionName, filter)
		}
		return fmt.Errorf("MongoDBClient: Failed to find one in %s with filter: %v: %v", collectionName, filter, err)
	}

	return nil
}

// FindMany retrieves multiple documents from the specified collection in the MongoDB database.
// It takes a context, the name of the collection, and a filter to match the documents.
// It returns a slice of documents that match the filter and an error if the operation fails.
func (m *MongoDBClient) FindMany(ctx context.Context, collectionName string, filter interfaces.Document) ([]interfaces.Document, error) {
	fmt.Printf("MongoDBClient: Finding many in %s with filter: %v\n", collectionName, filter)
	cursor, err := m.db.Collection(collectionName).Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("MongoDBClient: Finding many in %s with filter: %v failed: %v", collectionName, filter, err)
	}

	defer func() {
		if err := cursor.Close(ctx); err != nil {
			fmt.Printf("MongoDBClient: Failed to close cursor: %v\n", err)
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

	return results, nil
}

// UpdateOne updates a single document in the specified collection in the MongoDB database.
// It takes a context, the name of the collection, a filter to match the document,
// and an update document that specifies the changes to be applied.
// It returns the count of modified documents and an error if the operation fails.
func (m *MongoDBClient) UpdateOne(ctx context.Context, collectionName string, filter interfaces.Document, update interfaces.Document) (int64, error) {
	fmt.Printf("MongoDBClient: Updating one in %s with filter %v, update %v\n", collectionName, filter, update)

	res, err := m.db.Collection(collectionName).UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, fmt.Errorf("MongoDBClient: Failed updating one in %s with filter %v, update %v: %v", collectionName, filter, update, err)
	}

	return res.ModifiedCount, nil
}

// DeleteOne deletes a single document from the specified collection in the MongoDB database.
// It takes a context, the name of the collection, and a filter to match the document to be deleted.
// It returns the count of deleted documents and an error if the operation fails.
func (m *MongoDBClient) DeleteOne(ctx context.Context, collectionName string, filter interfaces.Document) (int64, error) {
	fmt.Printf("MongoDBClient: Deleting one from %s with filter %v\n", collectionName, filter)

	res, err := m.db.Collection(collectionName).DeleteOne(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("MongoDBClient: Failed deleting one from %s with filter %v: %v", collectionName, filter, err)
	}

	return res.DeletedCount, nil
}

// DeleteMany deletes multiple documents from the specified collection in the MongoDB database.
// It takes a context, the name of the collection, and a filter to match the documents to be deleted.
// It returns the count of deleted documents and an error if the operation fails.
func (m *MongoDBClient) DeleteMany(ctx context.Context, collectionName string, filter interfaces.Document) (int64, error) {
	fmt.Printf("MongoDBClient: Deleting many from %s with filter %v\n", collectionName, filter)

	res, err := m.db.Collection(collectionName).DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("MongoDBClient: Failed Deleting many from %s with filter %v: %v", collectionName, filter, err)
	}

	return res.DeletedCount, nil
}

// Ping checks the health of the MongoDB connection by sending a ping command.
// It takes a context for cancellation and timeouts.
func (m *MongoDBClient) Ping(ctx context.Context) error {
	fmt.Println("MongoDBClient: Pinging...")
	return m.client.Ping(ctx,nil)
}

// getDBNameFromMongoDSN extracts the database name from a MongoDB DSN (Data Source Name).
// It parses the DSN URL and retrieves the database name from the path.
// The database name is expected to be the first segment of the path after the leading slash.
// If the DSN is invalid or the database name cannot be determined, it returns an error
func (m *MongoDBClient) getDBNameFromMongoDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to parse MongoDB DSN: %w", err)
	}

	// The database name is typically the first part of the path,
	// after the leading slash.
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return "", fmt.Errorf("no database name found in MongoDB DSN path: %s", dsn)
	}

	// If there are additional path segments or query parameters in the path
	// (uncommon for just the database name, but possible),
	// we only want the first segment if the format dictates it.
	// For simplicity, we assume the path is directly the database name.
	// For more complex paths like /db/collection, you'd need further logic.
	if idx := strings.Index(dbName, "/"); idx != -1 {
		dbName = dbName[:idx]
	}

	return dbName, nil
}

// EnsureSchema performs MongoDB-specific index creation (not part of generic DBClient)
// It ensures that the specified collection has the required index defined by the schema.
// The schema is expected to be a mongo.IndexModel, which defines the index to be created
// If the collection does not exist, it will be created automatically by MongoDB when the index is created.
// This method is used to ensure that the necessary indexes are in place for efficient querying.
func (m *MongoDBClient) EnsureSchema(ctx context.Context, collectionName string, schema interfaces.Document) error {
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
	return err
}
