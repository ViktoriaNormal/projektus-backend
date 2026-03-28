package dto

import "github.com/google/uuid"

type AttachmentResponse struct {
	ID         uuid.UUID  `json:"id"`
	TaskID     *uuid.UUID `json:"task_id,omitempty"`
	CommentID  *uuid.UUID `json:"comment_id,omitempty"`
	FileName   string     `json:"file_name"`
	FilePath   string     `json:"file_path"`
	UploadedBy uuid.UUID  `json:"uploaded_by"`
}
