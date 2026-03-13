package dto

import "github.com/google/uuid"

type ClassOfServiceResponse struct {
	Value       string `json:"value"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateTaskClassRequest struct {
	ClassOfService string `json:"classOfService" binding:"required"`
}

type SwimlaneConfigRequest struct {
	SourceType    string            `json:"sourceType" binding:"required,oneof=class_of_service custom_field"`
	CustomFieldID *uuid.UUID        `json:"customFieldId,omitempty"`
	ValueMappings map[string]string `json:"valueMappings,omitempty"`
}

