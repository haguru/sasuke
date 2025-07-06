package app

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/metrics"
	"github.com/haguru/sasuke/internal/routes"
	"github.com/haguru/sasuke/internal/server"
	mongoUserRepo "github.com/haguru/sasuke/internal/userrepo/mongo"
	postgresUserRepo "github.com/haguru/sasuke/internal/userrepo/postgres"
	"github.com/haguru/sasuke/internal/userservice"
	"github.com/haguru/sasuke/pkg/databases/mongo"
	"github.com/haguru/sasuke/pkg/databases/postgres"

	structValidator "github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// App struct represents the main application structure.
// It contains the server instance and configuration settings.
// The server instance is used to handle HTTP requests and routes.
// The configuration settings are loaded from a local configuration file.
// The App struct is designed to be initialized with a configuration file path,
// which is read and validated during the initialization process.
// The App struct also includes methods for starting the server and handling routes.
type App struct {
	Server interfaces.Server // Placeholder for server interface
	Config *config.ServiceConfig
}

// NewApp initializes a new App instance with the provided configuration path.
// It reads the configuration from the specified path, validates it,
// and sets up the server and routes.
// If the configuration is invalid or the server fails to start,
// it returns an error.
// The function also initializes metrics and adds routes for handling HTTP requests.
func NewApp(configPath string) (*App, error) {
	cfg, err := config.ReadLocalConfig(configPath)
	if err != nil {
		return nil, err
	}

	app := &App{
		Config: cfg,
	}

	// Validate the configuration
	validator := structValidator.New()
	if err := validator.Struct(cfg); err != nil {
		// Validation failed, handle the error
		errors := err.(structValidator.ValidationErrors)
		return nil, fmt.Errorf("validation error: %s", errors)
	}
	// Initialize server, database, and metrics here if needed
	serverInstance := server.NewServer(cfg.Host, cfg.Port)
	app.Server = serverInstance

	// initialize metrics
	metricsInstance := metrics.NewMetrics(cfg.ServiceName)

	// check if cfg.PrivateKeyPath is provided
	if cfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("private key path is not provided in the configuration")
	}

	// Load or generate your ECDSA private key here
	var privateKey *ecdsa.PrivateKey
	privateKey, err = auth.LoadECDSAPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %v", err)
	}

	// initalize db client
	var dbClient interfaces.DBClient
	var userRepo interfaces.UserRepository
	switch cfg.Database.Type {
	case "mongo":
		// Initialize MongoDB client
		serverOptions := config.BuildServerAPIOptions(cfg.Database.MongoDB.Options)
		dbClient, err = mongo.NewMongoDB(cfg.Database.MongoDB.Timeout, serverOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MongoDB client: %v", err)
		}
		// Ensure the MongoDB client is connected
		dsn := fmt.Sprintf("mongodb://%s:%d/%s", cfg.Database.MongoDB.Host, cfg.Database.MongoDB.Port, cfg.Database.MongoDB.DatabaseName)
		if err = dbClient.Connect(context.Background(), dsn); err != nil {
			return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
		}

		// initialize MongoDB repository
		userRepo, err = mongoUserRepo.NewMongoUserRepository(dbClient)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MongoDB repository: %v", err)
		}

	case "postgres":
		// Initialize PostgreSQL client
		serverOptions := &cfg.Database.Postgres.Options

		// Create PostgreSQL database client
		dbClient = postgres.NewPostgresDatabaseClient(serverOptions.MaxOpenConns, serverOptions.MaxIdleConns, serverOptions.ConnMaxLifetime)

		// // Ensure the PostgreSQL client is connected
		// dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		// 	cfg.Database.Postgres.Host,
		// 	cfg.Database.Postgres.Port,
		// 	cfg.Database.Postgres.DatabaseName,
		// 	cfg.Database.Postgres.User,
		// 	cfg.Database.Postgres.Password,
		// )
		// initialize PostgreSQL repository
		userRepo, err = postgresUserRepo.NewPostgresUserRepository(dbClient)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL repository: %v", err)
		}

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	// Ensure indices for MongoDB or PostgreSQL
	if err = userRepo.EnsureIndices(context.Background()); err != nil { // Ensure indices are created
		return nil, fmt.Errorf("failed to ensure indices: %v", err)
	}

	// create user service
	userService := userservice.NewUserService(userRepo)

	// create for route
	route := routes.NewRoute(metricsInstance, userService, privateKey, validator)

	// metrics route
	metricsHandler := promhttp.Handler().ServeHTTP
	err = app.Server.AddRoute(routes.MetricsRouteAPI, metricsHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to add metrics route: %v", err)
	}

	// Add create route
	err = app.Server.AddRoute(routes.CreateRouteAPI, route.Create)
	if err != nil {
		return nil, fmt.Errorf("failed to add create route: %v", err)
	}

	// Add signup route
	err = app.Server.AddRoute(routes.SignupRouteAPI, route.Signup)
	if err != nil {
		return nil, fmt.Errorf("failed to add signup route: %v", err)
	}

	// Add login route
	err = app.Server.AddRoute(routes.LoginRouteAPI, route.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to add login route: %v", err)
	}

	return app, nil
}

func (app *App) Run() error {
	// start the server
	if err := app.Server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
