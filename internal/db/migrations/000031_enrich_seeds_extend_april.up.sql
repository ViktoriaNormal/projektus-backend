-- =============================================================================
-- Migration 000031: обогащение сидов + продление периода до 30.04.2026.
--
-- Что делает:
--   1. Нормализует поле `tasks.estimation`: убирает любые нечисловые символы
--      (типа «ч» в Kanban-задачах), превращает пустую строку в NULL.
--      Задачам, у которых после чистки не остаётся значения, но есть
--      исполнитель, проставляется детерминированная оценка 1..13.
--   2. Добавляет CHECK-ограничение на `tasks.estimation`: допустимо только
--      NULL либо строка вида `123` / `12.5` / `2.75` (до 2 знаков после точки).
--   3. Гарантирует, что у каждой «живой» задачи есть `deadline` — если был
--      NULL, то проставляется в диапазоне +3..+17 дней от `created_at`.
--   4. Создаёт для каждой задачи 1–3 чек-листа по 3–5 пунктов. У задач
--      в колонке `completed` все пункты отмечены. У задач «в работе»
--      отмечено ~60% пунктов. У остальных пункты не отмечены.
--   5. Создаёт task_dependencies для ~25% задач (в рамках одного проекта,
--      типы blocks/is_blocked_by/relates_to/subtask — по хэшу).
--   6. Продлевает сидовые встречи (Daily, синки) до 30.04.2026.
--   7. Добавляет ретроспективу спринта 6 (24.04) для MOB и MKT — спринт 6
--      на дату 16.04 ещё активен, но ретро уже запланирована в календаре.
-- =============================================================================

SET client_encoding = 'UTF8';

-- ======================== 1. Нормализация estimation ========================

-- Убираем все нечисловые символы, кроме точки (для десятичных значений).
UPDATE tasks
SET estimation = regexp_replace(estimation, '[^0-9.]', '', 'g')
WHERE estimation IS NOT NULL;

-- Пустая строка после чистки — это отсутствие оценки.
UPDATE tasks
SET estimation = NULL
WHERE estimation = '';

-- Если после чистки осталось что-то вроде «1.», «.5» или «1.2.3», приводим
-- к корректному формату: первая валидная часть, либо NULL.
UPDATE tasks
SET estimation = NULL
WHERE estimation IS NOT NULL AND estimation !~ '^[0-9]+(\.[0-9]{1,2})?$';

-- Задачам с исполнителем, но без оценки — проставляем детерминированное
-- значение 1..13 по хэшу id. Для Scrum-задач это story points,
-- для Kanban — часы.
UPDATE tasks
SET estimation = ((abs(hashtext(id::text)) % 13) + 1)::text
WHERE estimation IS NULL
  AND executor_id IS NOT NULL
  AND deleted_at IS NULL;

-- ======================== 2. DB CHECK на estimation ========================

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_estimation_numeric;
ALTER TABLE tasks ADD CONSTRAINT tasks_estimation_numeric
    CHECK (estimation IS NULL OR estimation ~ '^[0-9]+(\.[0-9]{1,2})?$');

-- ======================== 2a. Исполнитель у всех задач ========================
-- В сидах 000026 ~20% задач остались без исполнителя (executor_id = NULL).
-- По требованию «у задачи всегда должен быть участник-исполнитель из проекта»
-- — проставляем исполнителя из members того же проекта, выбранного
-- детерминированно по хэшу.

UPDATE tasks AS t
SET executor_id = sub.member_id
FROM (
    SELECT t2.id AS task_id,
           (ARRAY_AGG(m.id ORDER BY m.id))[(abs(hashtext(t2.id::text)) % COUNT(*)) + 1] AS member_id
    FROM tasks t2
    JOIN members m ON m.project_id = t2.project_id
    WHERE t2.executor_id IS NULL
      AND t2.deleted_at IS NULL
    GROUP BY t2.id
) sub
WHERE t.id = sub.task_id
  AND t.executor_id IS NULL;

-- ======================== 3. Дедлайны всем задачам ========================
-- У задач без дедлайна — задаём от +3 до +17 дней от created_at, только
-- по рабочим дням (сб/вс сдвигаем на пн).

