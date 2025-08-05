package models

import "time"

// Page represents a single wiki page within a silo.
type Page struct {
	ID                int
	SiloID            int
	ParentID          *int
	Slug              string
	Title             string
	CurrentRevisionID int
	ArchivedAt        *time.Time
}
