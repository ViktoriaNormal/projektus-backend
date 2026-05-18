-- =============================================================================
-- Migration 043: WIP на всех досках MOBAPP/PORTAL + throughput Kanban PORTAL
--  - Все недели (кроме текущей ISO): минимум 2 завершённые задачи
--  - Текущая ISO-неделя: ровно 1 завершённая задача
--  - Больше задач в колонках system_type = in_progress
-- =============================================================================

SET client_encoding = 'UTF8';

-- ── Вспомогательная: ISO-неделя как в Go weekKey() ─────────────────────────
CREATE OR REPLACE FUNCTION pg_temp.iso_week_key(ts TIMESTAMPTZ)
RETURNS TEXT LANGUAGE sql IMMUTABLE AS $$
    SELECT to_char(ts AT TIME ZONE 'Europe/Moscow', 'IYYY')
        || '-W'
        || to_char(ts AT TIME ZONE 'Europe/Moscow', 'IW');
$$;

-- ── Завершение задачи Kanban PORTAL с корректной историей (нужен in_progress) ─
CREATE OR REPLACE PROCEDURE pg_temp.portal_kanban_complete(
    p_task_id UUID,
    p_wip_col UUID,
    p_done_col UUID,
    p_done_ts TIMESTAMPTZ,
    p_wip_hist_id UUID,
    p_done_hist_id UUID
)
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM task_status_history h
        JOIN columns c ON c.id = h.column_id
        WHERE h.task_id = p_task_id
          AND c.system_type IN ('in_progress', 'paused')
    ) THEN
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (
            p_wip_hist_id,
            p_task_id,
            p_wip_col,
            p_done_ts - INTERVAL '4 days',
            p_done_ts - INTERVAL '2 hours'
        )
        ON CONFLICT (id) DO NOTHING;
    END IF;

    UPDATE tasks
    SET column_id = p_done_col
    WHERE id = p_task_id;

    UPDATE task_status_history
    SET left_at = p_done_ts
    WHERE task_id = p_task_id
      AND left_at IS NULL;

    INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
    VALUES (p_done_hist_id, p_task_id, p_done_col, p_done_ts, NULL)
    ON CONFLICT (id) DO NOTHING;
END;
$$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 1. PORTAL Kanban: починить историю у недавних завершений (без in_progress)
-- ═══════════════════════════════════════════════════════════════════════════

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT * FROM (VALUES
    ('f4300000-0000-0000-0002-000000000001'::uuid, 'e1000000-0000-0000-0002-000000000043'::uuid,
     'd0010000-0000-0000-0002-000000000003'::uuid, '2026-05-14 10:00:00+03'::timestamptz, '2026-05-17 08:00:00+03'::timestamptz),
    ('f4300000-0000-0000-0002-000000000002'::uuid, 'e1000000-0000-0000-0002-000000000047'::uuid,
     'd0010000-0000-0000-0002-000000000003'::uuid, '2026-05-13 09:00:00+03'::timestamptz, '2026-05-17 12:30:00+03'::timestamptz),
    ('f4300000-0000-0000-0002-000000000003'::uuid, 'e1000000-0000-0000-0002-000000000048'::uuid,
     'd0010000-0000-0000-0002-000000000004'::uuid, '2026-05-15 11:00:00+03'::timestamptz, '2026-05-18 07:00:00+03'::timestamptz)
) AS v(id, task_id, column_id, entered_at, left_at)
WHERE NOT EXISTS (
    SELECT 1 FROM task_status_history h
    JOIN columns c ON c.id = h.column_id
    WHERE h.task_id = v.task_id
      AND c.system_type IN ('in_progress', 'paused')
)
ON CONFLICT (id) DO NOTHING;

-- Перенос лишних завершений с текущей недели на предыдущую (оставим 1 на W21)
UPDATE task_status_history h
SET entered_at = '2026-05-10 16:00:00+03',
    left_at    = '2026-05-10 16:00:00+03'
