-- =============================================================================
-- Migration 049: дедлайны (все задачи 038+), cycle time ~60% в 5–7 дн., блокировки
-- ref: 18.05.2026
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_049_deadline_backup;
DROP TABLE IF EXISTS migration_049_history_backup;

CREATE TABLE migration_049_deadline_backup AS
SELECT t.id AS task_id, t.deadline
FROM tasks t
WHERE t.project_id IN (
    'c0000000-0000-0000-0001-000000000000'::uuid,
    'c0000000-0000-0000-0002-000000000000'::uuid
)
AND t.deleted_at IS NULL;

CREATE TABLE migration_049_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
  AND t.deleted_at IS NULL
  AND EXISTS (
      SELECT 1 FROM columns c
      WHERE c.id = t.column_id AND c.system_type = 'completed'
  );

-- ── 1. Дедлайны: завершённые + активные (оба проекта, все доски) ─────────────
DO $$
DECLARE
    ref_ts TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 18:00:00+03';
BEGIN
    -- Завершённые: deadline >= дата завершения + 1 дн. и не «просрочен» на ref (май 2026)
    UPDATE tasks t
    SET deadline = GREATEST(
        sub.done_ts + INTERVAL '1 day',
        ref_ts + INTERVAL '1 day' + make_interval(hours => (abs(hashtext(t.id::text)) % 12)::int)
    )
    FROM (
        SELECT
            h.task_id,
            MAX(h.entered_at) AS done_ts
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE c.system_type = 'completed'
        GROUP BY h.task_id
    ) sub
    WHERE t.id = sub.task_id
      AND t.deleted_at IS NULL
      AND t.project_id IN (
          'c0000000-0000-0000-0001-000000000000'::uuid,
          'c0000000-0000-0000-0002-000000000000'::uuid
      )
      AND EXISTS (
          SELECT 1 FROM columns c
          WHERE c.id = t.column_id AND c.system_type = 'completed'
      );

    -- Завершённые без истории
    UPDATE tasks t
    SET deadline = ref_ts + make_interval(days => (14 + (abs(hashtext(t.id::text)) % 30))::int)
    FROM columns c
    WHERE c.id = t.column_id
      AND t.deleted_at IS NULL
      AND c.system_type = 'completed'
      AND t.project_id IN (
          'c0000000-0000-0000-0001-000000000000'::uuid,
          'c0000000-0000-0000-0002-000000000000'::uuid
      )
      AND NOT EXISTS (
          SELECT 1 FROM task_status_history h
          JOIN columns c2 ON c2.id = h.column_id
          WHERE h.task_id = t.id AND c2.system_type = 'completed'
      );

    -- Активные: ≤5 просроченных, 3 «скоро», остальные в будущем
    WITH active AS (
        SELECT
            t.id,
            t.project_id,
            row_number() OVER (
                PARTITION BY t.project_id
                ORDER BY abs(hashtext(t.id::text || 'dl049'))
            ) AS rn
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.deleted_at IS NULL
          AND c.system_type <> 'completed'
          AND t.project_id IN (
              'c0000000-0000-0000-0001-000000000000'::uuid,
              'c0000000-0000-0000-0002-000000000000'::uuid
          )
    )
    UPDATE tasks t
    SET deadline = CASE
        WHEN a.rn <= 5 THEN
            ref_ts - make_interval(days => (2 + ((a.rn - 1) % 8))::int)
            - make_interval(hours => (abs(hashtext(t.id::text)) % 6)::int)
        WHEN a.rn <= 8 THEN
            ref_ts + make_interval(days => (a.rn - 5)::int)
            + make_interval(hours => (10 + (abs(hashtext(t.id::text)) % 5))::int)
        ELSE
            ref_ts + make_interval(days => (7 + (abs(hashtext(t.id::text || 'fut049')) % 28))::int)
            + make_interval(hours => 12)
    END
    FROM active a
    WHERE t.id = a.id;

    -- Активные без дедлайна — часть «скоро»
    UPDATE tasks t
    SET deadline = ref_ts + make_interval(days => 1)
        + make_interval(hours => (abs(hashtext(t.id::text)) % 8)::int)
    WHERE t.deleted_at IS NULL
      AND t.deadline IS NULL
      AND t.project_id IN (
          'c0000000-0000-0000-0001-000000000000'::uuid,
          'c0000000-0000-0000-0002-000000000000'::uuid
      )
      AND EXISTS (
          SELECT 1 FROM columns c
          WHERE c.id = t.column_id AND c.system_type <> 'completed'
      )
      AND abs(hashtext(t.id::text || 'soon049')) % 6 = 0;
END $$;

