package dto

type TagResponse struct {
	ID      string `json:"id"`
	BoardID string `json:"board_id"`
	Name    string `json:"name"`
}

type AddTagRequest struct {
	Name string `json:"name" binding:"required"`
}

type SetTaskTagsRequest struct {
	Tags []string `json:"tags" binding:"required"`
}