WHERE h.id IN (
    SELECT h2.id
    FROM task_status_history h2
    JOIN columns c ON c.id = h2.column_id AND c.system_type = 'completed'
    WHERE h2.task_id = 'e1000000-0000-0000-0002-000000000047'
      AND pg_temp.iso_week_key(h2.entered_at) = pg_temp.iso_week_key(NOW())
      AND h2.left_at IS NULL
    LIMIT 1
);

-- PORTAL-48 — единственное завершение на текущей неделе (18.05.2026)
CALL pg_temp.portal_kanban_complete(
    'e1000000-0000-0000-0002-000000000048',
    'd0010000-0000-0000-0002-000000000004',
    'd0010000-0000-0000-0002-000000000005',
    '2026-05-18 11:30:00+03',
    'f4300000-0000-0000-0002-000000000010',
    'f4300000-0000-0000-0002-000000000011'
);

-- Недели W17 и W18: по 2 завершения (добавляем по одной)
CALL pg_temp.portal_kanban_complete(
    'e1000000-0000-0000-0002-000000000049',
    'd0010000-0000-0000-0002-000000000003',
    'd0010000-0000-0000-0002-000000000005',
    '2026-04-24 14:00:00+03',
    'f4300000-0000-0000-0002-000000000020',
    'f4300000-0000-0000-0002-000000000021'
);

CALL pg_temp.portal_kanban_complete(
    'e1000000-0000-0000-0002-000000000050',
    'd0010000-0000-0000-0002-000000000003',
    'd0010000-0000-0000-0002-000000000005',
    '2026-04-30 15:30:00+03',
    'f4300000-0000-0000-0002-000000000030',
    'f4300000-0000-0000-0002-000000000031'
);

-- Динамически: любая неделя из последних 8 (кроме текущей) с <2 завершениями
DO $$
DECLARE
    wk TEXT;
    d DATE;
    i INT;
    cnt INT;
    cur_wk TEXT := pg_temp.iso_week_key(NOW());
    task_rec RECORD;
    done_ts TIMESTAMPTZ;
    hist_idx INT := 100;
BEGIN
  FOR i IN 0..7 LOOP
    d := (NOW()::date - ((7 - i) * 7));
    wk := pg_temp.iso_week_key(d::timestamptz);

    IF wk = cur_wk THEN
        CONTINUE;
    END IF;

    SELECT COUNT(*) INTO cnt
    FROM (
        SELECT t.id
        FROM tasks t
        JOIN task_status_history h ON h.task_id = t.id
        JOIN columns c ON c.id = h.column_id
        WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'
          AND t.board_id = 'd0000000-0000-0000-0002-000000000001'
          AND t.deleted_at IS NULL
        GROUP BY t.id
        HAVING MIN(CASE WHEN c.system_type IN ('in_progress', 'paused') THEN h.entered_at END) IS NOT NULL
           AND MAX(CASE WHEN c.system_type = 'completed' THEN h.entered_at END) IS NOT NULL
           AND pg_temp.iso_week_key(MAX(CASE WHEN c.system_type = 'completed' THEN h.entered_at END)) = wk
    ) sub;

    WHILE cnt < 2 LOOP
        SELECT t.id INTO task_rec
        FROM tasks t
        JOIN columns col ON col.id = t.column_id
        WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'
          AND t.board_id = 'd0000000-0000-0000-0002-000000000001'
          AND col.system_type = 'initial'
          AND t.deleted_at IS NULL
          AND NOT EXISTS (
              SELECT 1 FROM task_status_history h2
              JOIN columns c2 ON c2.id = h2.column_id
              WHERE h2.task_id = t.id AND c2.system_type = 'completed'
          )
        ORDER BY t.created_at
        LIMIT 1;

        IF task_rec.id IS NULL THEN
            EXIT;
        END IF;

        hist_idx := hist_idx + 1;
        done_ts := d::timestamptz + INTERVAL '1 day' + make_interval(hours => (cnt * 5 + 10)::int);

        CALL pg_temp.portal_kanban_complete(
            task_rec.id,
            'd0010000-0000-0000-0002-000000000003',
            'd0010000-0000-0000-0002-000000000005',
            done_ts,
            ('f4300000-0000-0000-0002-' || lpad(hist_idx::text, 12, '0'))::uuid,
            ('f4300000-0000-0000-0003-' || lpad(hist_idx::text, 12, '0'))::uuid
        );

        cnt := cnt + 1;
    END LOOP;
  END LOOP;
