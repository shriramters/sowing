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
	CurrentRevisionID int
	ArchivedAt        *time.Time
	Children          []*Page
}
