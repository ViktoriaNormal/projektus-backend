package domain

import (
	"time"

	"github.com/google/uuid"
)

// Deterministic UUIDs for system board fields (task fields).
// These are stable constants — they never change across boards or projects.
var SystemBoardFieldIDs = map[string]uuid.UUID{
	"title":       uuid.MustParse("00000000-0000-0000-0001-000000000001"),
	"description": uuid.MustParse("00000000-0000-0000-0001-000000000002"),
	"status":      uuid.MustParse("00000000-0000-0000-0001-000000000003"),
	"author":      uuid.MustParse("00000000-0000-0000-0001-000000000004"),
	"assignee":    uuid.MustParse("00000000-0000-0000-0001-000000000005"),
	"watchers":    uuid.MustParse("00000000-0000-0000-0001-000000000006"),
	"deadline":    uuid.MustParse("00000000-0000-0000-0001-000000000007"),
	"priority":    uuid.MustParse("00000000-0000-0000-0001-000000000008"),
	"estimation":  uuid.MustParse("00000000-0000-0000-0001-000000000009"),
	"sprint":      uuid.MustParse("00000000-0000-0000-0001-000000000010"),
	"created_at":  uuid.MustParse("00000000-0000-0000-0001-000000000011"),
}

// Deterministic UUIDs for system project params.
var SystemProjectParamIDs = map[string]uuid.UUID{
	"project_name":    uuid.MustParse("00000000-0000-0000-0002-000000000001"),
	"project_desc":    uuid.MustParse("00000000-0000-0000-0002-000000000002"),
	"project_status":  uuid.MustParse("00000000-0000-0000-0002-000000000003"),
	"project_owner":   uuid.MustParse("00000000-0000-0000-0002-000000000004"),
	"project_created": uuid.MustParse("00000000-0000-0000-0002-000000000005"),
}

// DefaultProjectParams defines system project parameters with metadata.
var DefaultProjectParams = []struct {
	Key        string
	Name       string
	FieldType  string
	IsRequired bool
}{
	{Key: "project_name", Name: "Название", FieldType: "text", IsRequired: true},
	{Key: "project_desc", Name: "Описание", FieldType: "text", IsRequired: false},
	{Key: "project_status", Name: "Статус проекта", FieldType: "select", IsRequired: true},
	{Key: "project_owner", Name: "Ответственный за проект", FieldType: "user", IsRequired: true},
	{Key: "project_created", Name: "Дата создания", FieldType: "datetime", IsRequired: true},
}

// GenerateSystemBoardFields generates system board field metadata from constants.
// Filters by projectType (scrum/kanban) and uses board config for dynamic options.
func GenerateSystemBoardFields(projectType, priorityType, estimationUnit string, priorityOptions []string, allFieldDefs []DefaultBoardFieldDef) []BoardCustomField {
	var result []BoardCustomField
	for _, def := range allFieldDefs {
		if !isAvailableFor(def.AvailableFor, projectType) {
			continue
		}
		id, ok := SystemBoardFieldIDs[def.Key]
		if !ok {
			continue
		}
		options := def.Options
		if def.Key == "priority" {
			if len(priorityOptions) > 0 {
				options = priorityOptions
			} else if len(options) == 0 {
				options = defaultPriorityOptionsForType(priorityType)
			}
		}
		result = append(result, BoardCustomField{
			ID:         id.String(),
			Name:       def.Name,
			FieldType:  def.FieldType,
			IsSystem:   true,
			IsRequired: def.IsRequired,
			Options:    options,
		})
	}
	return result
}

// GenerateSystemProjectParams generates system project params with values from a project.
func GenerateSystemProjectParams(project *Project) []ProjectParam {
	var desc *string
	if project.Description != nil {
		desc = project.Description
	}
	status := string(project.Status)
	ownerID := project.OwnerID.String()
	createdAt := project.CreatedAt.Format(time.RFC3339)
	name := project.Name

	valueMap := map[string]*string{
		"project_name":    &name,
		"project_desc":    desc,
		"project_status":  &status,
		"project_owner":   &ownerID,
		"project_created": &createdAt,
	}

	var result []ProjectParam
	for _, def := range DefaultProjectParams {
		id := SystemProjectParamIDs[def.Key]
		result = append(result, ProjectParam{
			ID:         id.String(),
			ProjectID:  project.ID.String(),
			Name:       def.Name,
			FieldType:  def.FieldType,
			IsSystem:   true,
			IsRequired: def.IsRequired,
			Options:    nil,
			Value:      valueMap[def.Key],
		})
	}
	return result
}

// GenerateSystemProjectParamsForTemplate generates system project params metadata (no values).
func GenerateSystemProjectParamsForTemplate() []ProjectParam {
	var result []ProjectParam
	for _, def := range DefaultProjectParams {
		id := SystemProjectParamIDs[def.Key]
		result = append(result, ProjectParam{
			ID:         id.String(),
			Name:       def.Name,
			FieldType:  def.FieldType,
			IsSystem:   true,
			IsRequired: def.IsRequired,
		})
	}
	return result
}

func defaultPriorityOptionsForType(priorityType string) []string {
	switch priorityType {
	case "priority":
		return []string{"Низкий", "Средний", "Высокий", "Критичный"}
	case "service_class":
		return []string{"Ускоренный", "С фиксированной датой", "Стандартный", "Нематериальный"}
	}
	return nil
}

func isAvailableFor(availableFor []string, projectType string) bool {
	for _, t := range availableFor {
		if t == projectType {
			return true
		}
	}
	return false
}

