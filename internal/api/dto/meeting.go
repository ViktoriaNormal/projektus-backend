package dto

// CreateMeetingRequest соответствует схеме CreateMeetingRequest в OpenAPI.
type CreateMeetingRequest struct {
	ProjectID      *string  `json:"projectId"`                    // uuid, опционально
	Name           string   `json:"name" binding:"required"`
	Description    *string  `json:"description"`
	MeetingType    *string  `json:"meetingType"`                  // один из типов из ТЗ, но пока строка
	Location       string   `json:"location" binding:"required"`
	StartTime      string   `json:"startTime" binding:"required"` // RFC3339
	EndTime        string   `json:"endTime" binding:"required"`   // RFC3339
	ParticipantIDs []string `json:"participantIds"`               // uuid-строки
}

// UpdateMeetingRequest соответствует UpdateMeetingRequest в OpenAPI.
type UpdateMeetingRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	MeetingType *string `json:"meetingType"`
	Location    *string `json:"location"`
	StartTime   *string `json:"startTime"` // RFC3339
	EndTime     *string `json:"endTime"`   // RFC3339
}

type MeetingParticipantResponse struct {
	ID        string `json:"id"`
	MeetingID string `json:"meetingId"`
	UserID    string `json:"userId"`
	Status    string `json:"status"`
}

type MeetingResponse struct {
	ID          string  `json:"id"`
	ProjectID   *string `json:"projectId,omitempty"`
	OrganizerID string  `json:"organizerId"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	MeetingType string  `json:"meetingType"`
	Location    *string `json:"location"`
	Status      string  `json:"status"`
	StartTime   string  `json:"startTime"`
	EndTime     string  `json:"endTime"`
}

type MeetingDetailsResponse struct {
	MeetingResponse
	Participants []MeetingParticipantResponse `json:"participants"`
}

