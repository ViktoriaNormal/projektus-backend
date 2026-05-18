-- Дополнение к 041: завершения PORTAL в ISO-неделе 21 (12–18.05.2026) для графика throughput.

SET client_encoding = 'UTF8';

UPDATE tasks
SET column_id = 'd0010000-0000-0000-0002-000000000005'::uuid
WHERE id IN (
    'e1000000-0000-0000-0002-000000000043'::uuid,
    'e1000000-0000-0000-0002-000000000047'::uuid,
    'e1000000-0000-0000-0002-000000000048'::uuid
)
AND project_id = 'c0000000-0000-0000-0002-000000000000'::uuid;

DO $$
DECLARE
    rec RECORD;
    idx INT := 0;
BEGIN
    FOR rec IN
        SELECT *
        FROM (VALUES
            ('e1000000-0000-0000-0002-000000000043'::uuid, TIMESTAMPTZ '2026-05-17 11:00:00+03'),
            ('e1000000-0000-0000-0002-000000000047'::uuid, TIMESTAMPTZ '2026-05-17 15:30:00+03'),
            ('e1000000-0000-0000-0002-000000000048'::uuid, TIMESTAMPTZ '2026-05-18 10:00:00+03')
        ) AS v(task_id, completed_at)
    LOOP
        idx := idx + 1;
        UPDATE task_status_history
        SET left_at = rec.completed_at
        WHERE task_id = rec.task_id AND left_at IS NULL;

        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (
            ('f4200000-0000-0000-0002-' || lpad(idx::text, 12, '0'))::uuid,
            rec.task_id,
            'd0010000-0000-0000-0002-000000000005'::uuid,
            rec.completed_at,
            NULL
        )
        ON CONFLICT (id) DO NOTHING;
    END LOOP;
END $$;
