package viewmodels

import (
	"html/template"
	"sowing/internal/models"
	"time"
)

// RevisionViewModel combines revision and user information for display.
type RevisionViewModel struct {
	ID        int
	CreatedAt time.Time
	Author    string
	Comment   *string
}

// PageData is a unified struct to hold all possible data for any page.
// SiloPages is now a tree structure instead of a flat list.
type PageData struct {
	ShowSidebar  bool
	Silos        []models.Silo
	Silo         models.Silo
	Page         models.Page    // The current page being viewed
	Revisions    []RevisionViewModel
	SiloPages    []*models.Page // The page tree for the sidebar
	Content      template.HTML
	AllSiloPages []models.Page // For the parent dropdown on the new page
	ParentID     int           // The pre-selected parent on the new page
	CurrentUser  *models.User
	IsLoggedIn   bool
}
