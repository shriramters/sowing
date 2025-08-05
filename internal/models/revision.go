package models

import "time"

// Revision represents a version of a page's content.
type Revision struct {
	ID        int
	PageID    int
	Content   string
	AuthorID  int
	Comment   *string
	CreatedAt time.Time
}
