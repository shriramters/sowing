package web

import (
	"database/sql"
	"html/template"
	"net/http"

	"sowing/internal/attachment"
	"sowing/internal/auth"
	"sowing/internal/page"
	"sowing/internal/silo"
)

// Server holds the dependencies for the web server.
type Server struct {
	db             *sql.DB
	templates      map[string]*template.Template
	authService    *auth.Service
	attachmentRepo *attachment.Repository
	pageRepo       *page.Repository
	siloRepo       *silo.Repository
}

// NewServer creates a new server with the given dependencies.
func NewServer(db *sql.DB, templates map[string]*template.Template) *Server {
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo)
	attachmentRepo := attachment.NewRepository(db)
	pageRepo := page.NewRepository(db)
	siloRepo := silo.NewRepository(db)

	return &Server{
		db:             db,
		templates:      templates,
		authService:    authService,
		attachmentRepo: attachmentRepo,
		pageRepo:       pageRepo,
		siloRepo:       siloRepo,
	}
}

// ServeHTTP implements the http.Handler interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.routes().ServeHTTP(w, r)
}
