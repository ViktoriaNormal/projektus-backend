package domain

import "time"

type Attachment struct {
	ID         string
	TaskID     *string
	CommentID  *string
	FileName   string
	FilePath   string
	UploadedBy string
	UploadedAt time.Time
}

