-- Fix duplicate templates: keep only the first of each (name, project_type) pair
-- Encoding: UTF-8

SET client_encoding = 'UTF8';

-- Delete duplicate Scrum templates (keep the oldest one)
DELETE FROM project_templates
WHERE id IN (
    SELECT id FROM (
        SELECT id, ROW_NUMBER() OVER (PARTITION BY name, project_type ORDER BY created_at ASC) AS rn
        FROM project_templates
    ) sub
    WHERE sub.rn > 1
);
