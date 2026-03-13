ALTER TABLE roles
    DROP CONSTRAINT IF EXISTS fk_roles_project;

DROP TABLE IF EXISTS projects;

