package attachment

import (
	"database/sql"
	"sowing/internal/models"
)

// Repository provides access to the attachment storage.
type Repository struct {
	DB *sql.DB
}

// NewRepository creates a new attachment repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// Create inserts a new attachment record into the database.
func (r *Repository) Create(attachment *models.Attachment) error {
	_, err := r.DB.Exec(
		"INSERT INTO attachments (filename, unique_filename, mime_type, size) VALUES (?, ?, ?, ?)",
		attachment.Filename, attachment.UniqueFilename, attachment.MimeType, attachment.Size)
	return err
}
