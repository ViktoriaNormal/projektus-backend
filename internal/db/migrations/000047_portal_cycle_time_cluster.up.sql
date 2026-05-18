-- =============================================================================
-- Migration 047: PORTAL Kanban — кластер cycle time ~6 дн. для диаграммы рассеяния
-- Большинство задач: 5–8 дней; часть — короткие (1–3) и длинные (9–10) выбросы.
-- Недели завершения (throughput) не меняются — только entered_at in_progress.
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_047_history_backup;

CREATE TABLE migration_047_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  )
  AND t.deleted_at IS NULL;

DO $$
DECLARE
    -- 48 значений: 8 коротких + 32 в ядре 5–8 + 8 длинных
    cycle_plan DOUBLE PRECISION[] := ARRAY[
        1.2, 1.5, 1.8, 2.0, 2.3, 2.6, 3.0, 3.4,
        5.0, 5.2, 5.4, 5.5, 5.6, 5.8, 6.0, 6.0,
        6.1, 6.2, 6.3, 6.4, 6.5, 6.5, 6.6, 6.7,
        6.8, 6.9, 7.0, 7.1, 7.2, 7.3, 7.5, 7.6,
        7.7, 7.8, 7.9, 8.0, 8.0, 8.1, 8.2, 8.3,
        8.8, 9.1, 9.3, 9.6, 9.8, 10.0, 10.2, 10.4
    ];

    task_rec     RECORD;
    p_task_id    UUID;
    done_ts      TIMESTAMPTZ;
    cycle_days   DOUBLE PRECISION;
    work_enter   TIMESTAMPTZ;
    review_enter TIMESTAMPTZ;
    ready_enter  TIMESTAMPTZ;
    todo_enter   TIMESTAMPTZ;
    idx          INT;
    pre_days     INT;
    cfd_start    TIMESTAMPTZ := TIMESTAMPTZ '2026-04-18 00:00:00+03';
BEGIN
    -- Закреплённые cycle time (недели завершения не трогаем)
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
            cycle_days := 7.0;
        END IF;

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
        ready_enter := work_enter - make_interval(days => (1 + (abs(hashtext(p_task_id::text || '-rdy')) % 2))::int);
        pre_days := 2 + (abs(hashtext(p_task_id::text || '-pre')) % 3);
        todo_enter := GREATEST(ready_enter - make_interval(days => pre_days::int), cfd_start - INTERVAL '20 days');

        UPDATE task_status_history SET entered_at = todo_enter, left_at = ready_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;
        UPDATE task_status_history SET entered_at = ready_enter, left_at = work_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000003'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000004'::uuid;
    END LOOP;

    idx := 0;
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
        ORDER BY hashtext(t.id::text)
    LOOP
        LOOP
            idx := idx + 1;
            EXIT WHEN idx NOT IN (27, 34, 48);
        END LOOP;

        cycle_days := cycle_plan[idx];

        SELECT MAX(h.entered_at) INTO done_ts
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE h.task_id = task_rec.id AND c.system_type = 'completed';

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
        ready_enter := work_enter - make_interval(days => (1 + (abs(hashtext(task_rec.id::text || '-rdy')) % 2))::int);
        pre_days := 2 + (abs(hashtext(task_rec.id::text || '-pre')) % 3);
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

    -- Дизайн-доска: 8 завершённых — тоже кластер ~6 дн. + 2 выброса
    idx := 0;
    FOR task_rec IN
        SELECT t.id
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
          AND c.system_type = 'completed'
        ORDER BY hashtext(t.id::text)
    LOOP
        idx := idx + 1;
        cycle_days := (ARRAY[2.0, 5.5, 6.0, 6.5, 6.8, 7.0, 7.5, 9.5])[idx];

        SELECT MAX(h.entered_at) INTO done_ts
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE h.task_id = task_rec.id AND c.system_type = 'completed';

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.4)::int);
        ready_enter := work_enter - INTERVAL '2 days';
        todo_enter := ready_enter - INTERVAL '2 days';

        UPDATE task_status_history SET entered_at = todo_enter, left_at = ready_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000011'::uuid;
        UPDATE task_status_history SET entered_at = ready_enter, left_at = work_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000012'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000013'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000014'::uuid;
    END LOOP;
END $$;
