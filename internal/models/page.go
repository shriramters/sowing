package models

import "time"

// Page represents a single wiki page within a silo.
// The Children field is added to support a hierarchical structure.
type Page struct {
	ID                int
	SiloID            int
	ParentID          *int
	Slug              string
	Title             string
	Position          int // The order of the page within its level
	CurrentRevisionID int
	ArchivedAt        *time.Time
	Children          []*Page
	Path              string // for convenience, not stored in db, e.g., "servers/web-server"
}
