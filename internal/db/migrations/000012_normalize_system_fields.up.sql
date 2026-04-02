-- Migration: Remove system fields from `fields` table.
-- System field metadata will be generated from Go constants at runtime.

-- Step 1: Add priority_options column to boards (replaces system field customization).
ALTER TABLE boards ADD COLUMN IF NOT EXISTS priority_options JSONB;

-- Step 2: Copy priority options from system fields into boards.
UPDATE boards b
SET priority_options = f.options
FROM fields f
WHERE f.board_id = b.id AND f.is_system = true AND f.field_type = 'priority'
  AND f.options IS NOT NULL;

-- Step 3: Update swimlane_group_by references from old system field UUIDs to deterministic constants.
-- The new constant UUID for "priority" system field is 00000000-0000-0000-0001-000000000008.
UPDATE boards b
SET swimlane_group_by = '00000000-0000-0000-0001-000000000008'
WHERE EXISTS (
    SELECT 1 FROM fields f
    WHERE f.id::text = b.swimlane_group_by
      AND f.is_system = true
      AND f.field_type = 'priority'
);

-- Also update references to other system fields that might be used in swimlane_group_by.
UPDATE boards b
SET swimlane_group_by = '00000000-0000-0000-0001-000000000005'
WHERE EXISTS (
    SELECT 1 FROM fields f
    WHERE f.id::text = b.swimlane_group_by
      AND f.is_system = true
      AND f.field_type = 'user'
      AND f.name = E'\u0418\u0441\u043f\u043e\u043b\u043d\u0438\u0442\u0435\u043b\u044c'
);

-- Step 4: Delete all system field records from fields table.
DELETE FROM fields WHERE is_system = true;

-- Step 5: Drop is_system column — all remaining records are custom fields.
ALTER TABLE fields DROP COLUMN is_system;
