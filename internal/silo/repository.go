package silo

import (
	"context"
	"database/sql"
	"fmt"
	"sowing/internal/models"
)

// Repository provides access to the silo storage.
type Repository struct {
	DB *sql.DB
}

// NewRepository creates a new silo repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// FindBySlug finds a silo by its slug.
func (r *Repository) FindBySlug(slug string) (*models.Silo, error) {
	var silo models.Silo
	err := r.DB.QueryRow("SELECT id, slug, name FROM silos WHERE slug = ?", slug).Scan(&silo.ID, &silo.Slug, &silo.Name)
	if err != nil {
		return nil, err
	}
	return &silo, nil
}

// List lists all non-archived silos.
func (r *Repository) List() ([]models.Silo, error) {
	rows, err := r.DB.Query("SELECT id, slug, name, archived_at, cover_image FROM silos WHERE archived_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var silos []models.Silo
	for rows.Next() {
		var silo models.Silo
		if err := rows.Scan(&silo.ID, &silo.Slug, &silo.Name, &silo.ArchivedAt, &silo.CoverImage); err != nil {
			return nil, err
		}
		silos = append(silos, silo)
	}
	return silos, nil
}

// Create creates a new silo, a home page, and an initial revision in a transaction.
func (r *Repository) Create(name, slug string, coverImageURL *string) error {
	ctx := context.Background()
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "INSERT INTO silos (name, slug, cover_image) VALUES (?, ?, ?)", name, slug, coverImageURL)
	if err != nil {
		return fmt.Errorf("error creating silo: %w", err)
	}
	siloID, _ := res.LastInsertId()

	res, err = tx.ExecContext(ctx, "INSERT INTO pages (silo_id, slug, title, current_revision_id) VALUES (?, 'home', 'Home', -1)", siloID)
	if err != nil {
		return fmt.Errorf("error creating home page: %w", err)
	}
	pageID, _ := res.LastInsertId()

	initialContent := fmt.Sprintf("* Welcome to the %s Silo!", name)
	// Assuming author_id 1 for system/initial creation
	res, err = tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, 1, 'Initial creation', ?)", pageID, initialContent)
	if err != nil {
		return fmt.Errorf("error creating initial revision: %w", err)
	}
	revisionID, _ := res.LastInsertId()

	_, err = tx.ExecContext(ctx, "UPDATE pages SET current_revision_id = ? WHERE id = ?", revisionID, pageID)
	if err != nil {
		return fmt.Errorf("error updating page with revision ID: %w", err)
	}

	return tx.Commit()
}
