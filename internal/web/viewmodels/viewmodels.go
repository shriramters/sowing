package viewmodels

import (
	"html/template"
	"sowing/internal/models"
)

// PageData is a unified struct to hold all possible data for any page.
// SiloPages is now a tree structure instead of a flat list.
type PageData struct {
	ShowSidebar  bool
	Silos        []models.Silo
	Silo         models.Silo
	Page         models.Page    // The current page being viewed
	SiloPages    []*models.Page // The page tree for the sidebar
	Content      template.HTML
	AllSiloPages []models.Page // For the parent dropdown on the new page
	ParentID     int           // The pre-selected parent on the new page
	CurrentUser  *models.User
	IsLoggedIn   bool
}
