package models

// Identity represents a user's authentication method.
type Identity struct {
	ID             int
	UserID         int
	Provider       string
	ProviderUserID string
	PasswordHash   *string
}
