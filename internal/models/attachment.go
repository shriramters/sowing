package models

import "time"

// Attachment represents an uploaded file.
type Attachment struct {
	ID             int
	Filename       string
	UniqueFilename string
	MimeType       string
	Size           int64
	CreatedAt      time.Time
}
