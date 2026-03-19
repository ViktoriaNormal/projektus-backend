package domain

import "github.com/google/uuid"

type ClassOfService string

const (
	ClassOfServiceExpedite   ClassOfService = "expedite"
	ClassOfServiceFixedDate  ClassOfService = "fixed_date"
	ClassOfServiceStandard   ClassOfService = "standard"
	ClassOfServiceIntangible ClassOfService = "intangible"
)

// GetAllDefaultClasses возвращает все классы обслуживания по умолчанию.
func GetAllDefaultClasses() []ClassOfService {
	return []ClassOfService{
		ClassOfServiceExpedite,
		ClassOfServiceFixedDate,
		ClassOfServiceStandard,
		ClassOfServiceIntangible,
	}
}

type SwimlaneSourceType string

const (
	SwimlaneSourceClassOfService SwimlaneSourceType = "class_of_service"
	SwimlaneSourceCustomField    SwimlaneSourceType = "custom_field"
)

type SwimlaneConfig struct {
	BoardID       uuid.UUID          `json:"board_id"`
	SourceType    SwimlaneSourceType `json:"source_type"`
	CustomFieldID *uuid.UUID         `json:"custom_field_id,omitempty"`
	ValueMappings map[string]string  `json:"value_mappings,omitempty"`
}

type WipLimit struct {
	BoardID    uuid.UUID  `json:"board_id"`
	ColumnID   *uuid.UUID `json:"column_id,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlane_id,omitempty"`
	Limit      *int       `json:"limit,omitempty"`
}

type WipCount struct {
	BoardID    uuid.UUID  `json:"board_id"`
	ColumnID   *uuid.UUID `json:"column_id,omitempty"`
	SwimlaneID *uuid.UUID `json:"swimlane_id,omitempty"`
	Count      int        `json:"count"`
	Limit      *int       `json:"limit,omitempty"`
}
