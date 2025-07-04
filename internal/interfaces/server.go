package interfaces

import(
	"net/http"
)

// Server interface defines the methods for a server implementation.
type Server interface {
	AddRoute(route string, handler func(w http.ResponseWriter, r *http.Request)) error	
	ListenAndServe() error
}