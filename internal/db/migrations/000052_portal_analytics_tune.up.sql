-- =============================================================================
-- Migration 052: PORTAL Kanban — throughput, cycle time, WIP
--  - Завершения W10–W21 (8-недельный график: нед. 1–3 не пустые, W17 в 4–8)
--  - Cycle time 1.2–10.4 дн. (без floor GREATEST(5))
--  - WIP ≤10 почти всегда; 5 дней превышения с разными значениями
--  - CFD: сохраняем длинные этапы «Надо сделать» / «Готово к работе» из 051
-- ref: 18.05.2026 | ID: f0520000-0000-0000-0002-*
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_052_history_backup;

CREATE TABLE migration_052_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  )
  AND t.deleted_at IS NULL;

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
    cfd_start    TIMESTAMPTZ := TIMESTAMPTZ '2026-04-19 00:00:00+03';
    ref_end      TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 12:00:00+03';

    -- 47 завершений (без 3 закреплённых): W10–W13 + W14–W21
    week_plan TEXT[] := ARRAY[
        '2026-W10', '2026-W10',
        '2026-W11', '2026-W11',
        '2026-W12', '2026-W12',
        '2026-W13', '2026-W13', '2026-W13',
        '2026-W14', '2026-W14', '2026-W14',
        '2026-W15', '2026-W15', '2026-W15', '2026-W15',
        '2026-W16', '2026-W16', '2026-W16', '2026-W16',
        '2026-W17', '2026-W17', '2026-W17', '2026-W17', '2026-W17',
        '2026-W18', '2026-W18', '2026-W18', '2026-W18', '2026-W18', '2026-W18',
        '2026-W19', '2026-W19', '2026-W19', '2026-W19', '2026-W19', '2026-W19',
        '2026-W20', '2026-W20', '2026-W20', '2026-W20', '2026-W20',
        '2026-W21', '2026-W21', '2026-W21', '2026-W21', '2026-W21'
    ];

    cycle_plan DOUBLE PRECISION[] := ARRAY[
        1.2, 1.5, 1.8, 2.0, 2.3, 2.6, 3.0, 3.4,
        5.0, 5.2, 5.4, 5.5, 5.6, 5.8, 6.0, 6.1,
        6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9,
        7.0, 7.1, 7.2, 7.3, 7.5, 7.6, 7.7, 7.8,
        7.9, 8.0, 8.1, 8.2, 8.3, 8.5, 8.8, 9.1,
        9.3, 9.6, 9.8, 10.0, 10.2, 10.4, 8.6
    ];

    slot           INT;

    task_rec       RECORD;
    p_task_id      UUID;
    done_ts        TIMESTAMPTZ;
    cycle_days     DOUBLE PRECISION;
    work_enter     TIMESTAMPTZ;
    review_enter   TIMESTAMPTZ;
    todo_enter     TIMESTAMPTZ;
    ready_enter    TIMESTAMPTZ;
    idx            INT;

    col_ids        UUID[];
    target_idx     INT;
    col_cnt        INT;
    i              INT;
    cur_leave      TIMESTAMPTZ;
    cur_enter      TIMESTAMPTZ;
    dur            INTERVAL;
    stage_days     INT;
    task_seq       INT := 0;
    seg            INT;

    wip_col_work   UUID := 'd0010000-0000-0000-0002-000000000003'::uuid;
    wip_col_review UUID := 'd0010000-0000-0000-0002-000000000004'::uuid;
