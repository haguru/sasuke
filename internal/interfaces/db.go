package interfaces


import "context"

// Document is a generic interface to represent data that can be stored
// and retrieved from the database. It could be a struct, a map[string]interface{},
// or any type that can be marshaled/unmarshaled by the specific database driver.
type Document interface{}

// DBClient defines the interface for a generic database client.
// It abstracts common database operations across different database types (e.g., MongoDB, SQL).
type DBClient interface {
	// Connect establishes a connection to the database.
	// It takes a context for cancellation and timeouts, and a DSN (Data Source Name) string.
	// Returns an error if the connection fails.
	Connect(ctx context.Context, dsn string) error

	// Disconnect closes the database connection.
	// Returns an error if the disconnection fails.
	Disconnect(ctx context.Context) error

	// InsertOne inserts a single document into the specified collection/table.
	// The collection/table name is provided by 'collectionName'.
	// 'document' is the data to be inserted.
	// Returns the ID of the inserted document (e.g., MongoDB ObjectID, SQL primary key) and an error.
	InsertOne(ctx context.Context, collectionName string, document Document) (interface{}, error)

	// FindOne retrieves a single document from the specified collection/table
	// that matches the provided filter.
	// 'filter' is a mechanism to specify query conditions (e.g., MongoDB BSON D, SQL WHERE clause).
	// 'result' is a pointer to the variable where the decoded document will be stored.
	// Returns an error if no document is found or an issue occurs during retrieval.
	FindOne(ctx context.Context, collectionName string, filter Document, result Document) error

	// FindMany retrieves multiple documents from the specified collection/table
	// that match the provided filter.
	// 'filter' is a mechanism to specify query conditions.
	// Returns a slice of documents and an error.
	FindMany(ctx context.Context, collectionName string, filter Document) ([]Document, error)

	// UpdateOne updates a single document in the specified collection/table
	// that matches the provided filter with the given update data.
	// 'update' specifies the changes to be applied.
	// Returns the count of modified documents and an error.
	UpdateOne(ctx context.Context, collectionName string, filter Document, update Document) (int64, error)

	// DeleteOne deletes a single document from the specified collection/table
	// that matches the provided filter.
	// Returns the count of deleted documents and an error.
	DeleteOne(ctx context.Context, collectionName string, filter Document) (int64, error)

	// DeleteMany deletes multiple documents from the specified collection/table
	// that match the provided filter.
	// Returns the count of deleted documents and an error.
	DeleteMany(ctx context.Context, collectionName string, filter Document) (int64, error)

	// Ping checks the health of the database connection.
	// Returns an error if the database is unreachable or unhealthy.
	Ping(ctx context.Context) error
}