package page

import (
	"context"
	"database/sql"
	"fmt"
	"sowing/internal/models"
	"time"
)

// Repository provides access to the page storage.
type Repository struct {
	DB *sql.DB
}

// NewRepository creates a new page repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// FindByPath iteratively queries the database to find a page by its hierarchical path.
func (r *Repository) FindByPath(siloID int, path []string) (models.Page, error) {
	if len(path) == 0 {
		return models.Page{}, sql.ErrNoRows
	}

	var page models.Page
	var parentID *int

	for _, slug := range path {
		var err error
		var query string
		if parentID == nil {
			query = "SELECT id, title, current_revision_id, slug, parent_id FROM pages WHERE silo_id = ? AND slug = ? AND parent_id IS NULL"
			err = r.DB.QueryRow(query, siloID, slug).Scan(&page.ID, &page.Title, &page.CurrentRevisionID, &page.Slug, &page.ParentID)
		} else {
			query = "SELECT id, title, current_revision_id, slug, parent_id FROM pages WHERE silo_id = ? AND slug = ? AND parent_id = ?"
			err = r.DB.QueryRow(query, siloID, slug, *parentID).Scan(&page.ID, &page.Title, &page.CurrentRevisionID, &page.Slug, &page.ParentID)
		}

		if err != nil {
			return models.Page{}, err
		}

		pageID := page.ID
		parentID = &pageID
	}
	return page, nil
}

// GetRevisionContent gets the content of a specific revision.
func (r *Repository) GetRevisionContent(revisionID int) (string, error) {
	var content string
	err := r.DB.QueryRow("SELECT content FROM revisions WHERE id = ?", revisionID).Scan(&content)
	return content, err
}

// ListBySilo lists all non-archived pages for a given silo.
func (r *Repository) ListBySilo(siloID int) ([]models.Page, error) {
	rows, err := r.DB.Query("SELECT id, slug, title, parent_id, position FROM pages WHERE silo_id = ? AND archived_at IS NULL ORDER BY position ASC", siloID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allSiloPages []models.Page
	for rows.Next() {
		var page models.Page
		if err := rows.Scan(&page.ID, &page.Slug, &page.Title, &page.ParentID, &page.Position); err != nil {
			return nil, err
		}
		allSiloPages = append(allSiloPages, page)
	}
	return allSiloPages, nil
}

// Create creates a new page and its initial revision in a transaction.
func (r *Repository) Create(ctx context.Context, page *models.Page, revision *models.Revision) (int64, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	var parentIDPtr *int
	if page.ParentID != nil {
		parentIDPtr = page.ParentID
	}

	res, err := tx.ExecContext(ctx, "INSERT INTO pages (silo_id, parent_id, slug, title, current_revision_id) VALUES (?, ?, ?, ?, -1)", page.SiloID, parentIDPtr, page.Slug, page.Title)
	if err != nil {
		return 0, fmt.Errorf("error creating page: %w", err)
	}
	pageID, _ := res.LastInsertId()
	page.ID = int(pageID)

	res, err = tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, ?, ?, ?)", page.ID, revision.AuthorID, revision.Comment, revision.Content)
	if err != nil {
		return 0, fmt.Errorf("error creating revision: %w", err)
	}
	revisionID, _ := res.LastInsertId()

	_, err = tx.ExecContext(ctx, "UPDATE pages SET current_revision_id = ? WHERE id = ?", revisionID, page.ID)
	if err != nil {
		return 0, fmt.Errorf("error updating page with revision ID: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error committing transaction: %w", err)
	}
	return pageID, nil
}

// GetPathByID recursively finds the full path of a page given its ID.
func (r *Repository) GetPathByID(pageID int) (string, error) {
	var slug string
	var parentID sql.NullInt64
	err := r.DB.QueryRow("SELECT slug, parent_id FROM pages WHERE id = ?", pageID).Scan(&slug, &parentID)
	if err != nil {
		return "", err
	}

	if !parentID.Valid {
		return slug, nil
	}

	parentPath, err := r.GetPathByID(int(parentID.Int64))
	if err != nil {
		return "", err
	}

	return parentPath + "/" + slug, nil
}

// CreateRevision creates a new revision for a page and updates the page's current_revision_id.
func (r *Repository) CreateRevision(ctx context.Context, revision *models.Revision, pageID int) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, "INSERT INTO revisions (page_id, author_id, comment, content) VALUES (?, ?, ?, ?)", pageID, revision.AuthorID, revision.Comment, revision.Content)
	if err != nil {
		return fmt.Errorf("error creating revision: %w", err)
	}
	revisionID, _ := res.LastInsertId()

	_, err = tx.ExecContext(ctx, "UPDATE pages SET current_revision_id = ? WHERE id = ?", revisionID, pageID)
	if err != nil {
		return fmt.Errorf("error updating page with revision ID: %w", err)
	}

	return tx.Commit()
}

// Delete archives a page.
func (r *Repository) Delete(pageID int) error {
	_, err := r.DB.Exec("UPDATE pages SET archived_at = ? WHERE id = ?", time.Now(), pageID)
	return err
}
