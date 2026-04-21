-- =============================================================================
-- Migration 000034: Корпоративные еженедельные встречи + покрытие «≥ 2 встречи
-- на каждого пользователя в каждой неделе периода 02.02.2026 – 03.05.2026».
--
-- Логика:
--   * 13 недель (с 02.02 по 27.04). На каждую неделю генерируется 30 встреч
--     в разных тематиках (гильдии, демо, HR, q&a, клубы, стратегия и т.п.)
--     по 6 тайм-слотов × 5 рабочих дней.
--   * На каждую встречу зовётся 8 участников (ограничение 2..10 соблюдается).
--     Участники распределяются round-robin по 120 сотрудникам так, чтобы
--     за неделю каждому досталось ровно 2 корпоративные встречи
--     (30 × 8 = 240 слотов = 120 × 2).
--   * Админ добавляется отдельно в первые 2 встречи каждой недели
--     (эти встречи получат 9 участников вместо 8).
--   * Организатор каждой встречи — первый приглашённый участник
--     (детерминированно по min(user_id)).
--
-- Эффект: у каждого пользователя не менее 2 встреч на каждую неделю
-- указанного периода (в дополнение к проектным daily/sync/planning и
-- пользовательским встречам из 000032).
-- =============================================================================

SET client_encoding = 'UTF8';

-- ==================== Недели периода ====================

CREATE TEMP TABLE _weeks AS
SELECT
    (DATE '2026-02-02' + (w * 7))::date AS week_start,
    (w + 1) AS week_n
FROM generate_series(0, 12) AS w;  -- 13 недель: 02.02.2026 .. 27.04.2026

-- ==================== Слоты на неделе (30 штук) ====================
-- 6 встреч в день (9:30, 11:00, 13:00, 14:30, 16:00, 17:30) × 5 дней.
-- Длительность: 60 мин для первых 5 слотов дня, 45 мин для последнего.

CREATE TEMP TABLE _corp_slots AS
SELECT
    meeting_idx,
    ((meeting_idx - 1) / 6) AS day_offset,   -- 0..4 = пн..пт
    CASE ((meeting_idx - 1) % 6)
        WHEN 0 THEN  9
        WHEN 1 THEN 11
        WHEN 2 THEN 13
        WHEN 3 THEN 14
        WHEN 4 THEN 16
        ELSE        17
    END AS start_hour,
    CASE ((meeting_idx - 1) % 6)
        WHEN 0 THEN 30
        WHEN 1 THEN  0
        WHEN 2 THEN  0
        WHEN 3 THEN 30
        WHEN 4 THEN  0
        ELSE        30
    END AS start_min,
    CASE ((meeting_idx - 1) % 6) WHEN 5 THEN 45 ELSE 60 END AS duration_min,
    (ARRAY[
        'Понедельничный общий митинг',   'Product review',                'Гильдия Backend',
        'HR open hour',                   'Клуб чтения non-fiction',       'Кофе с руководителями',
        'Демо командных результатов',     'Гильдия QA',                    'Design review',
        'Обмен опытом между командами',   'Мини-лекция от эксперта',       'Разбор инцидентов недели',
        'Безопасность: воркшоп',          'Гильдия Frontend',              'Обратная связь от клиентов',
        'Менторская сессия',              'Рабочая группа DevOps',         'Open-space обсуждение',
        'Kaizen: улучшение процессов',    'Аналитика: ретро метрик',       'Гильдия Mobile',
        'Клубный час',                    'Q&A с CTO',                     'Английский клуб',
        'All-hands компании',             'Клуб код-ревью',                'Стратегическая сессия',
        'Тимбилдинг онлайн',              'TGIF: неформальное общение',    'Дайджест недели'
    ])[meeting_idx] AS title,
    (ARRAY[
        'Старт недели: цели и приоритеты',
        'Обзор продуктовых инициатив',
        'Архитектурные и серверные практики',
        'Вопросы и ответы с HR',
        'Обсуждение книг по soft skills',
        'Неформальное общение с руководством',
        'Презентация результатов команд',
        'Методики тестирования и автоматизация',
        'Обзор макетов и UX-гипотез',
        'Шеринг кейсов между командами',
        'Короткий доклад по актуальной теме',
        'Post-mortem прошедших инцидентов',
        'Разбор уязвимостей и security-практик',
        'Клиентская часть приложений: практики',
        'Выводы из feedback и поддержки',
        'Менторская сессия для сотрудников',
        'Проектные обсуждения по инфраструктуре',
        'Свободная площадка для идей',
        'Обсуждение улучшений процессов',
        'Анализ метрик и качества',
        'Разработка под мобильные платформы',
        'Неформальный обмен опытом',
        'Сессия вопросов и ответов с CTO',
        'Разговорный английский в рабочем контексте',
        'Общее собрание всей компании',
        'Совместный код-ревью и best practices',
        'Стратегические инициативы квартала',
        'Командообразование и игры',
        'Финал недели: неформальные разговоры',
        'Ключевые события недели'
    ])[meeting_idx] AS description
