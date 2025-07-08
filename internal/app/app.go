package app

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/routes"
	"github.com/haguru/sasuke/internal/server"
	mongoUserRepo "github.com/haguru/sasuke/internal/userrepo/mongo"
	postgresUserRepo "github.com/haguru/sasuke/internal/userrepo/postgres"
	"github.com/haguru/sasuke/internal/userservice"
	"github.com/haguru/sasuke/pkg/databases/mongo"
	"github.com/haguru/sasuke/pkg/databases/postgres"
	"github.com/haguru/sasuke/pkg/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	structValidator "github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// App represents the main application, containing server and configuration.
// It initializes with a config file, validates settings, and manages routes.
type App struct {
	Server     interfaces.Server
	Config     *config.ServiceConfig
	privateKey *ecdsa.PrivateKey
}

// NewApp creates and configures a new App instance.
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

	// Initialize server, database, and metrics
	serverInstance := server.NewServer(cfg.Host, cfg.Port)
	app.Server = serverInstance

	metricsInstance := app.initializeMetrics()

	if err := app.initializePrivateKey(); err != nil {
		return nil, fmt.Errorf("failed to initialize private key: %v", err)
	}

	dbClient, err := app.initializeDBClient()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database client: %v", err)
	}

	userRepo, err := app.initializeUserRepo(dbClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user repository: %v", err)
	}

	userService := userservice.NewUserService(userRepo)

	route := routes.NewRoute(metricsInstance, userService, app.privateKey, validator)

	metricsHandler := promhttp.HandlerFor(
		metricsInstance.GetRegistry(),
		promhttp.HandlerOpts{})

	tracedMetricsHandler := otelhttp.NewHandler(metricsHandler, routes.MetricsRouteAPI)

	err = app.Server.AddRoute(routes.MetricsRouteAPI, tracedMetricsHandler.ServeHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to add metrics route: %v", err)
	}

	err = app.Server.AddRoute(routes.CreateRouteAPI, route.Create)
	if err != nil {
		return nil, fmt.Errorf("failed to add create route: %v", err)
	}
	fmt.Println("Create route added successfully")

	err = app.Server.AddRoute(routes.SignupRouteAPI, route.Signup)
	if err != nil {
		return nil, fmt.Errorf("failed to add signup route: %v", err)
	}
	fmt.Println("Signup route added successfully")

	err = app.Server.AddRoute(routes.LoginRouteAPI, route.Login)
	if err != nil {
		return nil, fmt.Errorf("failed to add login route: %v", err)
	}
	fmt.Println("Login route added successfully")

	return app, nil
}

func (app *App) Run() error {
	// start the server
	if err := app.Server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}

func (app *App) initializeMetrics() interfaces.Metrics {
	appMetrics := metrics.NewMetrics(app.Config.ServiceName)
	appMetrics.RegisterCounter(routes.SignupRequestsTotal, routes.SignupRequestsTotalHelp)
	appMetrics.RegisterCounter(routes.SignupSuccessTotal, routes.SignupSuccessTotalHelp)
	appMetrics.RegisterCounter(routes.SignupErrorsTotal, routes.SignupErrorsTotalHelp)
	appMetrics.RegisterHistogram(
		routes.SignupDurationSeconds,
		routes.SignupDurationSecondsHelp,
		routes.SignupDurationSecondsBuckets)

	appMetrics.RegisterCounter(routes.LoginRequestsTotal, routes.LoginRequestsTotalHelp)
	appMetrics.RegisterCounter(routes.LoginSuccessTotal, routes.LoginSuccessTotalHelp)
	appMetrics.RegisterCounter(routes.LoginFailedTotal, routes.LoginFailedTotalHelp)
	appMetrics.RegisterHistogram(
		routes.LoginDurationSeconds,
		routes.LoginDurationSecondsHelp,
		routes.LoginDurationSecondsBuckets)

	return appMetrics
}

func (app *App) initializeDBClient() (interfaces.DBClient, error) {
	var dbClient interfaces.DBClient
	var err error

	switch app.Config.Database.Type {
	case "mongo":
		// Initialize MongoDB client
		dbClient, err = mongo.NewMongoDB(&app.Config.Database.MongoDB)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MongoDB client: %v", err)
		}

		// Ensure the MongoDB client is connected
		if err = dbClient.Connect(context.Background(), app.Config.Database.MongoDB.DSN); err != nil {
			return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
		}

	case "postgres":
		// Create PostgreSQL database client
		dbClient = postgres.NewPostgresDatabaseClient(&app.Config.Database.Postgres)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", app.Config.Database.Type)
	}

	return dbClient, nil
}

func (app *App) initializeUserRepo(dbClient interfaces.DBClient) (interfaces.UserRepository, error) {
	var userRepo interfaces.UserRepository
	var err error

	switch app.Config.Database.Type {
	case "mongo":
		// Initialize MongoDB repository
		userRepo, err = mongoUserRepo.NewMongoUserRepository(dbClient)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize MongoDB repository: %v", err)
		}

	case "postgres":
		// Initialize PostgreSQL repository
		userRepo, err = postgresUserRepo.NewPostgresUserRepository(dbClient)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize PostgreSQL repository: %v", err)
		}

	default:
		return nil, fmt.Errorf("unsupported database type: %s", app.Config.Database.Type)
	}

	// Ensure indices for MongoDB or PostgreSQL
	if err = userRepo.EnsureIndices(context.Background()); err != nil { // Ensure indices are created
		return nil, fmt.Errorf("failed to ensure indices: %v", err)
	}

	return userRepo, nil
}

func (app *App) initializePrivateKey() error {
	if app.Config.PrivateKeyPath == "" {
		return fmt.Errorf("private key path is not provided in the configuration")
	}

	privateKey, err := auth.LoadECDSAPrivateKey(app.Config.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}

	app.privateKey = privateKey
	return nil
}
