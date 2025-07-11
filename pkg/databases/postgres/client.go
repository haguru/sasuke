package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/interfaces"

	"github.com/google/uuid"
)

const (
	// DefaultMaxOpenConns is the default maximum number of open connections to the database.
	DefaultMaxOpenConns = 10
	// DefaultMaxIdleConns is the default maximum number of idle connections to the database.
	DefaultMaxIdleConns = 5
	// DefaultConnMaxLifetime is the default maximum amount of time a connection may be reused.
	DefaultConnMaxLifetime = 30 * time.Second

	// IDFIELD is the name of the ID field in PostgreSQL documents.
	IDFIELD = "id"
)

type PostgresDatabaseClient struct {
	db              *sql.DB
	Host            string          // Host is the PostgreSQL server host
	Port            int             // Port is the PostgreSQL server port
	MaxOpenConns    int             // MaxOpenConns is the maximum number of open connections to the database
	MaxIdleConns    int             // MaxIdleConns is the maximum number of idle connections to the database
	ConnMaxLifetime time.Duration   // ConnMaxLifetime is the maximum amount of time a connection may
	validColumns    map[string]bool // validColumns is a list of valid column names for sanitization
	validTables     map[string]bool // validTables is a list of valid table names for sanitization
}

func NewPostgresDatabaseClient(dbConfig *config.PostgresConfig) interfaces.DBClient {
	return &PostgresDatabaseClient{
		MaxOpenConns:    dbConfig.Options.MaxOpenConns,
		MaxIdleConns:    dbConfig.Options.MaxIdleConns,
		ConnMaxLifetime: dbConfig.Options.ConnMaxLifetime,
		validColumns:    config.ListToMap(dbConfig.ValidFields),
		validTables:     config.ListToMap(dbConfig.ValidTables),
	}
}

// Connect establishes a connection to a PostgreSQL database.
func (p *PostgresDatabaseClient) Connect(ctx context.Context, dsn string) error {
	var err error
	p.db, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	p.db.SetMaxOpenConns(p.MaxOpenConns)
	p.db.SetMaxIdleConns(p.MaxIdleConns)
	p.db.SetConnMaxLifetime(p.ConnMaxLifetime)

	return p.Ping(ctx)
}

// Disconnect closes the PostgreSQL database connection.
func (p *PostgresDatabaseClient) Disconnect(ctx context.Context) error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// InsertOne inserts a document and returns its ID.
func (p *PostgresDatabaseClient) InsertOne(ctx context.Context, tableName string, document interfaces.Document) (interface{}, error) {
	docMap, ok := document.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("PostgreSQL InsertOne expects document to be map[string]interface{}")
	}

	// Generate UUID for 'id' if not present in the document
	if _, exists := docMap["id"]; !exists {
		docMap["id"] = uuid.New().String()
	}

	columns := make([]string, 0, len(docMap))
	placeholders := make([]string, 0, len(docMap))
	values := make([]interface{}, 0, len(docMap))

	i := 1
	for col, val := range docMap {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	// This is a safe use of fmt.Sprintf for SQL query construction, as the table name is controlled and not user input.
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	) // #nosec G201

	var insertedID interface{} // Can be string (UUID), int, etc.
	err := p.db.QueryRowContext(ctx, query, values...).Scan(&insertedID)
	if err != nil {
		return nil, err
	}
	return insertedID, nil
}

