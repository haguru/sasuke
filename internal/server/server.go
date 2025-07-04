package server

import (
	"fmt"
	"net/http"

	"github.com/haguru/sasuke/internal/interfaces"
)

type Server struct {
	Port string
	Host string
}

// NewServer creates a new Server instance with the specified host and port.
func NewServer(host, port string) interfaces.Server {
	return &Server{
		Host: host,
		Port: port,
	}
}

// AddRoute adds a new route to the server.
// It takes a route string and a handler function as parameters.
// The handler function will be called when the route is accessed.
// It returns an error if the route cannot be added.
func (s *Server) AddRoute(route string, handler func(w http.ResponseWriter, r *http.Request)) error {
	http.HandleFunc(route, handler)
	// Optionally, you can log the route addition
	// fmt.Printf("Route added: %s\n", route)
	return nil
}

// ListenAndServe starts the HTTP server and listens for incoming requests.
func (s *Server) ListenAndServe() error {
	addr := s.Host + ":" + s.Port

	// Start the HTTP server with the specified address
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		// Log the error if the server fails to start
		// fmt.Printf("Failed to start server: %v\n", err)
		return fmt.Errorf("failed to start server: %v", err)
	}
	// Log the successful start of the server
	// fmt.Printf("Server is listening on %s\n", addr)
	return nil
}
