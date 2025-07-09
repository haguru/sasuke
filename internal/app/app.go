package app

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"

	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/middleware"
	"github.com/haguru/sasuke/internal/routes"
	"github.com/haguru/sasuke/internal/server"
	mongoUserRepo "github.com/haguru/sasuke/internal/userrepo/mongo"
	postgresUserRepo "github.com/haguru/sasuke/internal/userrepo/postgres"
	"github.com/haguru/sasuke/internal/userservice"
	"github.com/haguru/sasuke/pkg/databases/mongo"
	"github.com/haguru/sasuke/pkg/databases/postgres"
	"github.com/haguru/sasuke/pkg/metrics"
	"github.com/haguru/sasuke/pkg/zerolog"

	structValidator "github.com/go-playground/validator/v10"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/time/rate"
)

// App represents the main application, containing server and configuration.
// It initializes with a config file, validates settings, and manages routes.
type App struct {
	Server     interfaces.Server
	Config     *config.ServiceConfig
	privateKey *ecdsa.PrivateKey
	logger     interfaces.Logger
}

// NewApp creates and configures a new App instance.
func NewApp(configPath string) (*App, error) {
	cfg, err := config.ReadLocalConfig(configPath)
	if err != nil {
		fmt.Printf("Failed to read config: %v\n", err)
		return nil, err
	}

	app := &App{
		Config: cfg,
		logger: zerolog.NewZerologLogger(cfg.ServiceName),
	}

	app.logger.SetLevel(cfg.LogLevel)
	app.logger.Info("Starting service", "service_name", cfg.ServiceName)

	// Validate the configuration
	validator := structValidator.New()
	if err := validator.Struct(cfg); err != nil {
		errors := err.(structValidator.ValidationErrors)
		app.logger.Error("Configuration validation failed", "errors", errors.Error())
		return nil, fmt.Errorf("validation error: %s", errors)
	}
	app.logger.Info("Configuration validated successfully")

	// Initialize server, database, and metrics
	serverInstance := server.NewServer(cfg.Host, cfg.Port, app.logger)
	app.Server = serverInstance
	app.logger.Info("Server initialized", "host", cfg.Host, "port", cfg.Port)

	metricsInstance := app.initializeMetrics()
	app.logger.Info("Metrics initialized")

	if err := app.initializePrivateKey(); err != nil {
		app.logger.Error("Failed to initialize private key", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize private key: %v", err)
	}
	app.logger.Info("Private key initialized")

	dbClient, err := app.initializeDBClient()
	if err != nil {
		app.logger.Error("Failed to initialize database client", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize database client: %v", err)
	}
	app.logger.Info("Database client initialized", "db_type", cfg.Database.Type)

	userRepo, err := app.initializeUserRepo(dbClient)
	if err != nil {
		app.logger.Error("Failed to initialize user repository", "error", err.Error())
		return nil, fmt.Errorf("failed to initialize user repository: %v", err)
	}
	app.logger.Info("User repository initialized", "db_type", cfg.Database.Type)

	userService := userservice.NewUserService(userRepo, app.logger)
	app.logger.Info("User service initialized")

	route := routes.NewRoute(
		metricsInstance,
		userService,
		app.privateKey,
		validator,
		app.logger,
	)

	metricsHandler := promhttp.HandlerFor(
		metricsInstance.GetRegistry(),
		promhttp.HandlerOpts{})

	tracedMetricsHandler := otelhttp.NewHandler(metricsHandler, routes.MetricsRouteAPI)

	err = app.Server.AddRoute(routes.MetricsRouteAPI, tracedMetricsHandler.ServeHTTP)
	if err != nil {
		app.logger.Error("Failed to add metrics route", "error", err.Error())
		return nil, fmt.Errorf("failed to add metrics route: %v", err)
	}
	app.logger.Info("Metrics route added", "route", routes.MetricsRouteAPI)

	err = app.Server.AddRoute(routes.CreateRouteAPI, route.Create)
	if err != nil {
		app.logger.Error("Failed to add create route", "error", err.Error())
		return nil, fmt.Errorf("failed to add create route: %v", err)
	}
	app.logger.Info("Create route added", "route", routes.CreateRouteAPI)

	err = app.Server.AddRoute(routes.SignupRouteAPI, route.Signup)
	if err != nil {
		app.logger.Error("Failed to add signup route", "error", err.Error())
		return nil, fmt.Errorf("failed to add signup route: %v", err)
	}
	app.logger.Info("Signup route added", "route", routes.SignupRouteAPI)

	loginLimiter := rate.NewLimiter(rate.Every(cfg.RateLimiter.Interval), cfg.RateLimiter.Limit)
	app.logger.Info("Login rate limiter initialized", "interval", cfg.RateLimiter.Interval, "limit", cfg.RateLimiter.Limit)

	// Wrap the login handler with rate limiting middleware.
	rateLimiter := middleware.RateLimitMiddleware(loginLimiter, app.logger)
	loginHandler := rateLimiter(http.HandlerFunc(route.Login))

	err = app.Server.AddRoute(routes.LoginRouteAPI, loginHandler.ServeHTTP)
	if err != nil {
		app.logger.Error("Failed to add login route", "error", err.Error())
		return nil, fmt.Errorf("failed to add login route: %v", err)
	}
	app.logger.Info("Login route added", "route", routes.LoginRouteAPI)

	return app, nil
}

func (app *App) Run() error {
	app.logger.Info("Starting server")
	if err := app.Server.ListenAndServe(); err != nil {
		app.logger.Error("Failed to start server", "error", err.Error())
		return fmt.Errorf("failed to start server: %v", err)
	}
	app.logger.Info("Server stopped gracefully")
	return nil
}

func (app *App) initializeMetrics() interfaces.Metrics {
	app.logger.Info("Initializing metrics")
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

	app.logger.Info("Metrics counters and histograms registered")
	return appMetrics
}

func (app *App) initializeDBClient() (interfaces.DBClient, error) {
	app.logger.Info("Initializing database client", "db_type", app.Config.Database.Type)
	var dbClient interfaces.DBClient
	var err error

	switch app.Config.Database.Type {
	case "mongo":
		dbClient, err = mongo.NewMongoDB(&app.Config.Database.MongoDB, app.logger)
		if err != nil {
			app.logger.Error("Failed to initialize MongoDB client", "error", err.Error())
			return nil, fmt.Errorf("failed to initialize MongoDB client: %v", err)
		}

		if err = dbClient.Connect(context.Background(), app.Config.Database.MongoDB.DSN); err != nil {
			app.logger.Error("Failed to connect to MongoDB", "error", err.Error())
			return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
		}
		app.logger.Info("MongoDB client created and connected")

	case "postgres":
		dbClient = postgres.NewPostgresDatabaseClient(&app.Config.Database.Postgres)
		app.logger.Info("Postgres client created")

	default:
		app.logger.Error("Unsupported database type", "db_type", app.Config.Database.Type)
		return nil, fmt.Errorf("unsupported database type: %s", app.Config.Database.Type)
	}

	return dbClient, nil
}

func (app *App) initializeUserRepo(dbClient interfaces.DBClient) (interfaces.UserRepository, error) {
	app.logger.Info("Initializing user repository", "db_type", app.Config.Database.Type)
	var userRepo interfaces.UserRepository
	var err error

	switch app.Config.Database.Type {
	case "mongo":
		userRepo, err = mongoUserRepo.NewMongoUserRepository(dbClient)
		if err != nil {
			app.logger.Error("Failed to initialize MongoDB repository", "error", err.Error())
			return nil, fmt.Errorf("failed to initialize MongoDB repository: %v", err)
		}
		app.logger.Info("MongoDB user repository initialized")

	case "postgres":
		userRepo, err = postgresUserRepo.NewPostgresUserRepository(dbClient)
		if err != nil {
			app.logger.Error("Failed to initialize PostgreSQL repository", "error", err.Error())
			return nil, fmt.Errorf("failed to initialize PostgreSQL repository: %v", err)
		}
		app.logger.Info("PostgreSQL user repository initialized")

	default:
		app.logger.Error("Unsupported database type for user repository", "db_type", app.Config.Database.Type)
		return nil, fmt.Errorf("unsupported database type: %s", app.Config.Database.Type)
	}

	if err = userRepo.EnsureIndices(context.Background()); err != nil {
		app.logger.Error("Failed to ensure indices", "error", err.Error())
		return nil, fmt.Errorf("failed to ensure indices: %v", err)
	}
	app.logger.Info("User repository indices ensured")

	return userRepo, nil
}

func (app *App) initializePrivateKey() error {
	app.logger.Info("Initializing private key", "path", app.Config.PrivateKeyPath)
	if app.Config.PrivateKeyPath == "" {
		app.logger.Error("Private key path not provided in configuration")
		return fmt.Errorf("private key path is not provided in the configuration")
	}

	privateKey, err := auth.LoadECDSAPrivateKey(app.Config.PrivateKeyPath)
	if err != nil {
		app.logger.Error("Failed to load private key", "error", err.Error())
		return fmt.Errorf("failed to load private key: %v", err)
	}

	app.privateKey = privateKey
	app.logger.Info("Private key loaded successfully")
	return nil
}