UPDATE tasks
SET deadline = created_at + make_interval(days => 3 + (abs(hashtext(id::text)) % 15))
WHERE deadline IS NULL
  AND deleted_at IS NULL;

UPDATE tasks SET deadline = deadline + INTERVAL '2 days' WHERE EXTRACT(DOW FROM deadline) = 6;
UPDATE tasks SET deadline = deadline + INTERVAL '1 day'  WHERE EXTRACT(DOW FROM deadline) = 0;

-- ======================== 4. Чек-листы ========================
-- Для каждой «живой» задачи создаём 1..3 чек-листа. В чек-листе 3..5 пунктов.
-- Пункты отмечаются по состоянию задачи в колонке.

-- Временная таблица шаблонов чек-листов (название + 6 возможных пунктов).
CREATE TEMP TABLE _checklist_templates (
    tpl_n INT PRIMARY KEY,
    name TEXT,
    item_1 TEXT, item_2 TEXT, item_3 TEXT, item_4 TEXT, item_5 TEXT
);

INSERT INTO _checklist_templates VALUES
    (1, 'Подготовка',
     'Собрать бизнес-требования',
     'Согласовать ТЗ с продакт-менеджером',
     'Определить критерии приёмки',
     'Оценить трудозатраты с командой',
     'Зафиксировать риски и зависимости'),
    (2, 'Разработка',
     'Реализовать основную логику',
     'Покрыть юнит-тестами',
     'Обновить миграции БД',
     'Пройти код-ревью',
     'Обновить техническую документацию'),
    (3, 'Проверка',
     'Провести ручное тестирование',
     'Прогнать авто-тесты',
     'Проверить на стейдже',
     'Замерить производительность',
     'Проверить edge-кейсы'),
    (4, 'Релиз',
     'Согласовать время выката',
     'Подготовить changelog',
     'Выкатить в production',
     'Проверить post-release метрики',
     'Оповестить заинтересованных'),
    (5, 'Аналитика',
     'Собрать метрики использования',
     'Оценить эффект на целевые KPI',
     'Сформировать итоговый отчёт',
     'Поделиться инсайтами с командой',
     'Запланировать следующие шаги'),
    (6, 'Документация',
     'Описать решение в Confluence',
     'Обновить README проекта',
     'Снять скринкаст для коллег',
     'Добавить примеры в wiki',
     'Проверить корректность API-доков');

-- Собираем пары (task_id, seq=1..cl_count), где cl_count = (hash % 3) + 1.
CREATE TEMP TABLE _task_checklists AS
SELECT
    t.id AS task_id,
    t.column_id,
    col.system_type AS task_system_type,
    seq,
    gen_random_uuid() AS checklist_id,
    ((abs(hashtext(t.id::text || 'tpl' || seq)) % 6) + 1) AS tpl_n,
    ((abs(hashtext(t.id::text || 'items' || seq)) % 3) + 3) AS item_count
FROM tasks t
JOIN columns col ON col.id = t.column_id
CROSS JOIN LATERAL generate_series(1, (abs(hashtext(t.id::text)) % 3) + 1) AS seq
WHERE t.deleted_at IS NULL;

INSERT INTO checklists (id, task_id, name, created_at)
SELECT
    tc.checklist_id,
    tc.task_id,
    tpl.name,
    NOW()
FROM _task_checklists tc
JOIN _checklist_templates tpl ON tpl.tpl_n = tc.tpl_n
ON CONFLICT (id) DO NOTHING;

-- Пункты чек-листа: item_idx от 1 до item_count. Отметка is_checked зависит
-- от состояния задачи: completed → все true, in_progress → ~60% (item_idx <= ceil(0.6 * item_count)),
-- initial → все false.
INSERT INTO checklist_items (id, checklist_id, content, is_checked, sort_order)
SELECT
    gen_random_uuid(),
    tc.checklist_id,
    CASE item_idx
        WHEN 1 THEN tpl.item_1
        WHEN 2 THEN tpl.item_2
        WHEN 3 THEN tpl.item_3
        WHEN 4 THEN tpl.item_4
        ELSE        tpl.item_5
    END,
    CASE
        WHEN tc.task_system_type = 'completed' THEN TRUE
        WHEN tc.task_system_type = 'in_progress'
             AND item_idx <= ceil(tc.item_count * 0.6)::int THEN TRUE
        ELSE FALSE
    END,
    item_idx
