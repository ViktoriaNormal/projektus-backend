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
	BoardID       uuid.UUID
	SourceType    SwimlaneSourceType
	CustomFieldID *uuid.UUID
	ValueMappings map[string]string
}

type WipLimit struct {
	BoardID   uuid.UUID
	ColumnID  *uuid.UUID
	SwimlaneID *uuid.UUID
	Limit     *int
}

type WipCount struct {
	BoardID   uuid.UUID
	ColumnID  *uuid.UUID
	SwimlaneID *uuid.UUID
	Count     int
	Limit     *int
}

