-- Revert: set all system fields back to is_required = false (except Название)
UPDATE fields SET is_required = false WHERE id::text LIKE '20000000%' AND name != 'Название';