BEGIN
    -- ── Основная доска: закреплённые завершённые ─────────────────────────────
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

        SELECT h.entered_at INTO todo_enter
        FROM task_status_history h
        WHERE h.task_id = p_task_id
          AND h.column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;

        SELECT h.entered_at INTO ready_enter
        FROM task_status_history h
        WHERE h.task_id = p_task_id
          AND h.column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
        IF ready_enter IS NULL OR ready_enter >= work_enter THEN
            ready_enter := work_enter - INTERVAL '2 days';
        END IF;
        IF todo_enter IS NULL OR todo_enter >= ready_enter THEN
            todo_enter := ready_enter - INTERVAL '3 days';
        END IF;

        UPDATE task_status_history SET entered_at = todo_enter, left_at = ready_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;
        UPDATE task_status_history SET entered_at = ready_enter, left_at = work_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000003'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000004'::uuid;
        UPDATE task_status_history SET entered_at = done_ts, left_at = NULL
        WHERE task_id = p_task_id AND column_id = 'd0010000-0000-0000-0002-000000000005'::uuid;
    END LOOP;

    -- ── Основная доска: остальные завершённые (throughput + cycle time) ───────
    slot := 0;
    FOR task_rec IN
        SELECT t.id
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
          AND t.deleted_at IS NULL
          AND c.system_type = 'completed'
          AND t.id NOT IN (
              'e1000000-0000-0000-0002-000000000048',
              'e1000000-0000-0000-0002-000000000049',
              'e1000000-0000-0000-0002-000000000050'
          )
        ORDER BY hashtext(t.id::text)
    LOOP
        slot := slot + 1;

        done_ts := pg_temp.iso_week_done_ts(week_plan[slot], slot);
        cycle_days := cycle_plan[slot];

        SELECT h.entered_at INTO todo_enter
        FROM task_status_history h
        WHERE h.task_id = task_rec.id
          AND h.column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;

        SELECT h.entered_at INTO ready_enter
        FROM task_status_history h
        WHERE h.task_id = task_rec.id
          AND h.column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);

        IF ready_enter IS NULL OR ready_enter >= work_enter THEN
            ready_enter := work_enter - make_interval(days => (1 + (abs(hashtext(task_rec.id::text || '-rdy')) % 2))::int);
        END IF;
        IF todo_enter IS NULL OR todo_enter >= ready_enter THEN
            todo_enter := GREATEST(
                ready_enter - make_interval(days => (2 + (abs(hashtext(task_rec.id::text || '-pre')) % 3))::int),
                cfd_start - INTERVAL '5 days'
            );
        END IF;

        UPDATE task_status_history SET entered_at = todo_enter, left_at = ready_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000001'::uuid;
        UPDATE task_status_history SET entered_at = ready_enter, left_at = work_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000002'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000003'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000004'::uuid;
        UPDATE task_status_history SET entered_at = done_ts, left_at = NULL
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000005'::uuid;
    END LOOP;

    -- ── Основная доска: активные — укороченные in_progress этапы ─────────────
    DELETE FROM task_status_history h
    USING tasks t
    JOIN columns c ON c.id = t.column_id
    WHERE h.task_id = t.id
      AND t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
      AND t.deleted_at IS NULL
      AND c.system_type <> 'completed';

    FOR task_rec IN
        SELECT t.id, t.column_id, t.created_at
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
          AND t.deleted_at IS NULL
          AND c.system_type <> 'completed'
        ORDER BY t.created_at, t.id
    LOOP
        task_seq := task_seq + 1;

        SELECT array_agg(col.id ORDER BY col.sort_order)
        INTO col_ids
        FROM columns col
        WHERE col.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid;

        col_cnt := coalesce(array_length(col_ids, 1), 0);
        target_idx := NULL;
        FOR i IN 1..col_cnt LOOP
            IF col_ids[i] = task_rec.column_id THEN
                target_idx := i - 1;
                EXIT;
            END IF;
        END LOOP;
        IF target_idx IS NULL THEN
            CONTINUE;
        END IF;

        cur_leave := ref_end;
        FOR i IN REVERSE target_idx..0 LOOP
            IF i = target_idx AND col_ids[i + 1] IN (wip_col_work, wip_col_review) THEN
                stage_days := 2 + (abs(hashtext(task_rec.id::text || '-wip')) % 3);
            ELSIF i = target_idx THEN
                stage_days := 3 + (abs(hashtext(task_rec.id::text || '-cur')) % 3);
            ELSIF col_ids[i + 1] IN (wip_col_work, wip_col_review) THEN
                stage_days := 2 + (abs(hashtext(task_rec.id::text || '-w' || i::text)) % 2);
            ELSE
                stage_days := 3 + (abs(hashtext(task_rec.id::text || '-s' || i::text)) % 3);
            END IF;
            dur := make_interval(days => stage_days);
            cur_enter := cur_leave - dur;

            IF i = 0 THEN
                cur_enter := GREATEST(
                    cur_enter,
                    cfd_start + make_interval(days => (abs(hashtext(task_rec.id::text)) % 18)::int),
                    task_rec.created_at
                );
            END IF;

            seg := 520 + task_seq * 10 + (target_idx - i);
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (
                ('f0520000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid,
                task_rec.id,
                col_ids[i + 1],
                cur_enter,
                CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
            );
            cur_leave := cur_enter;
        END LOOP;
    END LOOP;

    -- ── 5 дней целевого превышения WIP (разные значения) ─────────────────────
    -- Активные задачи в in_progress: 23.04→11, 02.05→13, 06.05→12, 09.05→14, 14.05→11
    UPDATE task_status_history SET
        entered_at = TIMESTAMPTZ '2026-04-20 09:00:00+03',
        left_at    = TIMESTAMPTZ '2026-04-24 18:00:00+03'
    WHERE task_id = 'e1000000-0000-0000-0002-000000000044'::uuid
      AND column_id = wip_col_review AND left_at IS NULL;

    UPDATE task_status_history SET
        entered_at = TIMESTAMPTZ '2026-04-27 09:00:00+03',
        left_at    = TIMESTAMPTZ '2026-05-03 18:00:00+03'
    WHERE task_id = 'e1000000-0000-0000-0002-000000000046'::uuid
      AND column_id = wip_col_review AND left_at IS NULL;

    UPDATE task_status_history SET
        entered_at = TIMESTAMPTZ '2026-05-01 09:00:00+03',
        left_at    = TIMESTAMPTZ '2026-05-08 18:00:00+03'
    WHERE task_id = 'e1000000-0000-0000-0002-000000000007'::uuid
      AND column_id = wip_col_review AND left_at IS NULL;

    UPDATE task_status_history SET
        entered_at = TIMESTAMPTZ '2026-04-26 09:00:00+03',
        left_at    = TIMESTAMPTZ '2026-05-04 18:00:00+03'
    WHERE task_id = 'e1000000-0000-0000-0002-000000000045'::uuid
      AND column_id = wip_col_work AND left_at IS NULL;

    UPDATE task_status_history SET
        entered_at = TIMESTAMPTZ '2026-05-05 09:00:00+03',
        left_at    = TIMESTAMPTZ '2026-05-12 18:00:00+03'
    WHERE task_id = 'e1000000-0000-0000-0002-000000000086'::uuid
      AND column_id = wip_col_review AND left_at IS NULL;

    -- ── Дизайн-доска: завершённые (W15–W21 + cycle) ──────────────────────────
    idx := 0;
    FOR task_rec IN
        SELECT t.id, t.created_at
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
          AND t.deleted_at IS NULL
          AND c.system_type = 'completed'
        ORDER BY hashtext(t.id::text)
    LOOP
        idx := idx + 1;
        done_ts := pg_temp.iso_week_done_ts(
            (ARRAY['2026-W15','2026-W16','2026-W17','2026-W17','2026-W18','2026-W19','2026-W20','2026-W21'])[idx],
            idx + 200
        );
        cycle_days := (ARRAY[2.0, 5.5, 6.0, 6.5, 6.8, 7.0, 7.5, 9.5])[idx];

        SELECT h.entered_at INTO todo_enter
        FROM task_status_history h
        WHERE h.task_id = task_rec.id
          AND h.column_id = 'd0010000-0000-0000-0002-000000000011'::uuid;

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.4)::int);
        IF todo_enter IS NULL OR todo_enter >= work_enter THEN
            todo_enter := work_enter - INTERVAL '3 days';
        END IF;
        todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '5 days');

        UPDATE task_status_history SET entered_at = todo_enter, left_at = work_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000011'::uuid;
        UPDATE task_status_history SET entered_at = work_enter, left_at = review_enter
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000012'::uuid;
        UPDATE task_status_history SET entered_at = review_enter, left_at = done_ts
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000013'::uuid;
        UPDATE task_status_history SET entered_at = done_ts, left_at = NULL
        WHERE task_id = task_rec.id AND column_id = 'd0010000-0000-0000-0002-000000000014'::uuid;
    END LOOP;

    -- ── Дизайн-доска: активные — укороченные этапы ───────────────────────────
    DELETE FROM task_status_history h
    USING tasks t
    JOIN columns c ON c.id = t.column_id
    WHERE h.task_id = t.id
      AND t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
      AND t.deleted_at IS NULL
      AND c.system_type <> 'completed';

    task_seq := 0;
    FOR task_rec IN
        SELECT t.id, t.column_id, t.created_at
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
          AND t.deleted_at IS NULL
          AND c.system_type <> 'completed'
        ORDER BY t.created_at, t.id
    LOOP
        task_seq := task_seq + 1;

        SELECT array_agg(col.id ORDER BY col.sort_order) INTO col_ids
        FROM columns col WHERE col.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid;

        col_cnt := coalesce(array_length(col_ids, 1), 0);
        target_idx := NULL;
        FOR i IN 1..col_cnt LOOP
            IF col_ids[i] = task_rec.column_id THEN
                target_idx := i - 1;
                EXIT;
            END IF;
        END LOOP;
        IF target_idx IS NULL THEN
            CONTINUE;
        END IF;

        cur_leave := ref_end;
        FOR i IN REVERSE target_idx..0 LOOP
            IF i = target_idx AND col_ids[i + 1] IN (
                'd0010000-0000-0000-0002-000000000012'::uuid,
                'd0010000-0000-0000-0002-000000000013'::uuid
            ) THEN
                stage_days := 2 + (abs(hashtext(task_rec.id::text || '-dw')) % 3);
            ELSIF i = target_idx THEN
                stage_days := 3 + (abs(hashtext(task_rec.id::text || '-dc')) % 2);
            ELSE
                stage_days := 3 + (abs(hashtext(task_rec.id::text || '-ds' || i::text)) % 3);
            END IF;
            dur := make_interval(days => stage_days);
            cur_enter := cur_leave - dur;
            IF i = 0 THEN
                cur_enter := GREATEST(
                    cur_enter,
                    cfd_start + make_interval(days => (abs(hashtext(task_rec.id::text)) % 14)::int),
                    task_rec.created_at
                );
            END IF;

            seg := 920 + task_seq * 10 + (target_idx - i);
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (
                ('f0520000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid,
                task_rec.id, col_ids[i + 1], cur_enter,
                CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
            );
            cur_leave := cur_enter;
        END LOOP;
    END LOOP;
END $$;

-- ── Синхронизация дедлайнов завершённых PORTAL ────────────────────────────────
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
      AND t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
      AND EXISTS (
          SELECT 1 FROM columns c
          WHERE c.id = t.column_id AND c.system_type = 'completed'
      );
END $$;
