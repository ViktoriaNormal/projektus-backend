-- =============================================================================
-- Migration 051: PORTAL Kanban — расширение полос CFD
--  - Пересборка task_status_history для основной и дизайн-досок
--  - Длинные этапы (4–8 дн.) и завершения в окне CFD (19.04–18.05)
--  - Сохранение column_id задач и распределения cycle time (~60% в 5–7 дн.)
-- ref: 18.05.2026 | ID: f0510000-0000-0000-0002-*
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_051_history_backup;
DROP TABLE IF EXISTS migration_051_task_done_ts;

CREATE TABLE migration_051_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  )
  AND t.deleted_at IS NULL;

-- Исходные даты завершения (до удаления истории)
CREATE TABLE migration_051_task_done_ts AS
SELECT
    h.task_id,
    MAX(h.entered_at) AS orig_done_ts
FROM migration_051_history_backup h
JOIN columns c ON c.id = h.column_id
WHERE c.system_type = 'completed'
GROUP BY h.task_id;

DELETE FROM task_status_history h
USING tasks t
WHERE h.task_id = t.id
  AND t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  )
  AND t.deleted_at IS NULL;

CREATE OR REPLACE FUNCTION pg_temp.widen_stage_days(
    p_task UUID,
    p_suffix TEXT,
    p_pct DOUBLE PRECISION,
    p_flow_len INT,
    p_min INT
)
RETURNS INT
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT GREATEST(
        p_min,
        GREATEST(1, round(p_flow_len * p_pct))::int
    );
$$;

DO $$
DECLARE
    cfd_start     TIMESTAMPTZ := TIMESTAMPTZ '2026-04-19 00:00:00+03';
    ref_end       TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 12:00:00+03';
    spread_start  TIMESTAMPTZ := TIMESTAMPTZ '2026-04-22 09:00:00+03';
    spread_end    TIMESTAMPTZ := TIMESTAMPTZ '2026-05-16 17:00:00+03';

    cycle_perm DOUBLE PRECISION[] := ARRAY[
        6.2, 2.1, 5.5, 9.2, 6.0, 3.4, 5.8, 8.1,
        1.8, 6.5, 5.2, 7.0, 6.8, 2.8, 5.0, 6.3,
        7.5, 5.6, 3.0, 9.5, 6.1, 5.4, 6.7, 2.4,
        5.9, 6.4, 7.2, 5.3, 6.9, 4.2, 6.6, 5.7,
        7.8, 6.0, 2.0, 8.8, 5.1, 6.2, 7.0, 3.6,
        9.0, 5.8, 4.8, 10.2, 7.3, 5.5, 8.5, 3.2
    ];

    task_rec       RECORD;
    task_id        UUID;
    done_ts        TIMESTAMPTZ;
    orig_done      TIMESTAMPTZ;
    cycle_days     DOUBLE PRECISION;
    jitter         DOUBLE PRECISION;
    flow_len       INT;
    todo_days      INT;
    ready_days     INT;
    work_days      INT;
    review_days    INT;
    todo_enter     TIMESTAMPTZ;
    ready_enter    TIMESTAMPTZ;
    work_enter     TIMESTAMPTZ;
    review_enter   TIMESTAMPTZ;

    remap_rank     INT := 0;
    remap_total    INT;
    idx            INT;
    seg            INT;
    task_seq       INT := 0;

    col_ids        UUID[];
    target_idx     INT;
    col_cnt        INT;
    i              INT;
    cur_leave      TIMESTAMPTZ;
    cur_enter      TIMESTAMPTZ;
    dur            INTERVAL;
    stage_days     INT;
