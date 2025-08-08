package controller

import (
	"html/template"
	"log"
	"net/http"

	"sowing/internal/auth"
)

// Auth provides auth handlers
type Auth struct {
	AuthService *auth.Service
	Templates   map[string]*template.Template
}

// Register registers the auth routes
func (a *Auth) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", a.loginGet)
	mux.HandleFunc("POST /login", a.loginPost)
	mux.HandleFunc("GET /logout", a.logout)
	mux.HandleFunc("GET /register", a.registerGet)
	mux.HandleFunc("POST /register", a.registerPost)
}

func (a *Auth) loginGet(w http.ResponseWriter, r *http.Request) {
	err := a.Templates["login.html"].ExecuteTemplate(w, "layout.html", nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func (a *Auth) loginPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	_, err := a.AuthService.Login(w, r, username, password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	a.AuthService.Logout(w, r)
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *Auth) registerGet(w http.ResponseWriter, r *http.Request) {
	err := a.Templates["register.html"].ExecuteTemplate(w, "layout.html", nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Internal Server Error", 500)
	}
}

func (a *Auth) registerPost(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	displayName := r.FormValue("display_name")
	password := r.FormValue("password")

	_, err := a.AuthService.RegisterUser(w, r, username, displayName, password)
	if err != nil {
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}
