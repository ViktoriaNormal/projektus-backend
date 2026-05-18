-- =============================================================================
-- Migration 045: реалистичная история переходов Kanban PORTAL (CFD, throughput)
-- ref: 18.05.2026, окно CFD = 30 дней (18.04–18.05)
--
-- Для каждой задачи на досках «Основная» и «Дизайн и контент»:
--   полная цепочка по sort_order колонок до текущего column_id;
--   завершения разнесены по апрелю–маю (не «с февраля в Done»);
--   на каждом этапе задача проводит 1–6 дней (детерминированно по id).
-- ID: f4500000-0000-0000-0002-*
-- =============================================================================

SET client_encoding = 'UTF8';

-- Бэкап для отката (down восстанавливает из этой таблицы)
DROP TABLE IF EXISTS migration_045_history_backup;

CREATE TABLE migration_045_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  )
  AND t.deleted_at IS NULL;

DELETE FROM task_status_history h
USING tasks t
WHERE h.task_id = t.id
  AND t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
  AND t.board_id IN (
      'd0000000-0000-0000-0002-000000000001'::uuid,
      'd0000000-0000-0000-0002-000000000002'::uuid
  )
  AND t.deleted_at IS NULL;

CREATE OR REPLACE FUNCTION pg_temp.cfd_stage_duration(p_task UUID, p_stage INT)
RETURNS INTERVAL
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT make_interval(days => (
        CASE p_stage
            WHEN 0 THEN 2 + (abs(hashtext(p_task::text || '-s0')) % 3)
            WHEN 1 THEN 1 + (abs(hashtext(p_task::text || '-s1')) % 2)
            WHEN 2 THEN 3 + (abs(hashtext(p_task::text || '-s2')) % 4)
            WHEN 3 THEN 1 + (abs(hashtext(p_task::text || '-s3')) % 2)
            ELSE 1
        END
    )::int);
$$;

DO $$
DECLARE
    ref_end       TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 12:00:00+03';
    cfd_start     TIMESTAMPTZ := TIMESTAMPTZ '2026-04-18 00:00:00+03';
    spread_start  TIMESTAMPTZ := TIMESTAMPTZ '2026-04-20 09:00:00+03';
    spread_end    TIMESTAMPTZ := TIMESTAMPTZ '2026-05-17 17:00:00+03';

    task_rec      RECORD;
    col_ids       UUID[];
    target_idx    INT;
    col_cnt       INT;
    i             INT;
    seg           INT;
    cur_leave     TIMESTAMPTZ;
    cur_enter     TIMESTAMPTZ;
    dur           INTERVAL;
    done_ts       TIMESTAMPTZ;
    done_rank     INT := 0;
    done_total    INT;
    task_seq      INT := 0;
    hist_id       UUID;
    is_completed  BOOLEAN;
BEGIN
    SELECT COUNT(*)::int
    INTO done_total
    FROM tasks t
    JOIN columns c ON c.id = t.column_id
    WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
      AND t.board_id IN (
          'd0000000-0000-0000-0002-000000000001'::uuid,
          'd0000000-0000-0000-0002-000000000002'::uuid
      )
      AND t.deleted_at IS NULL
      AND c.system_type = 'completed';

    FOR task_rec IN
        SELECT
            t.id,
            t.board_id,
            t.column_id,
            t.created_at,
            c.system_type AS col_system_type
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid
          AND t.board_id IN (
              'd0000000-0000-0000-0002-000000000001'::uuid,
              'd0000000-0000-0000-0002-000000000002'::uuid
          )
          AND t.deleted_at IS NULL
        ORDER BY t.board_id, t.created_at, t.id
    LOOP
        task_seq := task_seq + 1;

        SELECT array_agg(col.id ORDER BY col.sort_order)
        INTO col_ids
        FROM columns col
        WHERE col.board_id = task_rec.board_id;

        col_cnt := coalesce(array_length(col_ids, 1), 0);
        IF col_cnt = 0 THEN
            CONTINUE;
        END IF;

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

        is_completed := (task_rec.col_system_type = 'completed');

        IF is_completed THEN
            done_rank := done_rank + 1;

            IF task_rec.id = 'e1000000-0000-0000-0002-000000000048'::uuid THEN
                done_ts := TIMESTAMPTZ '2026-05-18 11:30:00+03';
            ELSIF task_rec.id = 'e1000000-0000-0000-0002-000000000049'::uuid THEN
                done_ts := TIMESTAMPTZ '2026-04-24 14:00:00+03';
            ELSIF task_rec.id = 'e1000000-0000-0000-0002-000000000050'::uuid THEN
                done_ts := TIMESTAMPTZ '2026-04-30 15:30:00+03';
            ELSIF done_total > 1 THEN
                done_ts := spread_start
                    + (spread_end - spread_start)
                      * ((done_rank - 1)::double precision / (done_total - 1)::double precision);
            ELSE
                done_ts := spread_end;
            END IF;

            cur_leave := done_ts;
        ELSE
            cur_leave := ref_end;
        END IF;

        FOR i IN REVERSE target_idx..0 LOOP
            IF is_completed AND i = target_idx THEN
                cur_enter := cur_leave;
            ELSE
                dur := pg_temp.cfd_stage_duration(task_rec.id, i);
                cur_enter := cur_leave - dur;
            END IF;

            IF i = 0 THEN
                cur_enter := LEAST(cur_enter, task_rec.created_at);
                IF is_completed AND done_ts > cfd_start THEN
                    cur_enter := LEAST(cur_enter, done_ts - INTERVAL '18 days');
                ELSIF NOT is_completed THEN
                    cur_enter := GREATEST(
                        cur_enter,
                        cfd_start + make_interval(days => (abs(hashtext(task_rec.id::text)) % 12)::int)
                    );
                END IF;
                cur_enter := GREATEST(cur_enter, task_rec.created_at);
            END IF;

            seg := task_seq * 10 + (target_idx - i);
            hist_id := ('f4500000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid;

            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (
                hist_id,
                task_rec.id,
                col_ids[i + 1],
                cur_enter,
                CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
            );

            cur_leave := cur_enter;
        END LOOP;
    END LOOP;
END $$;