// FindOne retrieves a single document matching the filter.
func (p *PostgresDatabaseClient) FindOne(ctx context.Context, tableName string, filter interfaces.Document, result interfaces.Document) error {
	if !p.validTables[tableName] {
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	// sanitize filterMap
	sanitizedFilterMap, err := p.sanitizeDocument(filter)
	if err != nil {
		return fmt.Errorf("PostgreSQL FindOne failed to sanitize filter: %w", err)
	}

	if len(sanitizedFilterMap) == 0 {
		return fmt.Errorf("PostgreSQL FindOne requires a non-empty filter")
	}

	whereClauses := make([]string, 0, len(sanitizedFilterMap))
	whereValues := make([]any, 0, len(sanitizedFilterMap))
	paramCount := 1
	for col, val := range sanitizedFilterMap {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, paramCount))
		whereValues = append(whereValues, val)
		paramCount++
	}
	whereString := strings.Join(whereClauses, " AND ")

	// Use reflection to get fields from the 'result' struct for SELECT and Scan
	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("result must be a pointer to a struct")
	}
	elem := resultValue.Elem()
	numFields := elem.NumField()

	columns := make([]string, numFields)
	fieldPointers := make([]any, numFields) // Pointers to fields in the struct for Scan()

	for i := range columns {
		field := elem.Type().Field(i)
		columns[i] = strings.ToLower(field.Name) // Convert field name to snake_case or whatever your DB uses
		fieldPointers[i] = elem.Field(i).Addr().Interface()
	}

	// This is a safe use of fmt.Sprintf for SQL query construction, as the table name is controlled and not user input.
	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIMIT 1",
		strings.Join(columns, ", "),
		tableName,
		whereString,
	) // #nosec G201

	row := p.db.QueryRowContext(ctx, query, whereValues...)
	err = row.Scan(fieldPointers...)
	if err == sql.ErrNoRows {
		// Reset the struct if no rows found, so it doesn't contain partial data
		reflect.New(elem.Type()).Elem().Set(elem)
		return nil // Return nil error as per DBClient interface if no document is found
	}
	return err
}

// FindMany returns multiple documents from a PostgreSQL table matching the filter.
func (p *PostgresDatabaseClient) FindMany(ctx context.Context, tableName string, filter interfaces.Document) ([]interfaces.Document, error) {
	if !p.validTables[tableName] {
		return nil, fmt.Errorf("invalid table name: %s", tableName)
	}

	// sanitize filterMap
	sanitizedFilterMap, err := p.sanitizeDocument(filter)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL FindMany failed to sanitize filter: %w", err)
	}

	whereClauses := make([]string, 0, len(sanitizedFilterMap))
	whereValues := make([]interface{}, 0, len(sanitizedFilterMap))
	paramCount := 1
	for col, val := range sanitizedFilterMap {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, paramCount))
		whereValues = append(whereValues, val)
		paramCount++
	}
	whereString := ""
	if len(whereClauses) > 0 {
		whereString = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Query selects all columns. For specific columns, add an argument.
	query := fmt.Sprintf("SELECT * FROM %s%s", tableName, whereString) // #nosec G201

	rows, err := p.db.QueryContext(ctx, query, whereValues...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			fmt.Printf("failed to close rows: %v", cerr)
		}
	}()

	var results []interfaces.Document
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		columnValues := make([]interface{}, len(columns))
		for i := range columns {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range columns {
			val := columnValues[i]
			if b, ok := val.([]byte); ok { // Handle byte slices for string-like types
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// UpdateOne updates a single row in a PostgreSQL table matching the filter.
func (p *PostgresDatabaseClient) UpdateOne(ctx context.Context, tableName string, filter interfaces.Document, update interfaces.Document) (int64, error) {
	if !p.validTables[tableName] {
		return 0, fmt.Errorf("invalid table name: %s", tableName)
	}

	// sanitize filterMap
	sanitizedFilterMap, err := p.sanitizeDocument(filter)
	if err != nil {
		return 0, fmt.Errorf("PostgreSQL FindMany failed to sanitize filter: %w", err)
	}

	// sanitize updateMap
	sanitizedUpdateMap, err := p.sanitizeDocument(update)
	if err != nil {
		return 0, fmt.Errorf("PostgreSQL UpdateOne failed to sanitize update: %w", err)
	}

	setClauses := make([]string, 0, len(sanitizedUpdateMap))
	whereClauses := make([]string, 0, len(sanitizedFilterMap))
	values := make([]interface{}, 0, len(sanitizedUpdateMap)+len(sanitizedFilterMap))
	paramCount := 1

	for col, val := range sanitizedUpdateMap {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, paramCount))
		values = append(values, val)
		paramCount++
	}

	for col, val := range sanitizedFilterMap {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, paramCount))
		values = append(values, val)
		paramCount++
	}

	// Table name is validated; safe for fmt.Sprintf.
	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "),
	) // #nosec G201

	res, err := p.db.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// DeleteOne deletes a single row from a PostgreSQL table matching the filter.