FROM _task_checklists tc
JOIN _checklist_templates tpl ON tpl.tpl_n = tc.tpl_n
CROSS JOIN LATERAL generate_series(1, tc.item_count) AS item_idx;

DROP TABLE _task_checklists;
DROP TABLE _checklist_templates;

-- ======================== 5. Task dependencies ========================
-- ~25% задач получают одну зависимость на задачу с близким created_at
-- в том же проекте. Тип зависимости определяется хэшем.

WITH ranked AS (
    SELECT
        t.id,
        t.project_id,
        t.created_at,
        row_number() OVER (PARTITION BY t.project_id ORDER BY t.created_at, t.id) AS rn,
        count(*) OVER (PARTITION BY t.project_id) AS total
    FROM tasks t
    WHERE t.deleted_at IS NULL
),
candidates AS (
    SELECT r1.id AS task_id,
           r2.id AS depends_on_task_id,
           (abs(hashtext(r1.id::text || '-dep')) % 4) AS type_idx
    FROM ranked r1
    JOIN ranked r2
      ON r2.project_id = r1.project_id
     AND r2.rn = ((abs(hashtext(r1.id::text || '-neighbor')) % (r1.total - 1)) + 1)
    WHERE r1.total > 1
      AND (abs(hashtext(r1.id::text || '-roulette')) % 100) < 25
      AND r1.id <> r2.id
)
INSERT INTO task_dependencies (id, task_id, depends_on_task_id, dependency_type)
SELECT
    gen_random_uuid(),
    task_id,
    depends_on_task_id,
    (ARRAY['relates_to','blocks','is_blocked_by','subtask'])[type_idx + 1]
FROM candidates
ON CONFLICT DO NOTHING;

-- ======================== 5a. Наблюдатели задач ========================
-- ~30% задач получают 1–2 наблюдателя из участников своего проекта.
-- Наблюдатели подписываются на уведомления об изменениях задачи.

WITH watch_candidates AS (
    SELECT
        t.id AS task_id,
        t.project_id,
        t.owner_id,
        t.executor_id,
        -- Целевое число наблюдателей: 0 (не попали), 1 или 2.
        CASE
            WHEN (abs(hashtext(t.id::text || 'watch-roulette')) % 100) >= 30 THEN 0
            WHEN (abs(hashtext(t.id::text || 'watch-count'))    % 10)  < 6  THEN 1
            ELSE 2
        END AS watcher_count
    FROM tasks t
    WHERE t.deleted_at IS NULL
),
ranked_members AS (
    SELECT
        m.id     AS member_id,
        m.project_id,
        row_number() OVER (PARTITION BY m.project_id ORDER BY m.id) AS rn,
        count(*) OVER (PARTITION BY m.project_id) AS total
    FROM members m
),
pairs AS (
    -- slot = 1..watcher_count: разные индексы участника по хэшу (не-владелец, не-исполнитель)
    SELECT
        wc.task_id,
        rm.member_id,
        slot
    FROM watch_candidates wc
    CROSS JOIN LATERAL generate_series(1, wc.watcher_count) AS slot
    JOIN ranked_members rm
      ON rm.project_id = wc.project_id
     AND rm.rn = ((abs(hashtext(wc.task_id::text || 'watch-slot-' || slot)) % rm.total) + 1)
    WHERE wc.watcher_count > 0
      AND rm.member_id <> wc.owner_id
      AND rm.member_id IS DISTINCT FROM wc.executor_id
)
INSERT INTO task_watchers (task_id, member_id)
SELECT DISTINCT task_id, member_id FROM pairs
ON CONFLICT (task_id, member_id) DO NOTHING;

-- ======================== 6. Продление встреч до 30.04.2026 ========================
-- Добавляем рабочие дни с 17.04 по 30.04 (17, 20, 21, 22, 23, 24, 27, 28, 29, 30).

CREATE TEMP TABLE _workdays_ext AS
SELECT d::date AS wd
FROM generate_series('2026-04-17'::date, '2026-04-30'::date, '1 day'::interval) d
WHERE EXTRACT(DOW FROM d) BETWEEN 1 AND 5;

