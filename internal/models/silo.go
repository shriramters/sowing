package models

import "time"

type Silo struct {
	ID         int
	Slug       string
	Name       string
	ArchivedAt *time.Time
}
