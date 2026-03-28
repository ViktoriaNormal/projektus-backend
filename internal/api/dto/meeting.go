package dto

// CreateMeetingRequest соответствует схеме CreateMeetingRequest в OpenAPI.
type CreateMeetingRequest struct {
	ProjectID      *string  `json:"project_id"`                      // uuid, опционально
	Name           string   `json:"name" binding:"required"`
	Description    *string  `json:"description"`
	MeetingType    *string  `json:"meeting_type"`                    // один из типов из ТЗ, но пока строка
	Location       string   `json:"location" binding:"required"`
	StartTime      string   `json:"start_time" binding:"required"`   // RFC3339
	EndTime        string   `json:"end_time" binding:"required"`     // RFC3339
	ParticipantIDs []string `json:"participant_ids"`                 // uuid-строки
}

// UpdateMeetingRequest соответствует UpdateMeetingRequest в OpenAPI.
type UpdateMeetingRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	MeetingType *string `json:"meeting_type"`
	Location    *string `json:"location"`
	StartTime   *string `json:"start_time"` // RFC3339
	EndTime     *string `json:"end_time"`   // RFC3339
}

type MeetingParticipantResponse struct {
	ID        string `json:"id"`
	MeetingID string `json:"meeting_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"`
}

type MeetingResponse struct {
	ID          string  `json:"id"`
	ProjectID   *string `json:"project_id,omitempty"`
	OrganizerID string  `json:"organizer_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	MeetingType string  `json:"meeting_type"`
	Location    *string `json:"location"`
	Status      string  `json:"status"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
}

type MeetingDetailsResponse struct {
	MeetingResponse
	Participants []MeetingParticipantResponse `json:"participants"`
}