END $$;

-- Текущая неделя: ровно 1 завершение (снять лишние на предыдущую неделю)
DO $$
DECLARE
    cur_wk TEXT := pg_temp.iso_week_key(NOW());
    cnt INT;
    extra RECORD;
    shift_ts TIMESTAMPTZ := (NOW()::date - 8)::timestamptz + INTERVAL '12 hours';
BEGIN
    SELECT COUNT(*) INTO cnt
    FROM (
        SELECT t.id
        FROM tasks t
        JOIN task_status_history h ON h.task_id = t.id
        JOIN columns c ON c.id = h.column_id
        WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'
          AND t.board_id = 'd0000000-0000-0000-0002-000000000001'
          AND t.deleted_at IS NULL
        GROUP BY t.id
        HAVING MIN(CASE WHEN c.system_type IN ('in_progress', 'paused') THEN h.entered_at END) IS NOT NULL
           AND MAX(CASE WHEN c.system_type = 'completed' THEN h.entered_at END) IS NOT NULL
           AND pg_temp.iso_week_key(MAX(CASE WHEN c.system_type = 'completed' THEN h.entered_at END)) = cur_wk
    ) s;

    WHILE cnt > 1 LOOP
        SELECT t.id AS task_id, h.id AS hist_id INTO extra
        FROM tasks t
        JOIN task_status_history h ON h.task_id = t.id
        JOIN columns c ON c.id = h.column_id AND c.system_type = 'completed'
        WHERE t.project_id = 'c0000000-0000-0000-0002-000000000000'
          AND t.board_id = 'd0000000-0000-0000-0002-000000000001'
          AND pg_temp.iso_week_key(h.entered_at) = cur_wk
          AND h.left_at IS NULL
          AND t.id <> 'e1000000-0000-0000-0002-000000000048'
        ORDER BY h.entered_at DESC
        LIMIT 1;

        IF extra.task_id IS NULL THEN
            EXIT;
        END IF;

        UPDATE task_status_history
        SET entered_at = shift_ts,
            left_at    = shift_ts
        WHERE id = extra.hist_id;

        cnt := cnt - 1;
    END LOOP;

    IF cnt = 0 THEN
        CALL pg_temp.portal_kanban_complete(
            'e1000000-0000-0000-0002-000000000045',
            'd0010000-0000-0000-0002-000000000003',
            'd0010000-0000-0000-0002-000000000005',
            (NOW()::date::timestamptz + INTERVAL '11 hours'),
            'f4300000-0000-0000-0002-000000000040',
            'f4300000-0000-0000-0002-000000000041'
        );
    END IF;
END $$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 2. Больше задач в колонках «В работе» (in_progress) на всех досках
-- ═══════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE PROCEDURE pg_temp.move_to_wip(
    p_task_id UUID,
    p_col_id UUID,
    p_hist_id UUID,
    p_ts TIMESTAMPTZ DEFAULT NOW()
)
LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM tasks WHERE id = p_task_id) THEN
        RETURN;
    END IF;

    UPDATE tasks SET column_id = p_col_id WHERE id = p_task_id;

    UPDATE task_status_history
    SET left_at = p_ts
    WHERE task_id = p_task_id AND left_at IS NULL;

    INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
    VALUES (p_hist_id, p_task_id, p_col_id, p_ts, NULL)
    ON CONFLICT (id) DO NOTHING;
END;
$$;

