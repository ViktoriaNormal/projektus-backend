package dto

import "github.com/google/uuid"

type AttachmentResponse struct {
	ID         uuid.UUID  `json:"id"`
	TaskID     *uuid.UUID `json:"taskId,omitempty"`
	CommentID  *uuid.UUID `json:"commentId,omitempty"`
	FileName   string     `json:"fileName"`
	FilePath   string     `json:"filePath"`
	UploadedBy uuid.UUID  `json:"uploadedBy"`
}

