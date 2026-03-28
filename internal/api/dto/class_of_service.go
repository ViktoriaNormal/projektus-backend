package dto

import "github.com/google/uuid"

type ClassOfServiceResponse struct {
	Value       string `json:"value"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateTaskClassRequest struct {
	ClassOfService string `json:"class_of_service" binding:"required"`
}

type SwimlaneConfigRequest struct {
	SourceType    string            `json:"source_type" binding:"required,oneof=class_of_service custom_field"`
	CustomFieldID *uuid.UUID        `json:"custom_field_id,omitempty"`
	ValueMappings map[string]string `json:"value_mappings,omitempty"`
}
