package web

import (
	"net/http"

	"sowing/internal/web/controller"
	"sowing/internal/web/middleware"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", StaticFileServer()))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	authController := controller.Auth{AuthService: s.authService, Templates: s.templates}
	authController.Register(mux)

	authenticatedMux := http.NewServeMux()
	siloController := controller.Silo{SiloRepo: s.siloRepo, Templates: s.templates}
	siloController.Register(authenticatedMux)

	pageController := controller.Page{PageRepo: s.pageRepo, SiloRepo: s.siloRepo, Templates: s.templates}
	pageController.Register(authenticatedMux)

	miscController := controller.Misc{AttachmentRepo: s.attachmentRepo}
	miscController.Register(authenticatedMux)

	mux.Handle("/", middleware.WithUser(s.authService)(middleware.Auth(s.authService)(authenticatedMux)))

	return mux
}
