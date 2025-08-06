package auth

import (
	"context"
	"encoding/gob"
	"errors"
	"net/http"
	

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"sowing/internal/models"
)

// Store will hold the session store.
var Store *sessions.CookieStore

func InitSessionStore(sessionKey string) error {
	if len(sessionKey) < 32 {
		return errors.New("session key must be at least 32 characters long")
	}
	Store = sessions.NewCookieStore([]byte(sessionKey))
	Store.Options.HttpOnly = true
	Store.Options.Path = "/"
	Store.Options.SameSite = http.SameSiteLaxMode // Protect against CSRF
	return nil
}

func init() {
	gob.Register(&models.User{})
}

// Service provides authentication-related services.
type Service struct {
	Repo *Repository
}

// NewService creates a new authentication service.
func NewService(repo *Repository) *Service {
	return &Service{Repo: repo}
}

// RegisterUser creates a new user.
func (s *Service) RegisterUser(w http.ResponseWriter, r *http.Request, username, displayName, password string) (*models.User, error) {
	// Check if user already exists
	if _, err := s.Repo.FindUserByUsername(username); err == nil {
		return nil, errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	passwordHash := string(hashedPassword)

	user := &models.User{
		Username:    username,
		DisplayName: displayName,
	}
	identity := &models.Identity{
		Provider:       "local",
		ProviderUserID: username,
		PasswordHash:   &passwordHash,
	}

	err = s.Repo.CreateUser(user, identity)
	if err != nil {
		return nil, err
	}

	// Retrieve the full user model with the ID
	createdUser, err := s.Repo.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}

	return createdUser, nil
}


// Login authenticates a user and creates a session.
func (s *Service) Login(w http.ResponseWriter, r *http.Request, username, password string) (*models.User, error) {
	user, err := s.Repo.FindUserByUsername(username)
	if err != nil {
		return nil, err
	}

	identity, err := s.Repo.FindIdentityByProvider("local", username)
	if err != nil {
		return nil, err
	}

	if identity.PasswordHash == nil {
		return nil, errors.New("user has no password set")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*identity.PasswordHash), []byte(password)); err != nil {
		return nil, err
	}

	session, _ := Store.Get(r, "sowing-session")
	session.Values["user"] = user

	// Set Secure flag based on request scheme or X-Forwarded-Proto header
	// This is crucial for correct behavior behind reverse proxies.
	session.Options.Secure = r.URL.Scheme == "https" || r.Header.Get("X-Forwarded-Proto") == "https"

	session.Save(r, w)

	return user, nil
}

// Logout destroys a user's session.
func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := Store.Get(r, "sowing-session")
	delete(session.Values, "user")

	// Ensure Secure flag is set correctly for logout cookie as well
	session.Options.Secure = r.URL.Scheme == "https" || r.Header.Get("X-Forwarded-Proto") == "https"

	session.Save(r, w)
}

// GetCurrentUser returns the currently logged-in user.
func (s *Service) GetCurrentUser(r *http.Request) *models.User {
	session, _ := Store.Get(r, "sowing-session")
	if user, ok := session.Values["user"].(*models.User); ok {
		return user
	}
	return nil
}

// Middleware to protect routes that require authentication.
func (s *Service) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.GetCurrentUser(r) == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithUser adds the current user to the request context.
func (s *Service) WithUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := s.GetCurrentUser(r)
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}