package domain

import (
	"time"

	"github.com/google/uuid"
)

type Attachment struct {
	ID          uuid.UUID  `json:"id"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	CommentID   *uuid.UUID `json:"comment_id,omitempty"`
	FileName    string     `json:"file_name"`
	FilePath    string     `json:"file_path"`
	FileSize    int64      `json:"file_size"`
	ContentType string     `json:"content_type"`
	UploadedBy  uuid.UUID  `json:"uploaded_by"`
	UploadedAt  time.Time  `json:"uploaded_at"`
}