func (p *PostgresDatabaseClient) DeleteOne(ctx context.Context, tableName string, filter interfaces.Document) (int64, error) {
	if !p.validTables[tableName] {
		return 0, fmt.Errorf("invalid table name: %s", tableName)
	}

	// sanitize filterMap
	sanitizedFilterMap, err := p.sanitizeDocument(filter)
	if err != nil {
		return 0, fmt.Errorf("PostgreSQL FindMany failed to sanitize filter: %w", err)
	}

	whereClauses := make([]string, 0, len(sanitizedFilterMap))
	whereValues := make([]interface{}, 0, len(sanitizedFilterMap))
	paramCount := 1
	for col, val := range sanitizedFilterMap {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, paramCount))
		whereValues = append(whereValues, val)
		paramCount++
	}

	// Table name is validated; safe for fmt.Sprintf.
	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		tableName,
		strings.Join(whereClauses, " AND "),
	) // #nosec G201

	res, err := p.db.ExecContext(ctx, query, whereValues...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// DeleteMany deletes multiple rows from a PostgreSQL table matching the filter.
func (p *PostgresDatabaseClient) DeleteMany(ctx context.Context, tableName string, filter interfaces.Document) (int64, error) {
	if !p.validTables[tableName] {
		return 0, fmt.Errorf("invalid table name: %s", tableName)
	}

	// sanitize filterMap
	sanitizedFilterMap, err := p.sanitizeDocument(filter)
	if err != nil {
		return 0, fmt.Errorf("PostgreSQL FindMany failed to sanitize filter: %w", err)
	}

	whereClauses := make([]string, 0, len(sanitizedFilterMap))
	whereValues := make([]interface{}, 0, len(sanitizedFilterMap))
	paramCount := 1
	for col, val := range sanitizedFilterMap {
		whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, paramCount))
		whereValues = append(whereValues, val)
		paramCount++
	}

	whereString := ""
	if len(whereClauses) > 0 {
		whereString = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Table name is validated; safe for fmt.Sprintf.
	query := fmt.Sprintf("DELETE FROM %s%s RETURNING id", tableName, whereString) // #nosec G201

	res, err := p.db.ExecContext(ctx, query, whereValues...)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}

// Ping checks the health of the PostgreSQL connection.
func (p *PostgresDatabaseClient) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// EnsureSchema creates the table and indices if needed.
func (p *PostgresDatabaseClient) EnsureSchema(ctx context.Context, tableName string, schema interfaces.Document) error {
	if p.db == nil {
		return fmt.Errorf("PostgresDatabaseClient is not connected to a database")
	}

	// Ensure schema is provided as a CREATE TABLE statement string
	schemaStr, ok := schema.(string)
	if !ok || !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(schemaStr)), "CREATE TABLE") {
		return fmt.Errorf("EnsureSchema expects schema to be a CREATE TABLE statement string")
	}
	_, err := p.db.ExecContext(ctx, schemaStr)
	return err
}

// SanitizeDocument removes the ID field and invalid keys to prevent SQL injection.
func (p *PostgresDatabaseClient) sanitizeDocument(document interfaces.Document) (map[string]interface{}, error) {
	if document == nil {
		return nil, fmt.Errorf("PostgreSQL SanitizeDocument: Document is nil")
	}

	docMap, ok := document.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("PostgreSQL SanitizeDocument expects document to be map[string]interface{}")
	}

	delete(docMap, IDFIELD)

	// Sanitize keys to prevent SQL injection and check for valid columns
	for key := range docMap {
		if strings.ContainsAny(key, "();--") || !p.validColumns[key] {
			fmt.Printf("PostgreSQL SanitizeDocument: Detected invalid or malicious key: %s\n", key)
			delete(docMap, key)
		}
	}

	return docMap, nil
}
