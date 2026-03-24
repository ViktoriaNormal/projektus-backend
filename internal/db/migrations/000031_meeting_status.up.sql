ALTER TABLE meetings ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'active';

-- Перенести данные из canceled_at в status
UPDATE meetings SET status = 'cancelled' WHERE canceled_at IS NOT NULL;
