-- =============================================================================
-- Migration 054: PORTAL «Дизайн и контент» — плотность аналитики
--  - +24 задачи, ~44 завершения (3–4/нед. за 12 нед.), cycle 3–30 дн. с хвостом
--  - 6 задач в WIP (В дизайне / На согласовании) для графика WIP Age
-- ID: f0540000-0000-0000-0002-*
-- =============================================================================

SET client_encoding = 'UTF8';

DROP TABLE IF EXISTS migration_054_history_backup;
DROP TABLE IF EXISTS migration_054_tasks_backup;
DROP TABLE IF EXISTS migration_054_new_tasks;

CREATE TABLE migration_054_history_backup AS
SELECT h.id, h.task_id, h.column_id, h.entered_at, h.left_at
FROM task_status_history h
JOIN tasks t ON t.id = h.task_id
WHERE t.board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
  AND t.deleted_at IS NULL;

CREATE TABLE migration_054_tasks_backup AS
SELECT id, column_id, deadline
FROM tasks
WHERE board_id = 'd0000000-0000-0000-0002-000000000002'::uuid
  AND deleted_at IS NULL;

CREATE TABLE migration_054_new_tasks (id UUID PRIMARY KEY);

-- ── 24 новые задачи (PORTAL-D01 … D24) ───────────────────────────────────────
INSERT INTO tasks (id, key, project_id, board_id, owner_id, executor_id, name, column_id, priority, estimation, created_at, deadline)
SELECT
    ('e1000000-0000-0000-0002-' || lpad(to_hex(512 + gs.n), 12, '0'))::uuid,
    'PORTAL-DC' || lpad(gs.n::text, 2, '0'),
    'c0000000-0000-0000-0002-000000000000'::uuid,
    'd0000000-0000-0000-0002-000000000002'::uuid,
    'c1000000-0000-0000-0002-000000000001'::uuid,
    CASE WHEN gs.n % 3 = 0 THEN 'c1000000-0000-0000-0002-000000000011'::uuid
         ELSE 'c1000000-0000-0000-0002-000000000010'::uuid END,
    (ARRAY[
        'Макеты страницы «Новости компании»',
        'Иконки для модуля уведомлений',
        'Контент: описание бенефитов',
        'UI паттерны для форм',
        'Иллюстрации для welcome-тура',
        'Макеты календаря отпусков',
        'Редизайн карточки сотрудника',
        'Гайд по использованию фото',
        'Баннеры для HR-акций',
        'Макеты страницы «Документы»',
        'Контент: шаблоны писем',
        'Иконки статусов задач',
        'Макеты виджета опросов',
        'Иллюстрации для 404/500',
        'Дизайн чек-листа онбординга',
        'Макеты фильтров каталога',
        'Контент: подсказки в интерфейсе',
        'UI-kit для модальных окон',
        'Макеты страницы «Моя команда»',
        'Иконки для раздела аналитики',
        'Контент: тексты пуш-уведомлений',
        'Макеты мобильного меню',
        'Иллюстрации для опросов NPS',
        'Редизайн блока быстрых ссылок'
    ])[gs.n],
    'd0010000-0000-0000-0002-000000000014'::uuid,
    CASE WHEN gs.n % 4 = 0 THEN 'Высокий' WHEN gs.n % 4 = 1 THEN 'Средний' ELSE 'Низкий' END,
    (2 + (gs.n % 6))::text,
    TIMESTAMPTZ '2026-04-01 09:00:00+03' + make_interval(days => (gs.n % 20)),
    TIMESTAMPTZ '2026-06-01 18:00:00+03' + make_interval(days => (gs.n % 14))
FROM generate_series(1, 24) AS gs(n)
ON CONFLICT (id) DO NOTHING;

INSERT INTO migration_054_new_tasks (id)
SELECT ('e1000000-0000-0000-0002-' || lpad(to_hex(512 + gs.n), 12, '0'))::uuid
FROM generate_series(1, 24) AS gs(n);

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
    ref_end        TIMESTAMPTZ := TIMESTAMPTZ '2026-05-18 12:00:00+03';

    -- 44 завершения: 3–4 задачи на каждую из 12 недель W10–W21
    week_plan TEXT[] := ARRAY[
        '2026-W10','2026-W10','2026-W10','2026-W10',
        '2026-W11','2026-W11','2026-W11','2026-W11',
        '2026-W12','2026-W12','2026-W12','2026-W12',
        '2026-W13','2026-W13','2026-W13',
        '2026-W14','2026-W14','2026-W14','2026-W14',
        '2026-W15','2026-W15','2026-W15','2026-W15',
        '2026-W16','2026-W16','2026-W16',
        '2026-W17','2026-W17','2026-W17','2026-W17',
        '2026-W18','2026-W18','2026-W18','2026-W18',
        '2026-W19','2026-W19','2026-W19',
        '2026-W20','2026-W20','2026-W20','2026-W20',
        '2026-W21','2026-W21','2026-W21','2026-W21','2026-W21'
    ];
    cycle_plan DOUBLE PRECISION[] := ARRAY[
        3.0, 4.0, 5.0, 5.5, 6.0, 6.5, 7.0, 7.5,
        4.5, 5.5, 6.5, 7.5, 8.0, 8.5, 9.0, 9.5,
        5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0,
        6.0, 7.0, 8.5, 10.5, 12.5, 14.0, 16.0, 18.0,
        22.0, 24.0, 25.0, 26.0, 28.0, 30.0,
        4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.5
    ];

    wip_ids UUID[] := ARRAY[
        'e1000000-0000-0000-0002-000000000071'::uuid,
        'e1000000-0000-0000-0002-000000000072'::uuid,
        'e1000000-0000-0000-0002-000000000092'::uuid,
        'e1000000-0000-0000-0002-000000000090'::uuid,
        'e1000000-0000-0000-0002-000000000091'::uuid,
        'e1000000-0000-0000-0002-000000000093'::uuid
    ];
    wip_cols UUID[] := ARRAY[
        col_work, col_work, col_work,
        col_review, col_review, col_review
    ];

    task_rec       RECORD;
    wip_rec        RECORD;
    idx            INT;
    wip_i          INT;
    done_ts        TIMESTAMPTZ;
    cycle_days     DOUBLE PRECISION;
    todo_enter     TIMESTAMPTZ;
    work_enter     TIMESTAMPTZ;
    review_enter   TIMESTAMPTZ;
    ideas_days     INT;
    hist_id        UUID;
    wip_age_days   INT;
    cur_enter      TIMESTAMPTZ;
    cur_leave      TIMESTAMPTZ;
