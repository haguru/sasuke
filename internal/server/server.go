package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/haguru/sasuke/internal/interfaces"
)

var (
	ReadTimeout  = 10 * time.Second
	WriteTimeout = 10 * time.Second
	IdleTimeout  = 30 * time.Second
)

type Server struct {
	Port   string
	Host   string
	server *http.Server
	mux    *http.ServeMux
}

// NewServer creates a new Server instance with the specified host and port.
func NewServer(host, port string) interfaces.Server {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:         host + ":" + port,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	return &Server{
		Host:   host,
		Port:   port,
		server: server,
		mux:    mux,
	}
}

// AddRoute adds a new route to the server.
// It takes a route string and a handler function as parameters.
// The handler function will be called when the route is accessed.
// It returns an error if the route cannot be added.
func (s *Server) AddRoute(route string, handler func(w http.ResponseWriter, r *http.Request)) error {
	s.mux.HandleFunc(route, handler)
	// Optionally, you can log the route addition
	// fmt.Printf("Route added: %s\n", route)
	return nil
}

// ListenAndServe starts the HTTP server and listens for incoming requests.
func (s *Server) ListenAndServe() error {
	// Start the HTTP server with the specified address
	err := s.server.ListenAndServe()
	if err != nil {
		// Log the error if the server fails to start
		// fmt.Printf("Failed to start server: %v\n", err)
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
