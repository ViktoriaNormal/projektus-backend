-- =============================================================================
-- Migration 046: PORTAL Kanban — история для полной аналитики
--  - Завершения за 12 ISO-недель (вкл. ранние «Нед 1–3» на графике throughput)
--  - Недели с 4–8 и 8–12 задачами (распределение скорости поставки)
--  - Cycle time 0–2, 2–4, 4–6, 6–8, 8–10 дней (время производства)
--  - Полная цепочка колонок для CFD
-- ref: 18.05.2026 | ID: f4510000-0000-0000-0002-*
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_046_history_backup;

CREATE TABLE migration_046_history_backup AS
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

-- ISO-неделя → метка времени завершения (среда недели + смещение по задаче)
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

-- ── Основная доска: 48 завершённых — неделя + cycle time ───────────────────
DO $$
DECLARE
    -- 12 недель × кол-во завершений (сумма = 48): ранние недели + средние диапазоны
    -- 48 завершений: ранние недели + 4-8 и 8-12 задач/нед. для распределения throughput
    week_plan  TEXT[] := ARRAY[
        '2026-W10',
        '2026-W11',
        '2026-W12', '2026-W12',
        '2026-W13',
        '2026-W14', '2026-W14', '2026-W14', '2026-W14', '2026-W14',
        '2026-W15', '2026-W15', '2026-W15', '2026-W15', '2026-W15',
        '2026-W16', '2026-W16', '2026-W16', '2026-W16', '2026-W16',
        '2026-W17', '2026-W17', '2026-W17', '2026-W17', '2026-W17', '2026-W17',
        '2026-W18', '2026-W18', '2026-W18', '2026-W18', '2026-W18',
        '2026-W18', '2026-W18', '2026-W18',
        '2026-W19', '2026-W19', '2026-W19', '2026-W19', '2026-W19',
        '2026-W19', '2026-W19', '2026-W19', '2026-W19',
        '2026-W20', '2026-W20', '2026-W20', '2026-W20',
        '2026-W21'
    ];
    -- cycle time (дни): 8×0-2, 8×2-4, 10×4-6, 10×6-8, 12×8-10
    cycle_plan DOUBLE PRECISION[] := ARRAY[
        1.2, 1.5, 1.8, 2.0, 1.3, 1.7, 1.9, 2.0,
        2.5, 2.8, 3.0, 3.2, 2.6, 3.1, 3.4, 3.8,
        4.5, 4.8, 5.0, 5.2, 5.4, 4.6, 5.1, 5.3, 5.5, 4.9,
        6.5, 6.8, 7.0, 7.2, 7.4, 6.6, 7.1, 7.3, 7.5, 6.9,
        8.5, 8.8, 9.0, 9.2, 9.4, 8.6, 9.1, 9.3, 9.5, 8.9,
        9.6, 9.8, 9.2, 8.7, 9.4, 9.1,
        3.0
    ];

    task_rec     RECORD;
    task_id      UUID;
    col_ids      UUID[];
    done_ts      TIMESTAMPTZ;
    cycle_days   DOUBLE PRECISION;
    work_enter   TIMESTAMPTZ;
    review_enter TIMESTAMPTZ;
    ready_enter  TIMESTAMPTZ;
    todo_enter   TIMESTAMPTZ;
    pre_days     INT;
    idx          INT;
    seg          INT;
    cfd_start    TIMESTAMPTZ := TIMESTAMPTZ '2026-04-18 00:00:00+03';
    ref_end      TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 12:00:00+03';

    target_idx   INT;
    col_cnt      INT;
    i            INT;
    cur_leave    TIMESTAMPTZ;
    cur_enter    TIMESTAMPTZ;
    dur          INTERVAL;
    task_seq     INT := 0;
