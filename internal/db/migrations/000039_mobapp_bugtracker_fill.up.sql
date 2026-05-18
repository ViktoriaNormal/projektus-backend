-- =============================================================================
-- Migration 039: дополнение баг-трекера MOBAPP (вторая доска Scrum-проекта)
-- Период: 17.02.2026 – 31.05.2026 (15 недель, ≥12 недель активности).
-- Добавляет 35 багов (MOBAPP-96 … MOBAPP-130) + историю статусов.
-- Не трогает существующие задачи и пользовательские правки.
-- =============================================================================

SET client_encoding = 'UTF8';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM projects WHERE id = 'c0000000-0000-0000-0001-000000000000') THEN
        RAISE NOTICE 'MOBAPP project not found — skipping bug-tracker seed';
        RETURN;
    END IF;
END $$;

-- ======================== Новые баги на баг-трекере ========================

INSERT INTO tasks (
    id, key, project_id, board_id, owner_id, executor_id,
    name, column_id, priority, estimation, created_at, deadline
)
SELECT
    ('e1000000-0000-0000-0001-' || lpad((96 + p.n)::text, 12, '0'))::uuid,
    'MOBAPP-' || (96 + p.n),
    'c0000000-0000-0000-0001-000000000000'::uuid,
    'd0000000-0000-0000-0001-000000000002'::uuid,
    p.owner_id,
    p.executor_id,
    p.bug_name,
    p.column_id,
    p.priority,
    p.estimation::text,
    p.created_at,
    p.deadline
FROM (
    SELECT
        gs.n,
        titles.bug_name,
        -- Распределение по неделям: ~2–3 бага на каждую из 15 недель
        (TIMESTAMP '2026-02-17 09:00:00+03'
            + make_interval(days => ((gs.n / 2) * 7 + (gs.n % 2) * 3)::int)
            + make_interval(hours => (9 + (abs(hashtext(gs.n::text)) % 6))::int)) AS created_at,
        CASE
            WHEN gs.n < 24 THEN 'd0010000-0000-0000-0001-000000000014'::uuid  -- Закрыто
            WHEN gs.n < 30 THEN
                CASE (gs.n % 3)
                    WHEN 0 THEN 'd0010000-0000-0000-0001-000000000012'::uuid  -- В работе
                    WHEN 1 THEN 'd0010000-0000-0000-0001-000000000013'::uuid  -- Тестирование
                    ELSE 'd0010000-0000-0000-0001-000000000012'::uuid
                END
            ELSE
                CASE (gs.n % 3)
                    WHEN 0 THEN 'd0010000-0000-0000-0001-000000000011'::uuid  -- Новые
                    WHEN 1 THEN 'd0010000-0000-0000-0001-000000000011'::uuid
                    ELSE 'd0010000-0000-0000-0001-000000000012'::uuid
                END
        END AS column_id,
        CASE (gs.n % 4)
            WHEN 0 THEN 'Критичный'
            WHEN 1 THEN 'Высокий'
            WHEN 2 THEN 'Средний'
            ELSE 'Низкий'
        END AS priority,
        (2 + (abs(hashtext(gs.n::text || 'est')) % 11))::text AS estimation,
        CASE
            WHEN gs.n % 12 IN (0, 1)  THEN 'c1000000-0000-0000-0001-000000000008'::uuid  -- QA
            WHEN gs.n % 12 IN (2, 3)  THEN 'c1000000-0000-0000-0001-000000000009'::uuid  -- QA
            WHEN gs.n % 12 IN (4, 5)  THEN 'c1000000-0000-0000-0001-000000000002'::uuid
            WHEN gs.n % 12 IN (6, 7)  THEN 'c1000000-0000-0000-0001-000000000003'::uuid
            WHEN gs.n % 12 IN (8, 9)  THEN 'c1000000-0000-0000-0001-000000000004'::uuid
            WHEN gs.n % 12 IN (10, 11) THEN 'c1000000-0000-0000-0001-000000000005'::uuid
            ELSE 'c1000000-0000-0000-0001-000000000006'::uuid
        END AS owner_id,
        CASE (gs.n % 10)
            WHEN 0 THEN 'c1000000-0000-0000-0001-000000000002'::uuid
            WHEN 1 THEN 'c1000000-0000-0000-0001-000000000003'::uuid
            WHEN 2 THEN 'c1000000-0000-0000-0001-000000000004'::uuid
            WHEN 3 THEN 'c1000000-0000-0000-0001-000000000005'::uuid
            WHEN 4 THEN 'c1000000-0000-0000-0001-000000000006'::uuid
            WHEN 5 THEN 'c1000000-0000-0000-0001-000000000007'::uuid
            WHEN 6 THEN 'c1000000-0000-0000-0001-000000000002'::uuid
            WHEN 7 THEN 'c1000000-0000-0000-0001-000000000003'::uuid
            WHEN 8 THEN 'c1000000-0000-0000-0001-000000000004'::uuid
            ELSE 'c1000000-0000-0000-0001-000000000005'::uuid
        END AS executor_id,
        CASE
            WHEN gs.n < 24 THEN
                (TIMESTAMP '2026-02-17 09:00:00+03'
                    + make_interval(days => ((gs.n / 2) * 7 + (gs.n % 2) * 3 + 2 + (gs.n % 4))::int)
                    + make_interval(hours => 18))
            WHEN gs.n < 30 THEN
                (TIMESTAMP '2026-05-01 09:00:00+03'
                    + make_interval(days => ((gs.n - 24) * 2)::int))
            ELSE
                (TIMESTAMP '2026-05-19 09:00:00+03'
                    + make_interval(days => ((gs.n - 30) * 2)::int))
        END AS deadline
    FROM generate_series(0, 34) AS gs(n)
    CROSS JOIN LATERAL (
        SELECT (ARRAY[
            'Крэш при холодном старте на Android 12',
            'Белый экран после ввода PIN-кода',
            'Не отображается клавиатура на экране логина iOS',
            'Ошибка SSL при подключении к API на старых устройствах',
            'Зависание splash-screen более 10 секунд',
            'Неверный формат даты в выписке за февраль',
            'Кнопка «Назад» закрывает приложение вместо навигации',
            'Не работает восстановление пароля по SMS',
            'Дублирование push при сворачивании приложения',
            'Крэш при открытии PDF-выписки',
            'Некорректный баланс в виджете после перевода СБП',
            'Пропадает маска ввода номера карты',
            'Ошибка 401 при обновлении refresh-токена',
            'Не загружается список счетов при VPN',
            'Крэш при скролле длинной истории операций',
            'Неверная валюта в деталях международного перевода',
            'Не открывается экран деталей вклада',
            'Таймаут при загрузке push-настроек',
            'Съезжает заголовок на экране перевода (iPhone 13 mini)',
            'Не сохраняется шаблон платежа ЖКХ',
            'Крэш при смене языка интерфейса',
            'Некорректная сортировка операций по сумме',
            'Ошибка при экспорте выписки в CSV',
            'Не работает Face ID после переустановки приложения',
            'Медленный отклик кнопки «Оплатить» на Android',
            'Утечка памяти при просмотре графика расходов',
            'Не приходит SMS-код подтверждения перевода',
            'Крэш при открытии чата поддержки (повтор)',
            'Неверный часовой пояс в истории операций',
            'Не обновляется список карт после блокировки',
            'Ошибка валидации суммы перевода с копейками',
            'Зависание при загрузке фото в чат поддержки',
            'Крэш при переключении тёмной темы в настройках',
            'Не отображаются категории в аналитике расходов',
            'Дублирование записей в истории после офлайн-режима',
            'Ошибка 500 при запросе справки о доходах',
            'Некорректный статус карты после разблокировки'
        ])[gs.n + 1] AS bug_name
    ) AS titles
) AS p
WHERE NOT EXISTS (
    SELECT 1 FROM tasks t
    WHERE t.id = ('e1000000-0000-0000-0001-' || lpad((96 + p.n)::text, 12, '0'))::uuid
)
ON CONFLICT (id) DO NOTHING;