-- ── 2. Cycle time PORTAL: ~60% в 5–7 дн., остальные — выбросы ─────────────────
DO $$
DECLARE
  -- 29 в ядре 5.0–7.0 + 19 выбросов, перемешаны
    cycle_perm DOUBLE PRECISION[] := ARRAY[
        6.2, 2.1, 5.5, 9.2, 6.0, 3.4, 5.8, 8.1,
        1.8, 6.5, 5.2, 7.0, 6.8, 2.8, 5.0, 6.3,
        7.5, 5.6, 3.0, 9.5, 6.1, 5.4, 6.7, 2.4,
        5.9, 6.4, 7.2, 5.3, 6.9, 4.2, 6.6, 5.7,
        7.8, 6.0, 2.0, 8.8, 5.1, 6.2, 7.0, 3.6,
        9.0, 5.8, 4.8, 10.2, 7.3, 5.5, 8.5, 3.2
    ];
    task_rec     RECORD;
    p_task_id    UUID;
    done_ts      TIMESTAMPTZ;
    cycle_days   DOUBLE PRECISION;
    jitter       DOUBLE PRECISION;
    work_enter   TIMESTAMPTZ;
    review_enter TIMESTAMPTZ;
    ready_enter  TIMESTAMPTZ;
    todo_enter   TIMESTAMPTZ;
    idx          INT := 0;
    pre_days     INT;
    cfd_start    TIMESTAMPTZ := TIMESTAMPTZ '2026-04-18 00:00:00+03';
BEGIN
    FOREACH p_task_id IN ARRAY ARRAY[
        'e1000000-0000-0000-0002-000000000049'::uuid,
        'e1000000-0000-0000-0002-000000000050'::uuid,
        'e1000000-0000-0000-0002-000000000048'::uuid
    ] LOOP
        SELECT MAX(h.entered_at) INTO done_ts
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE h.task_id = p_task_id AND c.system_type = 'completed';

        IF p_task_id = 'e1000000-0000-0000-0002-000000000048'::uuid THEN
            cycle_days := 3.0;
        ELSIF p_task_id = 'e1000000-0000-0000-0002-000000000049'::uuid THEN
            cycle_days := 6.2;
        ELSE
            cycle_days := 6.8;
        END IF;

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
        ready_enter := work_enter - INTERVAL '2 days';
        todo_enter := GREATEST(ready_enter - INTERVAL '3 days', cfd_start - INTERVAL '20 days');

        UPDATE task_status_history SET entered_at = todo_enter, left_at = ready_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;
        UPDATE task_status_history SET entered_at = ready_enter, left_at = work_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000003'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000004'::uuid;
    END LOOP;

    FOR task_rec IN
        SELECT t.id
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
          AND c.system_type = 'completed'
          AND t.id NOT IN (
              'e1000000-0000-0000-0002-000000000048',
              'e1000000-0000-0000-0002-000000000049',
              'e1000000-0000-0000-0002-000000000050'
          )
        ORDER BY (regexp_replace(t.key, '\D', '', 'g'))::int
    LOOP
        idx := idx + 1;
        jitter := ((abs(hashtext(task_rec.id::text || '-j49')) % 5) - 2) * 0.15;
        cycle_days := cycle_perm[idx] + jitter;
        IF cycle_days < 1.0 THEN cycle_days := 1.0; END IF;

        SELECT MAX(h.entered_at) INTO done_ts
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE h.task_id = task_rec.id AND c.system_type = 'completed';

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
        ready_enter := work_enter - make_interval(days => (1 + (abs(hashtext(task_rec.id::text)) % 2))::int);
        pre_days := 2 + (abs(hashtext(task_rec.id::text || '-p49')) % 3);
        todo_enter := GREATEST(ready_enter - make_interval(days => pre_days::int), cfd_start - INTERVAL '20 days');

        UPDATE task_status_history SET entered_at = todo_enter, left_at = ready_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;
        UPDATE task_status_history SET entered_at = ready_enter, left_at = work_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000003'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000004'::uuid;
    END LOOP;
END $$;

-- ── 3. Блокировки: 2 на MOBAPP + 2 на PORTAL (активные задачи) ─────────────────
DELETE FROM task_dependencies
WHERE task_id IN (
    SELECT id FROM tasks
    WHERE project_id IN (
        'c0000000-0000-0000-0001-000000000000'::uuid,
        'c0000000-0000-0000-0002-000000000000'::uuid
    )
);

INSERT INTO task_dependencies (id, task_id, depends_on_task_id, dependency_type) VALUES
    ('f0490000-0000-0000-0001-000000000001'::uuid,
     'e1000000-0000-0000-0001-000000000126'::uuid,
     'e1000000-0000-0000-0001-000000000125'::uuid, 'is_blocked_by'),
    ('f0490000-0000-0000-0001-000000000002'::uuid,
     'e1000000-0000-0000-0001-000000000129'::uuid,
     'e1000000-0000-0000-0001-000000000128'::uuid, 'is_blocked_by'),
    ('f0490000-0000-0000-0002-000000000001'::uuid,
     'e1000000-0000-0000-0002-000000000075'::uuid,
     'e1000000-0000-0000-0002-000000000078'::uuid, 'is_blocked_by'),
    ('f0490000-0000-0000-0002-000000000002'::uuid,
     'e1000000-0000-0000-0002-000000000076'::uuid,
     'e1000000-0000-0000-0002-000000000077'::uuid, 'is_blocked_by');