BEGIN
  -- Закреплённые задачи (throughput 043)
  FOREACH task_id IN ARRAY ARRAY[
      'e1000000-0000-0000-0002-000000000049'::uuid,
      'e1000000-0000-0000-0002-000000000050'::uuid,
      'e1000000-0000-0000-0002-000000000048'::uuid
  ] LOOP
    SELECT t.id, t.created_at INTO task_rec
    FROM tasks t WHERE t.id = task_id;

    IF task_id = 'e1000000-0000-0000-0002-000000000048'::uuid THEN
        idx := 48; done_ts := TIMESTAMPTZ '2026-05-18 11:30:00+03'; cycle_days := 3.0;
    ELSIF task_id = 'e1000000-0000-0000-0002-000000000049'::uuid THEN
        idx := 27; done_ts := TIMESTAMPTZ '2026-04-24 14:00:00+03'; cycle_days := cycle_plan[27];
    ELSE
        idx := 34; done_ts := TIMESTAMPTZ '2026-04-30 15:30:00+03'; cycle_days := cycle_plan[34];
    END IF;

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
        ready_enter := work_enter - make_interval(days => (1 + (abs(hashtext(task_rec.id::text || '-rdy')) % 2))::int);
        pre_days := 2 + (abs(hashtext(task_rec.id::text || '-pre')) % 3);
        todo_enter := ready_enter - make_interval(days => pre_days::int);
        todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '25 days');

        seg := idx * 10;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at) VALUES
            (('f4510000-0000-0000-0002-' || lpad((seg + 1)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000001'::uuid, todo_enter, ready_enter),
            (('f4510000-0000-0000-0002-' || lpad((seg + 2)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000002'::uuid, ready_enter, work_enter),
            (('f4510000-0000-0000-0002-' || lpad((seg + 3)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000003'::uuid, work_enter, review_enter),
            (('f4510000-0000-0000-0002-' || lpad((seg + 4)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000004'::uuid, review_enter, done_ts),
            (('f4510000-0000-0000-0002-' || lpad((seg + 5)::text, 12, '0'))::uuid,
             task_rec.id, 'd0010000-0000-0000-0002-000000000005'::uuid, done_ts, NULL);
  END LOOP;

  -- Остальные 45 завершённых — слоты 1..47 без 27, 34, 48
  idx := 0;
  FOR task_rec IN
      SELECT t.id, t.created_at
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

      done_ts := pg_temp.iso_week_done_ts(week_plan[idx], idx);
      cycle_days := cycle_plan[idx];

      work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
      review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.35)::int);
      ready_enter := work_enter - make_interval(days => (1 + (abs(hashtext(task_rec.id::text || '-rdy')) % 2))::int);
      pre_days := 2 + (abs(hashtext(task_rec.id::text || '-pre')) % 3);
      todo_enter := ready_enter - make_interval(days => pre_days::int);
      todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '25 days');

      seg := idx * 10;
      INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at) VALUES
          (('f4510000-0000-0000-0002-' || lpad((seg + 1)::text, 12, '0'))::uuid,
           task_rec.id, 'd0010000-0000-0000-0002-000000000001'::uuid, todo_enter, ready_enter),
          (('f4510000-0000-0000-0002-' || lpad((seg + 2)::text, 12, '0'))::uuid,
           task_rec.id, 'd0010000-0000-0000-0002-000000000002'::uuid, ready_enter, work_enter),
          (('f4510000-0000-0000-0002-' || lpad((seg + 3)::text, 12, '0'))::uuid,
           task_rec.id, 'd0010000-0000-0000-0002-000000000003'::uuid, work_enter, review_enter),
          (('f4510000-0000-0000-0002-' || lpad((seg + 4)::text, 12, '0'))::uuid,
           task_rec.id, 'd0010000-0000-0000-0002-000000000004'::uuid, review_enter, done_ts),
          (('f4510000-0000-0000-0002-' || lpad((seg + 5)::text, 12, '0'))::uuid,
           task_rec.id, 'd0010000-0000-0000-0002-000000000005'::uuid, done_ts, NULL);
  END LOOP;

    -- ── Незавершённые: основная доска ───────────────────────────────────────
    FOR task_rec IN
        SELECT t.id, t.column_id, t.created_at, c.system_type AS col_system_type
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
            dur := make_interval(days => (1 + (abs(hashtext(task_rec.id::text || '-s' || i::text)) % 3))::int);
            cur_enter := cur_leave - dur;
            IF i = 0 THEN
                cur_enter := GREATEST(
                    LEAST(cur_enter, task_rec.created_at),
                    cfd_start + make_interval(days => (abs(hashtext(task_rec.id::text)) % 10)::int),
                    task_rec.created_at
                );
            END IF;

            seg := 500 + task_seq * 10 + (target_idx - i);
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (
                ('f4510000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid,
                task_rec.id,
                col_ids[i + 1],
                cur_enter,
                CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
            );
            cur_leave := cur_enter;
        END LOOP;
    END LOOP;

    -- ── Дизайн-доска: 8 завершённых + 20 в работе ────────────────────────────
    idx := 0;
    FOR task_rec IN
        SELECT t.id, t.column_id, t.created_at, c.system_type AS col_system_type
        FROM tasks t
        JOIN columns c ON c.id = t.column_id
        WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
          AND t.deleted_at IS NULL
        ORDER BY (c.system_type = 'completed') DESC, t.created_at, t.id
    LOOP
        IF task_rec.col_system_type = 'completed' THEN
            idx := idx + 1;
            done_ts := pg_temp.iso_week_done_ts(
                (ARRAY['2026-W12','2026-W13','2026-W14','2026-W15','2026-W16','2026-W17','2026-W19','2026-W20'])[idx],
                idx + 100
            );
            cycle_days := (ARRAY[2.0, 3.5, 5.0, 6.5, 4.0, 7.0, 8.5, 9.0])[idx];

            work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
            review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.4)::int);
            ready_enter := work_enter - INTERVAL '2 days';
            todo_enter := GREATEST(ready_enter - INTERVAL '3 days', task_rec.created_at);

            seg := 800 + idx * 10;
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at) VALUES
                (('f4510000-0000-0000-0002-' || lpad((seg + 1)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000011'::uuid, todo_enter, ready_enter),
                (('f4510000-0000-0000-0002-' || lpad((seg + 2)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000012'::uuid, ready_enter, work_enter),
                (('f4510000-0000-0000-0002-' || lpad((seg + 3)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000013'::uuid, work_enter, review_enter),
                (('f4510000-0000-0000-0002-' || lpad((seg + 4)::text, 12, '0'))::uuid,
                 task_rec.id, 'd0010000-0000-0000-0002-000000000014'::uuid, done_ts, NULL);
        ELSE
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
                dur := make_interval(days => (1 + (abs(hashtext(task_rec.id::text || '-d' || i::text)) % 2))::int);
                cur_enter := cur_leave - dur;
                IF i = 0 THEN
                    cur_enter := GREATEST(cur_enter, task_rec.created_at);
                END IF;
                seg := 900 + task_seq * 10 + (target_idx - i);
                INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
                VALUES (
                    ('f4510000-0000-0000-0002-' || lpad(seg::text, 12, '0'))::uuid,
                    task_rec.id, col_ids[i + 1], cur_enter,
                    CASE WHEN i < target_idx THEN cur_leave ELSE NULL END
                );
                cur_leave := cur_enter;
            END LOOP;
        END IF;
    END LOOP;
END $$;
