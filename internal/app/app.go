package app

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/haguru/sasuke/config"
	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/metrics"
	"github.com/haguru/sasuke/internal/routes"
	"github.com/haguru/sasuke/internal/server"

	"github.com/go-playground/validator/v10"
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
	Server interface{} // Placeholder for server interface
	Config config.ServiceConfig
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
		Config: *cfg,
	}

	// Validate the configuration
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		// Validation failed, handle the error
		errors := err.(validator.ValidationErrors)
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

	// Initialize user repository
	
	// userRepo := userrepository.NewUserRepository(cfg.DatabaseURL) // Assuming you have a user repository implementation
	
	// initialize user service
	// userService := userservice.NewUserService(userRepo) // Assuming you have a user

	// create for route
	route := routes.NewRoute(metricsInstance, privateKey)

	// metrics route
	metricsHandler := promhttp.Handler().ServeHTTP
	err = serverInstance.AddRoute(routes.MetricsRouteAPI, metricsHandler)
	if err != nil {
		return nil, fmt.Errorf("failed to add metrics route: %v", err)
	}

	// Add routes to the server
	err = serverInstance.AddRoute(routes.CreateRouteAPI, route.Create)
	if err != nil {
		return nil, fmt.Errorf("failed to add create route: %v", err)
	}

	// start the server
	if err := serverInstance.ListenAndServe(); err != nil {
		return nil, fmt.Errorf("failed to start server: %v", err)
	}

	return app, nil
}
