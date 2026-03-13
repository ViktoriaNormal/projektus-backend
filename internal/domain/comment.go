package domain

import "time"

type Comment struct {
	ID        string
	TaskID    string
	AuthorID  string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	Mentions  []CommentMention
}

type CommentMention struct {
	ID              string
	CommentID       string
	ProjectMemberID string
}