-- ======================== Теги «Баг» (если есть на доске) ========================

INSERT INTO task_tags (task_id, tag_id)
SELECT t.id, 'aa000000-0000-0000-0001-000000000005'::uuid
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.id >= 'e1000000-0000-0000-0001-000000000096'
  AND t.id <= 'e1000000-0000-0000-0001-000000000130'
  AND EXISTS (SELECT 1 FROM tags WHERE id = 'aa000000-0000-0000-0001-000000000005')
ON CONFLICT DO NOTHING;

-- ======================== История статусов (только для багов без истории) ========================

-- Закрытые: Новые → В работе → Тестирование → Закрыто
INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000011',
    t.created_at,
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 6))::int)
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000014'
  AND NOT EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id);

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000012',
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 6))::int),
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 6))::int)
                 + make_interval(hours => (4 + (abs(hashtext(t.id::text || 'fix')) % 40))::int)
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000014'
  AND EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id AND h.column_id = 'd0010000-0000-0000-0001-000000000011');

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000013',
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 6))::int)
                 + make_interval(hours => (4 + (abs(hashtext(t.id::text || 'fix')) % 40))::int),
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 6))::int)
                 + make_interval(hours => (4 + (abs(hashtext(t.id::text || 'fix')) % 40))::int)
                 + make_interval(hours => (2 + (abs(hashtext(t.id::text || 'test')) % 20))::int)
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000014'
  AND EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id AND h.column_id = 'd0010000-0000-0000-0001-000000000012');

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000014',
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 6))::int)
                 + make_interval(hours => (4 + (abs(hashtext(t.id::text || 'fix')) % 40))::int)
                 + make_interval(hours => (2 + (abs(hashtext(t.id::text || 'test')) % 20))::int),
    NULL
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000014'
  AND EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id AND h.column_id = 'd0010000-0000-0000-0001-000000000013');

-- В работе: Новые → В работе
INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000011',
    t.created_at,
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 4))::int)
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000012'
  AND NOT EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id);

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000012',
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 4))::int),
    NULL
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000012'
  AND EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id AND h.column_id = 'd0010000-0000-0000-0001-000000000011');

-- Тестирование: Новые → В работе → Тестирование
INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000011',
    t.created_at,
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 5))::int)
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000013'
  AND NOT EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id);

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000012',
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 5))::int),
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 5))::int)
                 + make_interval(hours => (6 + (abs(hashtext(t.id::text || 'w')) % 30))::int)
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000013'
  AND EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id AND h.column_id = 'd0010000-0000-0000-0001-000000000011');

INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000013',
    t.created_at + make_interval(hours => (1 + (abs(hashtext(t.id::text)) % 5))::int)
                 + make_interval(hours => (6 + (abs(hashtext(t.id::text || 'w')) % 30))::int),
    NULL
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000013'
  AND EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id AND h.column_id = 'd0010000-0000-0000-0001-000000000012');

-- Новые: только колонка «Новые»
INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), t.id, 'd0010000-0000-0000-0001-000000000011',
    t.created_at,
    NULL
FROM tasks t
WHERE t.board_id = 'd0000000-0000-0000-0001-000000000002'
  AND t.column_id = 'd0010000-0000-0000-0001-000000000011'
  AND NOT EXISTS (SELECT 1 FROM task_status_history h WHERE h.task_id = t.id);
