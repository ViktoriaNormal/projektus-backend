package domain

import "time"

type Attachment struct {
	ID          string    `json:"id"`
	TaskID      *string   `json:"task_id,omitempty"`
	CommentID   *string   `json:"comment_id,omitempty"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	UploadedBy  string    `json:"uploaded_by"`
	UploadedAt  time.Time `json:"uploaded_at"`
}
