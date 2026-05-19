-- =============================================================================
-- Migration 053: PORTAL «Дизайн и контент» — throughput 12 нед., cycle time с хвостом
--  - 16 завершённых (51–62, 70 + 63–65); активные: 71, 72
--  - Завершения W10–W21; cycle 2.5–18 дн.
-- ref: 18.05.2026 | ID: f0530000-0000-0000-0002-*
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_053_history_backup;
DROP TABLE IF EXISTS migration_053_tasks_backup;

CREATE TABLE migration_053_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
  AND t.deleted_at IS NULL;

CREATE TABLE migration_053_tasks_backup AS
SELECT id, column_id, deadline
FROM tasks
WHERE board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
  AND deleted_at IS NULL;

CREATE OR REPLACE FUNCTION pg_temp.iso_week_done_ts(p_week TEXT, p_offset_hours INT)
RETURNS TIMESTAMPTZ
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT (
        to_date(
            substring(p_week from 1 for 4) || substring(p_week from 7 for 2),
            'IYYYIW'
        )::timestamp
        + INTERVAL '1 day' * 3
        + make_interval(hours => (10 + (p_offset_hours % 30))::int)
    ) AT TIME ZONE 'Europe/Moscow';
$$;

DO $$
DECLARE
    design_board   UUID := 'd0000000-0000-0000-0002-000000000002'::uuid;
    col_ideas      UUID := 'd0010000-0000-0000-0002-000000000011'::uuid;
    col_work       UUID := 'd0010000-0000-0000-0002-000000000012'::uuid;
    col_review     UUID := 'd0010000-0000-0000-0002-000000000013'::uuid;
    col_done       UUID := 'd0010000-0000-0000-0002-000000000014'::uuid;

    cfd_start      TIMESTAMPTZ := TIMESTAMPTZ '2026-04-19 00:00:00+03';

    week_plan TEXT[] := ARRAY[
        '2026-W10', '2026-W11', '2026-W12', '2026-W13',
        '2026-W14', '2026-W14',
        '2026-W15', '2026-W15',
        '2026-W16',
        '2026-W17', '2026-W17',
        '2026-W18', '2026-W18',
        '2026-W19',
        '2026-W20', '2026-W21'
    ];
    cycle_plan DOUBLE PRECISION[] := ARRAY[
        2.5, 3.0, 4.5, 5.5,
        6.0, 6.5,
        7.0, 7.5,
        8.5,
        9.5, 10.5,
        12.0, 14.0,
        16.5, 18.0, 7.0
    ];

    task_rec       RECORD;
    idx            INT;
    done_ts        TIMESTAMPTZ;
    cycle_days     DOUBLE PRECISION;
    todo_enter     TIMESTAMPTZ;
    work_enter     TIMESTAMPTZ;
    review_enter   TIMESTAMPTZ;
    ideas_days     INT;
    hist_id        UUID;
BEGIN
    -- Довести до 16 завершённых: 54–60, 64–65 (51–53, 55, 61–63, 70 уже done)
    UPDATE tasks
    SET column_id = col_done
    WHERE id IN (
        'e1000000-0000-0000-0002-000000000054'::uuid,
        'e1000000-0000-0000-0002-000000000056'::uuid,
        'e1000000-0000-0000-0002-000000000057'::uuid,
        'e1000000-0000-0000-0002-000000000058'::uuid,
        'e1000000-0000-0000-0002-000000000059'::uuid,
        'e1000000-0000-0000-0002-000000000060'::uuid,
        'e1000000-0000-0000-0002-000000000064'::uuid,
        'e1000000-0000-0000-0002-000000000065'::uuid
    )
      AND board_id = design_board
      AND deleted_at IS NULL;

    idx := 0;
    FOR task_rec IN
        SELECT t.id, t.created_at
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = design_board
          AND t.deleted_at IS NULL
          AND c.system_type = 'completed'
        ORDER BY hashtext(t.id::text)
    LOOP
        idx := idx + 1;
        IF idx > array_length(week_plan, 1) THEN
            RAISE EXCEPTION '053: more completed tasks (%) than week_plan slots (%)',
                idx, array_length(week_plan, 1);
        END IF;

        done_ts := pg_temp.iso_week_done_ts(week_plan[idx], idx + 300);
        cycle_days := cycle_plan[idx];

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.4)::int);
        ideas_days := 2 + (abs(hashtext(task_rec.id::text || '-ideas')) % 4);
        todo_enter := work_enter - make_interval(days => ideas_days);
        todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '5 days');

        DELETE FROM task_status_history WHERE task_id = task_rec.id;

        hist_id := ('f0530000-0000-0000-0002-' || lpad((idx * 10 + 1)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_ideas, todo_enter, work_enter);

        hist_id := ('f0530000-0000-0000-0002-' || lpad((idx * 10 + 2)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_work, work_enter, review_enter);

        hist_id := ('f0530000-0000-0000-0002-' || lpad((idx * 10 + 3)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_review, review_enter, done_ts);

        hist_id := ('f0530000-0000-0000-0002-' || lpad((idx * 10 + 4)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_done, done_ts, NULL);
    END LOOP;

    IF idx <> array_length(week_plan, 1) THEN
        RAISE EXCEPTION '053: expected % completed tasks, got %', array_length(week_plan, 1), idx;
    END IF;
END $$;

-- Дедлайны завершённых на дизайн-доске
DO $$
DECLARE
    ref_ts TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 18:00:00+03';
BEGIN
    UPDATE tasks t
    SET deadline = GREATEST(
        sub.done_ts + INTERVAL '1 day',
        ref_ts + INTERVAL '1 day' + make_interval(hours => (abs(hashtext(t.id::text)) % 12)::int)
    )
    FROM (
        SELECT h.task_id, MAX(h.entered_at) AS done_ts
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE c.system_type = 'completed'
        GROUP BY h.task_id
    ) sub
    WHERE t.id = sub.task_id
      AND t.deleted_at IS NULL
      AND t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
      AND EXISTS (
          SELECT 1 FROM columns c
          WHERE c.id = t.column_id AND c.system_type = 'completed'
      );
END $$;
