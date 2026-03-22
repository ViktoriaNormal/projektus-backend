-- Remove seeded templates (cascade deletes boards, columns, swimlanes, priority values)
DELETE FROM project_templates WHERE name = 'Scrum стандартный' AND project_type = 'scrum';
DELETE FROM project_templates WHERE name = 'Kanban стандартный' AND project_type = 'kanban';