-- MOBAPP — основная доска
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000088', 'd0010000-0000-0000-0001-000000000002', 'f4300000-0000-0000-0001-000000000001', '2026-05-16 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000089', 'd0010000-0000-0000-0001-000000000002', 'f4300000-0000-0000-0001-000000000002', '2026-05-16 11:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000061', 'd0010000-0000-0000-0001-000000000002', 'f4300000-0000-0000-0001-000000000003', '2026-05-15 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000062', 'd0010000-0000-0000-0001-000000000002', 'f4300000-0000-0000-0001-000000000004', '2026-05-15 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000081', 'd0010000-0000-0000-0001-000000000003', 'f4300000-0000-0000-0001-000000000005', '2026-05-17 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000082', 'd0010000-0000-0000-0001-000000000003', 'f4300000-0000-0000-0001-000000000006', '2026-05-17 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000083', 'd0010000-0000-0000-0001-000000000002', 'f4300000-0000-0000-0001-000000000007', '2026-05-17 11:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000084', 'd0010000-0000-0000-0001-000000000002', 'f4300000-0000-0000-0001-000000000008', '2026-05-17 12:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000063', 'd0010000-0000-0000-0001-000000000003', 'f4300000-0000-0000-0001-000000000009', '2026-05-14 14:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000064', 'd0010000-0000-0000-0001-000000000003', 'f4300000-0000-0000-0001-000000000010', '2026-05-14 15:00:00+03');

-- MOBAPP — баг-трекер
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000126', 'd0010000-0000-0000-0001-000000000012', 'f4300000-0000-0000-0001-000000000011', '2026-05-16 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000127', 'd0010000-0000-0000-0001-000000000012', 'f4300000-0000-0000-0001-000000000012', '2026-05-16 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000129', 'd0010000-0000-0000-0001-000000000013', 'f4300000-0000-0000-0001-000000000013', '2026-05-17 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000130', 'd0010000-0000-0000-0001-000000000013', 'f4300000-0000-0000-0001-000000000014', '2026-05-17 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000093', 'd0010000-0000-0000-0001-000000000013', 'f4300000-0000-0000-0001-000000000015', '2026-05-18 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000095', 'd0010000-0000-0000-0001-000000000012', 'f4300000-0000-0000-0001-000000000016', '2026-05-18 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000079', 'd0010000-0000-0000-0001-000000000012', 'f4300000-0000-0000-0001-000000000017', '2026-05-12 11:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0001-000000000080', 'd0010000-0000-0000-0001-000000000013', 'f4300000-0000-0000-0001-000000000018', '2026-05-13 11:00:00+03');

-- PORTAL — основная доска
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000044', 'd0010000-0000-0000-0002-000000000003', 'f4300000-0000-0000-0002-000000000050', '2026-05-16 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000046', 'd0010000-0000-0000-0002-000000000003', 'f4300000-0000-0000-0002-000000000051', '2026-05-16 10:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000007', 'd0010000-0000-0000-0002-000000000003', 'f4300000-0000-0000-0002-000000000053', '2026-05-15 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000008', 'd0010000-0000-0000-0002-000000000004', 'f4300000-0000-0000-0002-000000000054', '2026-05-16 14:00:00+03');

-- PORTAL — дизайн и контент
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000054', 'd0010000-0000-0000-0002-000000000012', 'f4300000-0000-0000-0002-000000000060', '2026-05-10 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000056', 'd0010000-0000-0000-0002-000000000012', 'f4300000-0000-0000-0002-000000000061', '2026-05-11 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000057', 'd0010000-0000-0000-0002-000000000013', 'f4300000-0000-0000-0002-000000000062', '2026-05-12 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000058', 'd0010000-0000-0000-0002-000000000013', 'f4300000-0000-0000-0002-000000000063', '2026-05-13 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000059', 'd0010000-0000-0000-0002-000000000012', 'f4300000-0000-0000-0002-000000000064', '2026-05-14 09:00:00+03');
CALL pg_temp.move_to_wip('e1000000-0000-0000-0002-000000000060', 'd0010000-0000-0000-0002-000000000013', 'f4300000-0000-0000-0002-000000000065', '2026-05-14 10:00:00+03');

DROP PROCEDURE IF EXISTS pg_temp.portal_kanban_complete(UUID, UUID, UUID, TIMESTAMPTZ, UUID, UUID);
DROP PROCEDURE IF EXISTS pg_temp.move_to_wip(UUID, UUID, UUID, TIMESTAMPTZ);
DROP FUNCTION IF EXISTS pg_temp.iso_week_key(TIMESTAMPTZ);