BEGIN
    -- Сколько завершённых нужно сдвинуть в окно CFD
    SELECT COUNT(*)::int INTO remap_total
    FROM tasks t
    JOIN columns c ON c.id = t.column_id
    LEFT JOIN migration_051_task_done_ts d ON d.task_id = t.id
    WHERE t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
      AND t.deleted_at IS NULL
      AND c.system_type = 'completed'
      AND t.id NOT IN (
          'e1000000-0000-0000-0002-000000000048',
          'e1000000-0000-0000-0002-000000000049',
          'e1000000-0000-0000-0002-000000000050'
      )
      AND (d.orig_done_ts IS NULL OR d.orig_done_ts < cfd_start);

    -- ── Основная доска: закреплённые завершённые ─────────────────────────────
    FOREACH task_id IN ARRAY ARRAY[
        'e1000000-0000-0000-0002-000000000049'::uuid,
        'e1000000-0000-0000-0002-000000000050'::uuid,
        'e1000000-0000-0000-0002-000000000048'::uuid
    ] LOOP
        SELECT t.id, t.created_at INTO task_rec FROM tasks t WHERE t.id = task_id;

        IF task_id = 'e1000000-0000-0000-0002-000000000048'::uuid THEN
            idx := 48; done_ts := TIMESTAMPTZ '2026-05-18 11:30:00+03'; cycle_days := 3.0;
        ELSIF task_id = 'e1000000-0000-0000-0002-000000000049'::uuid THEN
            idx := 27; done_ts := TIMESTAMPTZ '2026-04-24 14:00:00+03'; cycle_days := 6.2;
        ELSE
            idx := 34; done_ts := TIMESTAMPTZ '2026-04-30 15:30:00+03'; cycle_days := 6.8;
        END IF;

        flow_len := 14 + (abs(hashtext(task_rec.id::text || '-fl')) % 9);
        work_days := GREATEST(5, round(cycle_days * 0.65)::int);
        review_days := GREATEST(4, round(cycle_days * 0.35)::int);
        work_enter := done_ts - make_interval(days => work_days);
        review_enter := done_ts - make_interval(days => review_days);

        ready_days := pg_temp.widen_stage_days(task_rec.id, '-rdy', 0.20, flow_len, 3);
        todo_days := pg_temp.widen_stage_days(task_rec.id, '-todo', 0.25, flow_len, 4);
        ready_enter := work_enter - make_interval(days => ready_days);
        todo_enter := ready_enter - make_interval(days => todo_days);
        todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '5 days');

        seg := idx * 10;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at) VALUES
            (('f0510000-0000-0000-0002-' || lpad((seg + 1)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000001'::uuid, todo_enter, ready_enter),
            (('f0510000-0000-0000-0002-' || lpad((seg + 2)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000002'::uuid, ready_enter, work_enter),
            (('f0510000-0000-0000-0002-' || lpad((seg + 3)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000003'::uuid, work_enter, review_enter),
            (('f0510000-0000-0000-0002-' || lpad((seg + 4)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000004'::uuid, review_enter, done_ts),
            (('f0510000-0000-0000-0002-' || lpad((seg + 5)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000005'::uuid, done_ts, NULL);
    END LOOP;

    -- ── Основная доска: остальные завершённые ────────────────────────────────
    idx := 0;
    FOR task_rec IN
        SELECT
            t.id,
            t.created_at,
            d.orig_done_ts
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        LEFT JOIN migration_051_task_done_ts d ON d.task_id = t.id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
          AND t.deleted_at IS NULL
          AND c.system_type = 'completed'
          AND t.id NOT IN (
              'e1000000-0000-0000-0002-000000000048',
              'e1000000-0000-0000-0002-000000000049',
              'e1000000-0000-0000-0002-000000000050'
          )
        ORDER BY COALESCE(d.orig_done_ts, t.created_at), hashtext(t.id::text)
    LOOP
        LOOP
            idx := idx + 1;
            EXIT WHEN idx NOT IN (27, 34, 48);
        END LOOP;

        orig_done := task_rec.orig_done_ts;
        IF orig_done IS NULL OR orig_done < cfd_start THEN
            remap_rank := remap_rank + 1;
            IF remap_total > 1 THEN
                done_ts := spread_start
                    + (spread_end - spread_start)
                      * ((remap_rank - 1)::double precision / (remap_total - 1)::double precision);
            ELSE
                done_ts := spread_end;
            END IF;
        ELSE
            done_ts := orig_done;
        END IF;

        jitter := ((abs(hashtext(task_rec.id::text || '-j51')) % 5) - 2) * 0.15;
        cycle_days := cycle_perm[idx] + jitter;
        IF cycle_days < 1.0 THEN cycle_days := 1.0; END IF;

        flow_len := 14 + (abs(hashtext(task_rec.id::text || '-fl')) % 9);
        work_days := GREATEST(5, round(cycle_days * 0.65)::int);
        review_days := GREATEST(4, round(cycle_days * 0.35)::int);
        work_enter := done_ts - make_interval(days => work_days);
        review_enter := done_ts - make_interval(days => review_days);

        ready_days := pg_temp.widen_stage_days(task_rec.id, '-rdy', 0.20, flow_len, 3);
        todo_days := pg_temp.widen_stage_days(task_rec.id, '-todo', 0.25, flow_len, 4);
        ready_enter := work_enter - make_interval(days => ready_days);
        todo_enter := ready_enter - make_interval(days => todo_days);
        todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '5 days');

        seg := idx * 10;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at) VALUES
            (('f0510000-0000-0000-0002-' || lpad((seg + 1)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000001'::uuid, todo_enter, ready_enter),
            (('f0510000-0000-0000-0002-' || lpad((seg + 2)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000002'::uuid, ready_enter, work_enter),
            (('f0510000-0000-0000-0002-' || lpad((seg + 3)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000003'::uuid, work_enter, review_enter),
            (('f0510000-0000-0000-0002-' || lpad((seg + 4)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000004'::uuid, review_enter, done_ts),
            (('f0510000-0000-0000-0002-' || lpad((seg + 5)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000005'::uuid, done_ts, NULL);
    END LOOP;

    -- ── Основная доска: незавершённые (широкие этапы в окне CFD) ─────────────
    FOR task_rec IN
        SELECT t.id, t.column_id, t.created_at
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
          AND t.board_id = 'd0000000-0000-0000-0002-000000000001'::uuid
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
            IF i = target_idx THEN
                stage_days := 5 + (abs(hashtext(task_rec.id::text || '-cur')) % 8);
            ELSE
                stage_days := 4 + (abs(hashtext(task_rec.id::text || '-s' || i::text)) % 5);
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

            seg := 500 + task_seq * 10 + (target_idx - i);
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (
                ('f0510000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid,
                task_rec.id,
                col_ids[i + 1],
                cur_enter,
                CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
            );
            cur_leave := cur_enter;
        END LOOP;
    END LOOP;

    -- ── Дизайн-доска: завершённые + активные ─────────────────────────────────
    idx := 0;
    remap_rank := 0;
    SELECT COUNT(*)::int INTO remap_total
    FROM tasks t
    JOIN columns c ON c.id = t.column_id
    LEFT JOIN migration_051_task_done_ts d ON d.task_id = t.id
    WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
      AND t.deleted_at IS NULL
      AND c.system_type = 'completed'
      AND (d.orig_done_ts IS NULL OR d.orig_done_ts < cfd_start);

    FOR task_rec IN
        SELECT
            t.id,
            t.column_id,
            t.created_at,
            c.system_type AS col_system_type,
            d.orig_done_ts
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        LEFT JOIN migration_051_task_done_ts d ON d.task_id = t.id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
          AND t.deleted_at IS NULL
        ORDER BY (c.system_type = 'completed') DESC,
                 COALESCE(d.orig_done_ts, t.created_at),
                 t.id
    LOOP
        IF task_rec.col_system_type = 'completed' THEN
            idx := idx + 1;

            orig_done := task_rec.orig_done_ts;
            IF orig_done IS NULL OR orig_done < cfd_start THEN
                remap_rank := remap_rank + 1;
                IF remap_total > 1 THEN
                    done_ts := spread_start
                        + (spread_end - spread_start)
                          * ((remap_rank - 1)::double precision / (remap_total - 1)::double precision);
                ELSE
                    done_ts := spread_end;
                END IF;
            ELSE
                done_ts := orig_done;
            END IF;

            flow_len := 10 + (abs(hashtext(task_rec.id::text || '-dfl')) % 7);
            cycle_days := (ARRAY[4.0, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5, 8.0])[1 + ((idx - 1) % 8)];

            work_days := GREATEST(4, round(cycle_days * 0.55)::int);
            review_days := GREATEST(3, round(cycle_days * 0.45)::int);
            work_enter := done_ts - make_interval(days => work_days);
            review_enter := done_ts - make_interval(days => review_days);

            ready_days := pg_temp.widen_stage_days(task_rec.id, '-didea', 0.35, flow_len, 3);
            todo_enter := work_enter - make_interval(days => ready_days);
            todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '5 days');

            seg := 800 + idx * 10;
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at) VALUES
                (('f0510000-0000-0000-0002-' || lpad((seg + 1)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000011'::uuid, todo_enter, work_enter),
                (('f0510000-0000-0000-0002-' || lpad((seg + 2)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000012'::uuid, work_enter, review_enter),
                (('f0510000-0000-0000-0002-' || lpad((seg + 3)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000013'::uuid, review_enter, done_ts),
                (('f0510000-0000-0000-0002-' || lpad((seg + 4)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000014'::uuid, done_ts, NULL);
        ELSE
            task_seq := task_seq + 1;

            SELECT array_agg(col.id ORDER BY col.sort_order) INTO col_ids
            FROM columns col
            WHERE col.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid;

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
                IF i = target_idx THEN
                    stage_days := 4 + (abs(hashtext(task_rec.id::text || '-dcur')) % 6);
                ELSE
                    stage_days := 3 + (abs(hashtext(task_rec.id::text || '-d' || i::text)) % 5);
                END IF;
                dur := make_interval(days => stage_days);
                cur_enter := cur_leave - dur;

                IF i = 0 THEN
                    cur_enter := GREATEST(
                        cur_enter,
                        cfd_start + make_interval(days => (abs(hashtext(task_rec.id::text)) % 12)::int),
                        task_rec.created_at
                    );
                END IF;

                seg := 900 + task_seq * 10 + (target_idx - i);
                INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
                VALUES (
                    ('f0510000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid,
                    task_rec.id,
                    col_ids[i + 1],
                    cur_enter,
                    CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
                );
                cur_leave := cur_enter;
            END LOOP;
        END IF;
    END LOOP;
END $$;

-- ── Синхронизация дедлайнов завершённых PORTAL (логика 049) ───────────────────
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
      AND t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
      AND EXISTS (
          SELECT 1 FROM columns c
          WHERE c.id = t.column_id AND c.system_type = 'completed'
      );
END $$;

DROP TABLE IF EXISTS migration_051_task_done_ts;