BEGIN
    -- Все, кроме 6 WIP, → «Опубликовано»
    UPDATE tasks t
    SET column_id = col_done
    WHERE t.board_id = design_board
      AND t.deleted_at IS NULL
      AND t.id <> ALL(wip_ids);

    -- WIP: колонки «В дизайне» / «На согласовании»
    FOR wip_i IN 1..array_length(wip_ids, 1) LOOP
        UPDATE tasks SET column_id = wip_cols[wip_i]
        WHERE id = wip_ids[wip_i];
    END LOOP;

    -- Завершённые: полная история
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
            RAISE EXCEPTION '054: more completed (%) than plan slots (%)', idx, array_length(week_plan, 1);
        END IF;

        done_ts := pg_temp.iso_week_done_ts(week_plan[idx], idx + 400);
        cycle_days := cycle_plan[idx];

        work_enter := done_ts - make_interval(hours => (cycle_days * 24)::int);
        review_enter := done_ts - make_interval(hours => (cycle_days * 24 * 0.4)::int);
        ideas_days := 2 + (abs(hashtext(task_rec.id::text || '-ideas')) % 4);
        todo_enter := work_enter - make_interval(days => ideas_days);
        todo_enter := GREATEST(todo_enter, task_rec.created_at, cfd_start - INTERVAL '5 days');

        DELETE FROM task_status_history WHERE task_id = task_rec.id;

        hist_id := ('f0540000-0000-0000-0002-' || lpad((idx * 10 + 1)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_ideas, todo_enter, work_enter);

        hist_id := ('f0540000-0000-0000-0002-' || lpad((idx * 10 + 2)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_work, work_enter, review_enter);

        hist_id := ('f0540000-0000-0000-0002-' || lpad((idx * 10 + 3)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_review, review_enter, done_ts);

        hist_id := ('f0540000-0000-0000-0002-' || lpad((idx * 10 + 4)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, task_rec.id, col_done, done_ts, NULL);
    END LOOP;

    IF idx < 40 OR idx > array_length(week_plan, 1) THEN
        RAISE EXCEPTION '054: expected 40–% completed, got %', array_length(week_plan, 1), idx;
    END IF;

    -- WIP: цепочка до текущей колонки (возраст 4–14 дн.)
    FOR wip_i IN 1..array_length(wip_ids, 1) LOOP
        SELECT t.id, t.created_at INTO wip_rec
        FROM tasks t WHERE t.id = wip_ids[wip_i];

        wip_age_days := 4 + (abs(hashtext(wip_rec.id::text || '-age')) % 11);
        cur_leave := ref_end;
        work_enter := cur_leave - make_interval(days => wip_age_days);
        todo_enter := work_enter - make_interval(days => 2 + (abs(hashtext(wip_rec.id::text)) % 3));
        todo_enter := GREATEST(todo_enter, wip_rec.created_at, cfd_start);

        DELETE FROM task_status_history WHERE task_id = wip_rec.id;

        hist_id := ('f0540000-0000-0000-0002-' || lpad((900 + wip_i * 10 + 1)::text, 12, '0'))::uuid;
        INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
        VALUES (hist_id, wip_rec.id, col_ideas, todo_enter, work_enter);

        IF wip_cols[wip_i] = col_work THEN
            hist_id := ('f0540000-0000-0000-0002-' || lpad((900 + wip_i * 10 + 2)::text, 12, '0'))::uuid;
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (hist_id, wip_rec.id, col_work, work_enter, NULL);
        ELSE
            review_enter := work_enter + make_interval(days => 2 + (abs(hashtext(wip_rec.id::text || '-rv')) % 3));
            review_enter := LEAST(review_enter, cur_leave - INTERVAL '1 hour');

            hist_id := ('f0540000-0000-0000-0002-' || lpad((900 + wip_i * 10 + 2)::text, 12, '0'))::uuid;
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (hist_id, wip_rec.id, col_work, work_enter, review_enter);

            hist_id := ('f0540000-0000-0000-0002-' || lpad((900 + wip_i * 10 + 3)::text, 12, '0'))::uuid;
            INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
            VALUES (hist_id, wip_rec.id, col_review, review_enter, NULL);
        END IF;
    END LOOP;
END $$;

-- Дедлайны завершённых
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
