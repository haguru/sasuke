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
	MaxOpenConns    int             // MaxOpenConns is the maximum number of open connections to the database
	MaxIdleConns    int             // MaxIdleConns is the maximum number of idle connections to the database
	ConnMaxLifetime time.Duration   // ConnMaxLifetime is the maximum amount of time a connection may
	validColumns    map[string]bool // validColumns is a list of valid column names for sanitization
	validTables     map[string]bool // validTables is a list of valid table names for sanitization
	logger          interfaces.Logger
}

func NewPostgresDatabaseClient(dbConfig *config.PostgresConfig, logger interfaces.Logger) interfaces.DBClient {
	return &PostgresDatabaseClient{
		MaxOpenConns:    dbConfig.Options.MaxOpenConns,
		MaxIdleConns:    dbConfig.Options.MaxIdleConns,
		ConnMaxLifetime: dbConfig.Options.ConnMaxLifetime,
		validColumns:    config.ListToMap(dbConfig.ValidFields),
		validTables:     config.ListToMap(dbConfig.ValidTables),
		logger:          logger,
	}
}

// Connect establishes a connection to a PostgreSQL database.
func (p *PostgresDatabaseClient) Connect(ctx context.Context, dsn string) error {
	p.logger.Debug("Entering Connect", "dsn", dsn)
	var err error
	p.db, err = sql.Open("postgres", dsn)
	if err != nil {
		p.logger.Error("Failed to open PostgreSQL database", "error", err)
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	p.db.SetMaxOpenConns(p.MaxOpenConns)
	p.db.SetMaxIdleConns(p.MaxIdleConns)
	p.db.SetConnMaxLifetime(p.ConnMaxLifetime)
	p.logger.Info("PostgreSQL connection pool configured", "max_open_conns", p.MaxOpenConns, "max_idle_conns", p.MaxIdleConns, "conn_max_lifetime", p.ConnMaxLifetime)

	p.logger.Info("Pinging PostgreSQL database")
	return p.Ping(ctx)
}

// Disconnect closes the PostgreSQL database connection.
func (p *PostgresDatabaseClient) Disconnect(ctx context.Context) error {
	p.logger.Debug("Disconnecting from PostgreSQL database")
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// InsertOne inserts a document and returns its ID.
func (p *PostgresDatabaseClient) InsertOne(ctx context.Context, tableName string, document interfaces.Document) (interface{}, error) {
	p.logger.Debug("InsertOne called", "table", tableName, "document", document)
	docMap, ok := document.(map[string]interface{})
	if !ok {
		p.logger.Error("InsertOne expects document to be map[string]interface{}", "table", tableName)
		return nil, fmt.Errorf("PostgreSQL InsertOne expects document to be map[string]interface{}")
	}

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

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	) // #nosec G201
	p.logger.Info("Executing InsertOne query", "query", query, "values", values)

	var insertedID interface{}
	err := p.db.QueryRowContext(ctx, query, values...).Scan(&insertedID)
	if err != nil {
		p.logger.Error("InsertOne query failed", "error", err)
		return nil, err
	}
	p.logger.Info("InsertOne succeeded", "id", insertedID)
	return insertedID, nil
}

// FindOne retrieves a single document matching the filter.
func (p *PostgresDatabaseClient) FindOne(ctx context.Context, tableName string, filter interfaces.Document, result interfaces.Document) error {
	p.logger.Debug("FindOne called", "table", tableName, "filter", filter)
	if !p.validTables[tableName] {
		p.logger.Error("Invalid table name in FindOne", "table", tableName)
		return fmt.Errorf("invalid table name: %s", tableName)
	}

	sanitizedFilterMap, err := p.sanitizeDocument(filter)
	if err != nil {
		p.logger.Error("Failed to sanitize filter in FindOne", "error", err)
		return fmt.Errorf("PostgreSQL FindOne failed to sanitize filter: %w", err)
	}

	if len(sanitizedFilterMap) == 0 {
		p.logger.Error("FindOne requires a non-empty filter", "table", tableName)
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

	resultValue := reflect.ValueOf(result)
	if resultValue.Kind() != reflect.Ptr || resultValue.Elem().Kind() != reflect.Struct {
		p.logger.Error("Result must be a pointer to a struct in FindOne")
		return fmt.Errorf("result must be a pointer to a struct")
	}
	elem := resultValue.Elem()
	numFields := elem.NumField()

	columns := make([]string, numFields)
	fieldPointers := make([]any, numFields)

	for i := range columns {
		field := elem.Type().Field(i)
		columns[i] = strings.ToLower(field.Name)
		fieldPointers[i] = elem.Field(i).Addr().Interface()
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIMIT 1",
		strings.Join(columns, ", "),
		tableName,
		whereString,
	) // #nosec G201
	p.logger.Info("Executing FindOne query", "query", query, "values", whereValues)

	row := p.db.QueryRowContext(ctx, query, whereValues...)
	err = row.Scan(fieldPointers...)
	if err == sql.ErrNoRows {
		p.logger.Info("FindOne: no rows found", "table", tableName, "filter", filter)
		reflect.New(elem.Type()).Elem().Set(elem)
		return nil
	}
	if err != nil {
		p.logger.Error("FindOne query failed", "error", err)
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
