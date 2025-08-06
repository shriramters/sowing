package auth

import (
	"database/sql"
	"sowing/internal/models"
)

// Repository provides access to the authentication storage.
type Repository struct {
	DB *sql.DB
}

// NewRepository creates a new authentication repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// FindUserByUsername finds a user by their username.
func (r *Repository) FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.DB.QueryRow("SELECT id, username, display_name FROM users WHERE username = ?", username).Scan(&user.ID, &user.Username, &user.DisplayName)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindIdentityByProvider finds an identity by provider and provider user ID.
func (r *Repository) FindIdentityByProvider(provider, providerUserID string) (*models.Identity, error) {
	var identity models.Identity
	err := r.DB.QueryRow("SELECT id, user_id, provider, provider_user_id, password_hash FROM identities WHERE provider = ? AND provider_user_id = ?", provider, providerUserID).Scan(&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID, &identity.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

// CreateUser creates a new user and a corresponding identity.
func (r *Repository) CreateUser(user *models.User, identity *models.Identity) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	res, err := tx.Exec("INSERT INTO users (username, display_name) VALUES (?, ?)", user.Username, user.DisplayName)
	if err != nil {
		tx.Rollback()
		return err
	}

	userID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return err
	}

	identity.UserID = int(userID)

	_, err = tx.Exec("INSERT INTO identities (user_id, provider, provider_user_id, password_hash) VALUES (?, ?, ?, ?)", identity.UserID, identity.Provider, identity.ProviderUserID, identity.PasswordHash)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
