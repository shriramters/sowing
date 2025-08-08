package web

import (
	"database/sql"
	"html/template"
	"net/http"

	"sowing/internal/auth"
)

// Server holds the dependencies for the web server.
type Server struct {
	db          *sql.DB
	templates   map[string]*template.Template
	authService *auth.Service
}

// NewServer creates a new server with the given dependencies.
func NewServer(db *sql.DB, templates map[string]*template.Template) *Server {
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo)
	return &Server{
		db:          db,
		templates:   templates,
		authService: authService,
	}
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.routes().ServeHTTP(w, r)
}
