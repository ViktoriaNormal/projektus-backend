-- =============================================================================
-- Migration 000032: коррекция сидов — дедлайны и состав встреч.
--
-- 1. Уменьшение числа просроченных задач.
--    Условный «сегодня» в seed-наборе — 2026-04-16. После миграции 000031
--    задачи 000026 в разных статусах получили дедлайны по формуле
--    created_at + 3..17 дней, и у большей части незавершённых (не-completed)
--    задач дедлайн уже оказался в прошлом. Мы оставляем такое поведение
--    только у 10% таких задач (для иллюстративности «просрочки на доске»),
--    остальным сдвигаем дедлайн в будущее 2026-04-21..2026-05-31.
--    Completed-задачи не трогаем — у них просрочки не может быть по смыслу.
--
-- 2. Состав участников проектных встреч: 2..10 человек.
--    Сейчас каждая встреча имеет всех членов соответствующего проекта
--    (до 35 человек). Под требование «не более 10» — пересоздаём состав:
--    для каждой встречи детерминированно выбираем 4..8 участников
--    (ротация по хэшу id встречи). Организатор (created_by) всегда входит
--    в состав отдельным INSERT'ом с ON CONFLICT DO NOTHING.
--
-- 3. Пользовательские встречи.
--    Каждую неделю с 02.02 по 30.04.2026 — по 2 сквозные (не привязанные
--    к проекту) встречи в разных тематиках: 1:1, обмен опытом, карьерные
--    разговоры, Q&A с HR, тимбилдинги и т.п. Участники распределяются
--    round-robin по всем 120 сотрудникам (5..7 человек на встречу),
--    админ добавляется вручную на одну из встреч. Организатор = первый
--    участник. Это гарантирует, что каждый пользователь является
--    участником как минимум одной встречи.
-- =============================================================================

SET client_encoding = 'UTF8';

-- ==================== 1. Смягчение просроченных дедлайнов ====================

UPDATE tasks t
SET deadline = CASE
    WHEN (abs(hashtext(t.id::text || 'overdue')) % 100) < 10
        -- 10% оставляем просроченными в диапазоне 2026-04-10..2026-04-15
        THEN (DATE '2026-04-10' + make_interval(days => (abs(hashtext(t.id::text || 'pastday')) % 6)))::timestamptz
             + make_interval(hours => 10 + (abs(hashtext(t.id::text)) % 7))
    -- 90% — сдвигаем в будущее: 2026-04-21..2026-05-31 (40 рабочих дней)
    ELSE (DATE '2026-04-21' + make_interval(days => (abs(hashtext(t.id::text || 'futureday')) % 41)))::timestamptz
         + make_interval(hours => 10 + (abs(hashtext(t.id::text)) % 7))
END
FROM columns c
WHERE t.column_id = c.id
  AND c.system_type <> 'completed'
  AND t.deadline IS NOT NULL
  AND t.deadline < DATE '2026-04-21'
  AND t.deleted_at IS NULL;

-- Перенос дедлайнов-выходных на понедельник.
UPDATE tasks SET deadline = deadline + INTERVAL '2 days'
WHERE deleted_at IS NULL AND EXTRACT(DOW FROM deadline) = 6;
UPDATE tasks SET deadline = deadline + INTERVAL '1 day'
WHERE deleted_at IS NULL AND EXTRACT(DOW FROM deadline) = 0;

-- ==================== 2. Состав участников проектных встреч ====================

-- Сначала полностью очищаем существующие приглашения проектных встреч.
DELETE FROM meeting_participants mp
USING meetings m
WHERE mp.meeting_id = m.id
  AND m.project_id IS NOT NULL;

-- Новая подвыборка участников: 4..8 членов проекта детерминированно по хэшу.
INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT DISTINCT m.id, picked.user_id, 'accepted'
FROM meetings m
CROSS JOIN LATERAL (
    SELECT mm.user_id
    FROM members mm
    WHERE mm.project_id = m.project_id
    ORDER BY abs(hashtext(mm.id::text || m.id::text))
    LIMIT ((abs(hashtext(m.id::text || 'size')) % 5) + 4)   -- 4..8 участников
) picked
WHERE m.project_id IS NOT NULL;

-- Гарантируем, что организатор присутствует. После этого шага состав
-- встречи = 4..8 случайных членов + организатор = 4..9 уникальных
-- (max 10 не превышаем), либо 4..8, если организатор уже вошёл.
INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT m.id, m.created_by, 'accepted'
FROM meetings m
WHERE m.project_id IS NOT NULL
ON CONFLICT (meeting_id, user_id) DO NOTHING;

-- ==================== 3. Пользовательские (кросс-проектные) встречи ====================

-- Таблица «недель периода»: с 02.02.2026 до 30.04.2026.
CREATE TEMP TABLE _weeks AS
SELECT
    (DATE '2026-02-02' + (w * 7))::date AS week_start,
    (w + 1) AS week_n
FROM generate_series(0, 12) AS w;  -- 13 недель (2 фев – 4 мая, покрывает весь период)

-- Шаблоны тем для пользовательских встреч
CREATE TEMP TABLE _user_meeting_templates (
    slot INT,
    title TEXT,
    description TEXT,
    offset_day INT,     -- день недели (0..4 = пн..пт)
    start_hour INT,
    duration_min INT
);