-- Daily Scrum MOB — продлеваем, номер встречи = 1000 + индекс,
-- чтобы гарантированно не пересечься с существующими.
INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status)
SELECT
    ('a5001000-0000-0000-0001-' || lpad((1000 + row_number() OVER (ORDER BY wd))::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0001-000000000000'::uuid,
    'Daily Scrum MOB',
    'Ежедневная встреча команды мобильного банка',
    'daily',
    (wd + TIME '10:00')::timestamptz,
    (wd + TIME '10:15')::timestamptz,
    'b0000000-0000-0000-0000-000000000001'::uuid,
    'active'
FROM _workdays_ext
ON CONFLICT (id) DO NOTHING;

-- Daily Scrum MKT
INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status)
SELECT
    ('a5001000-0000-0000-0002-' || lpad((1000 + row_number() OVER (ORDER BY wd))::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0002-000000000000'::uuid,
    'Daily Scrum MKT',
    'Ежедневная встреча команды маркетплейса',
    'daily',
    (wd + TIME '10:30')::timestamptz,
    (wd + TIME '10:45')::timestamptz,
    'b0000000-0000-0000-0000-000000000002'::uuid,
    'active'
FROM _workdays_ext
ON CONFLICT (id) DO NOTHING;

-- Ретроспективы по окончании спринта 6 (24.04.2026, пятница)
INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status) VALUES
    ('a5200000-0000-0000-0001-000000000006', 'c0000000-0000-0000-0001-000000000000',
     'Ретроспектива спринта 6 (MOB)', 'Обсуждение результатов и улучшений', 'retrospective',
     '2026-04-24 16:00:00+03', '2026-04-24 17:00:00+03',
     'b0000000-0000-0000-0000-000000000001'::uuid, 'active'),
    ('a5200000-0000-0000-0002-000000000006', 'c0000000-0000-0000-0002-000000000000',
     'Ретроспектива спринта 6 (MKT)', 'Обсуждение результатов и улучшений', 'retrospective',
     '2026-04-24 17:00:00+03', '2026-04-24 18:00:00+03',
     'b0000000-0000-0000-0000-000000000002'::uuid, 'active')
ON CONFLICT (id) DO NOTHING;

-- Kanban еженедельные синки — продлеваем по средам (22.04, 29.04)
INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status)
SELECT
    ('a5301000-0000-0000-0003-' || lpad((1000 + row_number() OVER (ORDER BY wd))::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0003-000000000000'::uuid,
    'Еженедельный синк INF',
    'Обзор WIP, устранение блокеров',
    'sync',
    (wd + TIME '11:00')::timestamptz,
    (wd + TIME '12:00')::timestamptz,
    'b0000000-0000-0000-0000-000000000003'::uuid,
    'active'
FROM _workdays_ext
WHERE EXTRACT(DOW FROM wd) = 3
ON CONFLICT (id) DO NOTHING;

INSERT INTO meetings (id, project_id, name, description, meeting_type, start_time, end_time, created_by, status)
SELECT
    ('a5301000-0000-0000-0004-' || lpad((1000 + row_number() OVER (ORDER BY wd))::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0004-000000000000'::uuid,
    'Еженедельный синк SITE',
    'Обзор WIP, устранение блокеров',
    'sync',
    (wd + TIME '14:00')::timestamptz,
    (wd + TIME '15:00')::timestamptz,
    'b0000000-0000-0000-0000-000000000004'::uuid,
    'active'
FROM _workdays_ext
WHERE EXTRACT(DOW FROM wd) = 3
ON CONFLICT (id) DO NOTHING;

-- Участники новых встреч — как и раньше, все участники проекта принимают приглашение.
INSERT INTO meeting_participants (meeting_id, user_id, status)
SELECT m.id, mm.user_id, 'accepted'
FROM meetings m
JOIN members mm ON mm.project_id = m.project_id
WHERE m.start_time >= '2026-04-17'
ON CONFLICT (meeting_id, user_id) DO NOTHING;

DROP TABLE _workdays_ext;

-- ======================== 7. Новые задачи на период 17.04–30.04 ========================
-- По 15 задач на каждый проект с датами создания в этот период (рабочие дни).
-- Дедлайны — автоматически из общего UPDATE выше (задачи будут вставлены с NULL,
-- но потом мы пройдёмся повторным апдейтом ниже, раз уж новые записи тоже
-- нуждаются в дедлайне).
--
-- UUID задач: f1000000-0000-0000-000X-...0YY, где X = 1..4 (проект), YY = 001..015.

CREATE TEMP TABLE _new_tasks (
    task_id UUID,
    project_id UUID,
    board_id UUID,
    key TEXT,
    name TEXT,
    owner_member_n INT,
    executor_member_n INT,
    column_id UUID,
    swimlane_id UUID,
    priority TEXT,
    estimation TEXT,
    created_at TIMESTAMPTZ,
    system_type TEXT  -- initial / in_progress / completed
);

-- Рабочие дни 17.04–30.04 (10 дней)
CREATE TEMP TABLE _new_workdays AS
SELECT d::date AS wd, row_number() OVER (ORDER BY d) AS idx
FROM generate_series('2026-04-17'::date, '2026-04-30'::date, '1 day'::interval) d
WHERE EXTRACT(DOW FROM d) BETWEEN 1 AND 5;

-- MOB: 15 задач
INSERT INTO _new_tasks
SELECT
    ('f1000000-0000-0000-0001-' || lpad(n::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0001-000000000000'::uuid,
    'd0000000-0000-0000-0001-000000000000'::uuid,
    'MOB-' || (500 + n),
    (ARRAY[
        'Аналитика: динамика активности пользователей',
        'Оптимизация TTFB главного экрана',
        'Рефакторинг модуля авторизации',
        'Обновление иконок и ассетов темы',
        'Поддержка iOS 18 при сборке',
        'Добавить индикаторы загрузки',
        'Исправить крэш при переключении языков',
        'Улучшить читаемость истории операций',
        'Кейсы биометрии для Android 14',
        'Интеграция с новым SMS-провайдером',
        'Проверить совместимость с iPadOS',
        'Настройка CI для нового флага feature-preview',
        'Мониторинг крэшей через Sentry',
        'Переход на новую версию SDK аналитики',
        'Исследовать WebView для отчётов'
    ])[n],
    ((n * 7) % 30) + 1,
    ((n * 13) % 30) + 1,
    CASE
        WHEN (abs(hashtext(n::text || 'col')) % 10) < 3 THEN 'd0000000-0000-0000-0001-000000000001'::uuid  -- backlog
        WHEN (abs(hashtext(n::text || 'col')) % 10) < 8 THEN 'd0000000-0000-0000-0001-000000000002'::uuid  -- in_progress
        ELSE                                                 'd0000000-0000-0000-0001-000000000003'::uuid  -- на проверке
    END,
    NULL::uuid,
    (ARRAY['Низкий','Средний','Высокий','Критичный'])[((n * 3) % 4) + 1],
    ((n * 2) % 8 + 1)::text,
    ((SELECT wd FROM _new_workdays WHERE idx = ((n - 1) % 10) + 1)::timestamptz
        + make_interval(hours => 9 + (n % 8), mins => ((n * 7) % 60))),
    CASE
        WHEN (abs(hashtext(n::text || 'col')) % 10) < 3 THEN 'initial'
        ELSE                                                 'in_progress'
    END
FROM generate_series(1, 15) AS n;

-- MKT: 15 задач
INSERT INTO _new_tasks
SELECT
    ('f1000000-0000-0000-0002-' || lpad(n::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0002-000000000000'::uuid,
    'd0000000-0000-0000-0002-000000000000'::uuid,
    'MKT-' || (500 + n),
    (ARRAY[
        'Новый дашборд продавца с аналитикой продаж',
        'Фичефлаги для A/B тестов оформления',
        'Интеграция с СДЭК и Почтой России',
        'Массовое обновление цен через CSV',
        'Очереди на обработку заказов в Kafka',
        'Отчёт по возвратам за период',
        'Экспорт остатков в Excel',
        'API для складских систем 3PL',
        'Фильтры по характеристикам товаров',
        'Обновление UI карточки товара',
        'Автоматическая модерация по ML-модели',
        'Поиск по синонимам в каталоге',
        'Интеграция с чат-ботом поддержки',
        'Оптимизация запросов к Elasticsearch',
        'Локализация интерфейса на казахский'
    ])[n],
    ((n * 7) % 35) + 1,
    ((n * 13) % 35) + 1,
    CASE
        WHEN (abs(hashtext(n::text || 'col2')) % 10) < 3 THEN 'd0000000-0000-0000-0002-000000000001'::uuid
        WHEN (abs(hashtext(n::text || 'col2')) % 10) < 8 THEN 'd0000000-0000-0000-0002-000000000002'::uuid
        ELSE                                                  'd0000000-0000-0000-0002-000000000003'::uuid
    END,
    NULL::uuid,
    (ARRAY['Низкий','Средний','Высокий','Критичный'])[((n * 3) % 4) + 1],
    ((n * 2) % 8 + 1)::text,
    ((SELECT wd FROM _new_workdays WHERE idx = ((n - 1) % 10) + 1)::timestamptz
        + make_interval(hours => 10 + (n % 7), mins => ((n * 11) % 60))),
    CASE
        WHEN (abs(hashtext(n::text || 'col2')) % 10) < 3 THEN 'initial'
        ELSE                                                  'in_progress'
    END
FROM generate_series(1, 15) AS n;

-- INF (Kanban): 15 задач
INSERT INTO _new_tasks
SELECT
    ('f1000000-0000-0000-0003-' || lpad(n::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0003-000000000000'::uuid,
    'd0000000-0000-0000-0003-000000000000'::uuid,
    'INF-' || (500 + n),
    (ARRAY[
        'Миграция базы метаданных в отдельный namespace',
        'Обновление версии nginx-ingress до 1.11',
        'Настройка горизонтального масштабирования auth-api',
        'Выделение отдельного кластера для stage',
        'Внедрение OPA для security policies',
        'Аудит секретов в Vault, ротация',
        'Тестирование chaos-engineering сценариев',
        'Оптимизация потребления памяти etcd',
        'Исследование Cilium как CNI',
        'Обновление Helm-чартов до v3.14',
        'Перевод cron-задач в Kubernetes Jobs',
        'Подключение GPU-нод для ML-команды',
        'Настройка подкаста логов в ClickHouse',
        'Безопасность: отключение анонимного доступа',
        'Перенос остатков VM на spot-ноды'
    ])[n],
    ((n * 7) % 25) + 1,
    ((n * 13) % 25) + 1,
    CASE
        WHEN (abs(hashtext(n::text || 'col3')) % 10) < 2 THEN 'd0000000-0000-0000-0003-000000000001'::uuid
        WHEN (abs(hashtext(n::text || 'col3')) % 10) < 4 THEN 'd0000000-0000-0000-0003-000000000002'::uuid
        WHEN (abs(hashtext(n::text || 'col3')) % 10) < 8 THEN 'd0000000-0000-0000-0003-000000000003'::uuid
        ELSE                                                  'd0000000-0000-0000-0003-000000000004'::uuid
    END,
    (ARRAY[
        'd0000000-0000-0000-0003-000000000011',
        'd0000000-0000-0000-0003-000000000012',
        'd0000000-0000-0000-0003-000000000013',
        'd0000000-0000-0000-0003-000000000014'
    ])[((n * 3) % 4) + 1]::uuid,
    (ARRAY['Ускоренный','С фиксированной датой','Стандартный','Нематериальный'])[((n * 3) % 4) + 1],
    ((n * 2) % 6 + 1)::text,
    ((SELECT wd FROM _new_workdays WHERE idx = ((n - 1) % 10) + 1)::timestamptz
        + make_interval(hours => 11 + (n % 6), mins => ((n * 5) % 60))),
    CASE
        WHEN (abs(hashtext(n::text || 'col3')) % 10) < 4 THEN 'initial'
        ELSE                                                  'in_progress'
    END
FROM generate_series(1, 15) AS n;

-- SITE (Kanban): 15 задач
INSERT INTO _new_tasks
SELECT
    ('f1000000-0000-0000-0004-' || lpad(n::text, 12, '0'))::uuid,
    'c0000000-0000-0000-0004-000000000000'::uuid,
    'd0000000-0000-0000-0004-000000000000'::uuid,
    'SITE-' || (500 + n),
    (ARRAY[
        'Обновление страницы вакансий и формы отклика',
        'Дизайн раздела «Партнёрам»',
        'Интеграция с HR-системой Hurma',
        'Оптимизация LCP/INP главной',
        'Переход на Next.js 15',
        'Добавить виджет «Отзывы клиентов»',
        'Новая публикация новости: кейс-стади',
        'Обновление политики конфиденциальности',
        'A/B-тест кнопки «Оставить заявку»',
        'Переработка структуры блога',
        'Локализация на испанский',
        'Переиздание карточек услуг',
        'Интеграция с формой самозаписи в Bitrix24',
        'Удаление устаревших редиректов в NGINX',
        'Обновление фавикона и OG-тегов'
    ])[n],
    ((n * 7) % 28) + 1,
    ((n * 13) % 28) + 1,
    CASE
        WHEN (abs(hashtext(n::text || 'col4')) % 10) < 2 THEN 'd0000000-0000-0000-0004-000000000001'::uuid
        WHEN (abs(hashtext(n::text || 'col4')) % 10) < 4 THEN 'd0000000-0000-0000-0004-000000000002'::uuid
        WHEN (abs(hashtext(n::text || 'col4')) % 10) < 8 THEN 'd0000000-0000-0000-0004-000000000003'::uuid
        ELSE                                                  'd0000000-0000-0000-0004-000000000004'::uuid
    END,
    (ARRAY[
        'd0000000-0000-0000-0004-000000000011',
        'd0000000-0000-0000-0004-000000000012',
        'd0000000-0000-0000-0004-000000000013',
        'd0000000-0000-0000-0004-000000000014'
    ])[((n * 3) % 4) + 1]::uuid,
    (ARRAY['Ускоренный','С фиксированной датой','Стандартный','Нематериальный'])[((n * 3) % 4) + 1],
    ((n * 2) % 6 + 1)::text,
    ((SELECT wd FROM _new_workdays WHERE idx = ((n - 1) % 10) + 1)::timestamptz
        + make_interval(hours => 12 + (n % 5), mins => ((n * 13) % 60))),
    CASE
        WHEN (abs(hashtext(n::text || 'col4')) % 10) < 4 THEN 'initial'
        ELSE                                                  'in_progress'
    END
FROM generate_series(1, 15) AS n;

-- Вставляем новые задачи
INSERT INTO tasks (id, key, project_id, board_id, owner_id, executor_id, name, column_id, swimlane_id, priority, estimation, created_at, deadline)
SELECT
    nt.task_id,
    nt.key,
    nt.project_id,
    nt.board_id,
    ('c1000000-0000-0000-' || lpad(split_part(nt.project_id::text, '-', 4), 4, '0') || '-' || lpad(nt.owner_member_n::text, 12, '0'))::uuid,
    CASE WHEN (abs(hashtext(nt.task_id::text)) % 5) = 0 THEN NULL
         ELSE ('c1000000-0000-0000-' || lpad(split_part(nt.project_id::text, '-', 4), 4, '0') || '-' || lpad(nt.executor_member_n::text, 12, '0'))::uuid
    END,
    nt.name,
    nt.column_id,
    nt.swimlane_id,
    nt.priority,
    nt.estimation,
    nt.created_at,
    nt.created_at + make_interval(days => 3 + (abs(hashtext(nt.task_id::text)) % 15))
FROM _new_tasks nt
ON CONFLICT (id) DO NOTHING;

-- Для Scrum-задач добавляем их в активный спринт 6 (MOB/MKT)
INSERT INTO sprint_tasks (sprint_id, task_id, sort_order)
SELECT
    ('f0000000-0000-0000-' || lpad(split_part(nt.project_id::text, '-', 4), 4, '0') || '-000000000006')::uuid,
    nt.task_id,
    row_number() OVER (PARTITION BY nt.project_id ORDER BY nt.created_at)::int + 1000
FROM _new_tasks nt
WHERE nt.project_id IN (
    'c0000000-0000-0000-0001-000000000000'::uuid,
    'c0000000-0000-0000-0002-000000000000'::uuid
)
ON CONFLICT DO NOTHING;

-- Чек-листы для новых задач (2 листа, 4 пункта каждый)
CREATE TEMP TABLE _new_checklists AS
SELECT
    nt.task_id,
    gen_random_uuid() AS checklist_id,
    seq,
    CASE seq WHEN 1 THEN 'Подготовка' ELSE 'Проверка' END AS name
FROM _new_tasks nt
CROSS JOIN LATERAL generate_series(1, 2) AS seq;

INSERT INTO checklists (id, task_id, name, created_at)
SELECT checklist_id, task_id, name, NOW() FROM _new_checklists
ON CONFLICT (id) DO NOTHING;

INSERT INTO checklist_items (id, checklist_id, content, is_checked, sort_order)
SELECT
    gen_random_uuid(),
    nc.checklist_id,
    CASE nc.name
        WHEN 'Подготовка' THEN (ARRAY[
            'Собрать требования',
            'Согласовать с командой',
            'Определить критерии приёмки',
            'Зафиксировать риски'
        ])[item_idx]
        ELSE (ARRAY[
            'Юнит-тесты проходят',
            'Проверено на стейдже',
            'Пройден код-ревью',
            'Задокументировано в wiki'
        ])[item_idx]
    END,
    CASE
        WHEN nt.system_type = 'completed' THEN TRUE
        WHEN nt.system_type = 'in_progress' AND item_idx <= 2 THEN TRUE
        ELSE FALSE
    END,
    item_idx
FROM _new_checklists nc
JOIN _new_tasks nt ON nt.task_id = nc.task_id
CROSS JOIN LATERAL generate_series(1, 4) AS item_idx;

DROP TABLE _new_checklists;
DROP TABLE _new_tasks;
DROP TABLE _new_workdays;

-- ======================== 8. Финализация: executor + watchers и для новых задач ========================
-- Повторяем те же UPDATE/INSERT, что в секциях 2a и 5a, — теперь они
-- подхватят и свежесозданные задачи из раздела 7. Для старых задач
-- повторный прогон ничего не меняет (обновлять нечего, дубль подсекается
-- ON CONFLICT).

UPDATE tasks AS t
SET executor_id = sub.member_id
FROM (
    SELECT t2.id AS task_id,
           (ARRAY_AGG(m.id ORDER BY m.id))[(abs(hashtext(t2.id::text)) % COUNT(*)) + 1] AS member_id
    FROM tasks t2
    JOIN members m ON m.project_id = t2.project_id
    WHERE t2.executor_id IS NULL
      AND t2.deleted_at IS NULL
    GROUP BY t2.id
) sub
WHERE t.id = sub.task_id
  AND t.executor_id IS NULL;

WITH watch_candidates AS (
    SELECT
        t.id AS task_id,
        t.project_id,
        t.owner_id,
        t.executor_id,
        CASE
            WHEN (abs(hashtext(t.id::text || 'watch-roulette')) % 100) >= 30 THEN 0
            WHEN (abs(hashtext(t.id::text || 'watch-count'))    % 10)  < 6  THEN 1
            ELSE 2
        END AS watcher_count
    FROM tasks t
    WHERE t.deleted_at IS NULL
),
ranked_members AS (
    SELECT
        m.id     AS member_id,
        m.project_id,
        row_number() OVER (PARTITION BY m.project_id ORDER BY m.id) AS rn,
        count(*) OVER (PARTITION BY m.project_id) AS total
    FROM members m
),
pairs AS (
    SELECT
        wc.task_id,
        rm.member_id,
        slot
    FROM watch_candidates wc
    CROSS JOIN LATERAL generate_series(1, wc.watcher_count) AS slot
    JOIN ranked_members rm
      ON rm.project_id = wc.project_id
     AND rm.rn = ((abs(hashtext(wc.task_id::text || 'watch-slot-' || slot)) % rm.total) + 1)
    WHERE wc.watcher_count > 0
      AND rm.member_id <> wc.owner_id
      AND rm.member_id IS DISTINCT FROM wc.executor_id
)
INSERT INTO task_watchers (task_id, member_id)
SELECT DISTINCT task_id, member_id FROM pairs
ON CONFLICT (task_id, member_id) DO NOTHING;
