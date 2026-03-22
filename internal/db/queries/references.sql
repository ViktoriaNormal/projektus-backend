-- Reference data queries

-- name: ListRefColumnSystemTypes :many
SELECT key, name, description, sort_order
FROM ref_column_system_types
ORDER BY sort_order ASC;

-- name: ListRefTaskStatusTypes :many
SELECT key, name, description, is_column_type
FROM ref_task_status_types
ORDER BY key ASC;

-- name: ListRefFieldTypes :many
SELECT key, name
FROM ref_field_types
ORDER BY key ASC;

-- name: ListRefEstimationUnits :many
SELECT key, name, available_for
FROM ref_estimation_units
ORDER BY key ASC;

-- name: ListRefPriorityTypes :many
SELECT key, name, available_for, default_values
FROM ref_priority_types
ORDER BY key ASC;

-- name: ListRefSystemTaskFields :many
SELECT key, name, field_type, available_for, description, sort_order
FROM ref_system_task_fields
ORDER BY sort_order ASC;