FROM generate_series(1, 30) AS meeting_idx;

-- ==================== Вставка встреч ====================
-- UUID вида a7000000-0000-0000-{WEEK}-{MEETING}. Уникально для каждой (week, meeting).

INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status)
SELECT
    ('a7000000-0000-0000-' || lpad(w.week_n::text, 4, '0') || '-' || lpad(s.meeting_idx::text, 12, '0'))::uuid,
    NULL,
    s.title,
    s.description,
    'sync',
    (w.week_start + make_interval(days => s.day_offset, hours => s.start_hour, mins => s.start_min))::timestamptz,
    (w.week_start + make_interval(days => s.day_offset, hours => s.start_hour, mins => s.start_min + s.duration_min))::timestamptz,
    -- Временно — админ. Ниже перепишем на первого участника встречи.
    'b0000000-0000-0000-ffff-000000000001'::uuid,
    'active'
FROM _weeks w
CROSS JOIN _corp_slots s
ON CONFLICT (id) DO NOTHING;

-- ==================== Участники ====================
-- Round-robin по 120 сотрудникам. За неделю ровно 240 слотов = 120 × 2.
-- Формула: user_n = ((week_n - 1) * 240 + (meeting_idx - 1) * 8 + slot - 1) mod 120 + 1

INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT
    ('a7000000-0000-0000-' || lpad(w.week_n::text, 4, '0') || '-' || lpad(s.meeting_idx::text, 12, '0'))::uuid,
    ('b0000000-0000-0000-0000-' ||
     lpad((((w.week_n - 1) * 240 + (s.meeting_idx - 1) * 8 + slot - 1) % 120 + 1)::text, 12, '0'))::uuid,
    'accepted'
FROM _weeks w
CROSS JOIN _corp_slots s
CROSS JOIN generate_series(1, 8) AS slot
ON CONFLICT (meeting_id, user_id) DO NOTHING;

-- Admin: присутствует в первых двух встречах каждой недели
INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT
    ('a7000000-0000-0000-' || lpad(w.week_n::text, 4, '0') || '-' || lpad(m_idx::text, 12, '0'))::uuid,
    'b0000000-0000-0000-ffff-000000000001'::uuid,
    'accepted'
FROM _weeks w
CROSS JOIN (VALUES (1), (16)) AS v(m_idx)
ON CONFLICT (meeting_id, user_id) DO NOTHING;

-- ==================== Организатор — первый участник встречи ====================

UPDATE meetings m
SET created_by = sub.user_id
FROM (
    SELECT mp.meeting_id,
           (ARRAY_AGG(mp.user_id ORDER BY mp.user_id))[1] AS user_id
    FROM meeting_participants mp
    JOIN meetings m2 ON m2.id = mp.meeting_id
    WHERE m2.id::text LIKE 'a7000000-%'
    GROUP BY mp.meeting_id
) sub
WHERE m.id = sub.meeting_id;

DROP TABLE _corp_slots;
DROP TABLE _weeks;