INSERT INTO _user_meeting_templates VALUES
    (1, '1:1 с руководителем',              'Индивидуальный разговор о целях и развитии',        1, 14, 45),
    (2, 'Обмен опытом',                      'Неформальная встреча для шеринга знаний',           3, 17, 60),
    (3, 'Карьерный разговор',                'Обсуждение карьерных планов и роста',               2, 15, 45),
    (4, 'Корпоративный Q&A с HR',            'Ответы на вопросы сотрудников от HR',               4, 16, 60),
    (5, 'Тимбилдинг онлайн',                  'Командообразующая активность',                      0, 18, 60),
    (6, 'Внутренний митап: технологии',       'Доклады и обсуждение технологических трендов',      2, 17, 90),
    (7, 'Стратегическая сессия',              'Планирование кросс-командных инициатив',            1, 16, 60);

-- По 2 пользовательские встречи на каждую из 13 недель = 26 встреч.
-- UUID: a6000000-0000-0000-0000-00000000KKKK, где KKKK = (week_n-1)*2 + slot_idx.
WITH planned AS (
    SELECT
        w.week_n,
        w.week_start,
        slot_idx,
        (((w.week_n - 1) * 2 + slot_idx - 1) % 7) + 1 AS tpl_slot,
        (w.week_n - 1) * 2 + slot_idx AS meeting_idx
    FROM _weeks w
    CROSS JOIN generate_series(1, 2) AS slot_idx
)
INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status)
SELECT
    ('a6000000-0000-0000-0000-' || lpad(p.meeting_idx::text, 12, '0'))::uuid,
    NULL,  -- личная / кросс-проектная встреча без привязки к проекту
    tpl.title,
    tpl.description,
    'sync',
    (p.week_start + make_interval(days => tpl.offset_day) + make_interval(hours => tpl.start_hour))::timestamptz,
    (p.week_start + make_interval(days => tpl.offset_day) + make_interval(hours => tpl.start_hour) + make_interval(mins => tpl.duration_min))::timestamptz,
    -- Организатора подставим ниже (первым участником). До этого — временно ставим admin.
    'b0000000-0000-0000-ffff-000000000001'::uuid,
    'active'
FROM planned p
JOIN _user_meeting_templates tpl ON tpl.slot = p.tpl_slot
ON CONFLICT (id) DO NOTHING;

-- Участники: round-robin по 120 сотрудникам.
-- На каждую встречу (meeting_idx 1..26) выбираем 5..7 участников подряд
-- из списка 1..120 (по модулю), чтобы за серию встреч пройти всех.
CREATE TEMP TABLE _user_meeting_plan AS
SELECT
    meeting_idx,
    ((abs(hashtext('umeet-' || meeting_idx::text)) % 3) + 5) AS participant_count
FROM generate_series(1, 26) AS meeting_idx;

INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT
    ('a6000000-0000-0000-0000-' || lpad(p.meeting_idx::text, 12, '0'))::uuid,
    ('b0000000-0000-0000-0000-' || lpad((((p.meeting_idx - 1) * p.participant_count + k - 1) % 120 + 1)::text, 12, '0'))::uuid,
    'accepted'
FROM _user_meeting_plan p
CROSS JOIN LATERAL generate_series(1, p.participant_count) AS k
ON CONFLICT (meeting_id, user_id) DO NOTHING;

-- Назначаем организатором пользовательской встречи её первого участника
-- (user_id с наименьшим id среди приглашённых — детерминированно).
UPDATE meetings m
SET created_by = sub.user_id
FROM (
    SELECT mp.meeting_id,
           (ARRAY_AGG(mp.user_id ORDER BY mp.user_id))[1] AS user_id
    FROM meeting_participants mp
    JOIN meetings m2 ON m2.id = mp.meeting_id
    WHERE m2.project_id IS NULL
      AND m2.id::text LIKE 'a6000000-%'
    GROUP BY mp.meeting_id
) sub
WHERE m.id = sub.meeting_id;

-- Админ гарантированно попадает на первую пользовательскую встречу.
INSERT INTO meeting_participants (meeting_id, user_id, status) VALUES
    ('a6000000-0000-0000-0000-000000000001'::uuid,
     'b0000000-0000-0000-ffff-000000000001'::uuid, 'accepted')
ON CONFLICT (meeting_id, user_id) DO NOTHING;

-- «Спасательный» круг: если после round-robin кто-то из пользователей
-- всё равно остался без единой встречи, равномерно раздаём их по
-- пользовательским встречам с местами <10.
WITH orphans AS (
    SELECT u.id AS user_id,
           row_number() OVER (ORDER BY u.id) AS rn
    FROM users u
    LEFT JOIN meeting_participants mp ON mp.user_id = u.id
    WHERE mp.user_id IS NULL
),
target_meetings AS (
    SELECT m.id,
           row_number() OVER (ORDER BY m.id) AS mn,
           count(*) OVER () AS total_meetings
    FROM meetings m
    LEFT JOIN (
        SELECT meeting_id, count(*) AS cnt
        FROM meeting_participants
        GROUP BY meeting_id
    ) mpc ON mpc.meeting_id = m.id
    WHERE m.project_id IS NULL
      AND m.id::text LIKE 'a6000000-%'
      AND COALESCE(mpc.cnt, 0) < 10
)
INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT tm.id, o.user_id, 'accepted'
FROM orphans o
JOIN target_meetings tm
  ON tm.mn = ((o.rn - 1) % GREATEST(tm.total_meetings, 1)) + 1
ON CONFLICT (meeting_id, user_id) DO NOTHING;

DROP TABLE _user_meeting_plan;
DROP TABLE _user_meeting_templates;
DROP TABLE _weeks;
