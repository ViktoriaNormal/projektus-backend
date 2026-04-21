-- =============================================================================
-- Migration 000033: индивидуальные настройки уведомлений + непрочитанные
-- уведомления каждому пользователю.
--
-- 1. notification_settings: у каждого пользователя детерминированно
--    отключаем 2 из 8 событий (оставляем 6 включёнными, `in_system=false`
--    ровно у двух типов). У 5 «корпоративных» пользователей (admin и
--    руководители проектов employee001..004) дополнительно включаем
--    `in_email=true` для всех типов.
--
-- 2. notifications: для каждого (user_id, event_type) с `in_system=true`
--    создаём по одному непрочитанному уведомлению с осмысленными title,
--    body и payload. Ссылки на реальные сущности (задачу или встречу,
--    связанную с пользователем — как автор/исполнитель/наблюдатель/участник);
--    если пользователь не связан ни с одной — берём произвольную задачу
--    или встречу, чтобы payload остался валидным.
--
-- Итого на каждого пользователя — 6 непрочитанных уведомлений разных типов,
-- все 8 event_type задействованы по компании.
-- =============================================================================

SET client_encoding = 'UTF8';

-- ==================== 1. Индивидуальные настройки ====================

-- Каждому пользователю детерминированно отключаем 2 из 8 типов (rn = 1..2).
WITH ranked AS (
    SELECT
        user_id,
        event_type,
        row_number() OVER (
            PARTITION BY user_id
            ORDER BY abs(hashtext(user_id::text || event_type))
        ) AS rn
    FROM notification_settings
)
UPDATE notification_settings ns
SET in_system = CASE WHEN r.rn <= 2 THEN false ELSE true END
FROM ranked r
WHERE r.user_id = ns.user_id
  AND r.event_type = ns.event_type;

-- У admin и четырёх руководителей проектов (employee001..004) дополнительно
-- включаем email-уведомления для всех типов.
UPDATE notification_settings
SET in_email = true
WHERE user_id IN (
    'b0000000-0000-0000-ffff-000000000001'::uuid,  -- admin
    'b0000000-0000-0000-0000-000000000001'::uuid,  -- руководитель MOB
    'b0000000-0000-0000-0000-000000000002'::uuid,  -- руководитель MKT
    'b0000000-0000-0000-0000-000000000003'::uuid,  -- руководитель INF
    'b0000000-0000-0000-0000-000000000004'::uuid   -- руководитель SITE
);

-- ==================== 2. Кандидаты task/meeting на пользователя ====================

-- Задача, связанная с пользователем (автор, исполнитель или наблюдатель).
-- Если таких нет — fallback на произвольную задачу из БД.
CREATE TEMP TABLE _user_task AS
WITH related AS (
    SELECT DISTINCT ON (u.id)
        u.id AS user_id,
        t.id AS task_id,
        t.key AS task_key,
        t.name AS task_name
    FROM users u
    JOIN members m ON m.user_id = u.id
    JOIN tasks t ON t.project_id = m.project_id
        AND (
            t.owner_id = m.id
            OR t.executor_id = m.id
            OR EXISTS (
                SELECT 1 FROM task_watchers tw
                WHERE tw.task_id = t.id AND tw.member_id = m.id
            )
        )
    WHERE t.deleted_at IS NULL
      AND u.deleted_at IS NULL
    ORDER BY u.id, abs(hashtext(u.id::text || t.id::text))
),
orphans AS (
    SELECT u.id AS user_id
    FROM users u
    LEFT JOIN related r ON r.user_id = u.id
    WHERE u.deleted_at IS NULL
      AND r.user_id IS NULL
),
fallback AS (
    SELECT
        o.user_id,
        t.id AS task_id,
        t.key AS task_key,
        t.name AS task_name
    FROM orphans o
    CROSS JOIN LATERAL (
        SELECT id, key, name
        FROM tasks
        WHERE deleted_at IS NULL
        ORDER BY abs(hashtext(o.user_id::text || id::text))
        LIMIT 1
    ) t
)
SELECT * FROM related
UNION ALL
SELECT * FROM fallback;

-- Встреча, в которой пользователь — участник; fallback на произвольную встречу.
CREATE TEMP TABLE _user_meeting AS
WITH related AS (
    SELECT DISTINCT ON (u.id)
        u.id AS user_id,
        mt.id AS meeting_id,
        mt.name AS meeting_name,
        mt.start_time AS meeting_start_time
    FROM users u
    JOIN meeting_participants mp ON mp.user_id = u.id
    JOIN meetings mt ON mt.id = mp.meeting_id
    WHERE u.deleted_at IS NULL
    ORDER BY u.id, abs(hashtext(u.id::text || mt.id::text))
),
orphans AS (
    SELECT u.id AS user_id
    FROM users u
    LEFT JOIN related r ON r.user_id = u.id
    WHERE u.deleted_at IS NULL
      AND r.user_id IS NULL
),
fallback AS (
    SELECT
        o.user_id,
        m.id AS meeting_id,
        m.name AS meeting_name,
        m.start_time AS meeting_start_time
    FROM orphans o
    CROSS JOIN LATERAL (
        SELECT id, name, start_time
        FROM meetings
        ORDER BY abs(hashtext(o.user_id::text || id::text))
        LIMIT 1
    ) m
)
SELECT * FROM related
UNION ALL
SELECT * FROM fallback;

-- ==================== 3. Создание уведомлений ====================
-- Делаем один INSERT с CROSS JOIN по типам. Фильтруем по notification_settings.in_system.
-- created_at — случайный момент в последние 7 дней относительно 16.04.2026.

WITH event_specs AS (
    SELECT * FROM (VALUES
        ('task_assigned',               'task',    'Вам назначена задача',
            'Задача %s: %s — ожидает вашего внимания.'),
        ('comment_mention',             'task',    'Вас упомянули в комментарии',
            'В обсуждении задачи %s: %s вас упомянули — посмотрите детали.'),
        ('task_status_change_author',   'task',    'Статус вашей задачи изменён',
            'Задача %s: %s перемещена в новую колонку.'),
        ('task_status_change_assignee', 'task',    'Статус задачи изменён',
            'Назначенная вам задача %s: %s изменила статус.'),
        ('task_status_change_watcher',  'task',    'Обновление наблюдаемой задачи',
            'Статус задачи %s: %s, за которой вы следите, изменился.'),
        ('meeting_invite',              'meeting', 'Приглашение на встречу',
            'Вы приглашены на встречу «%s».'),
        ('meeting_change',              'meeting', 'Изменена встреча',
            'Параметры встречи «%s» обновлены — проверьте время и место.'),
        ('meeting_cancel',              'meeting', 'Отменена встреча',
            'Встреча «%s» отменена.')
    ) AS x(event_type, target_kind, title, body_tpl)
)
INSERT INTO notifications (id, user_id, event_type, title, body, payload, is_read, created_at)
SELECT
    gen_random_uuid(),
    u.id,
    e.event_type,
    e.title,
    CASE e.target_kind
        WHEN 'task'    THEN format(e.body_tpl, ut.task_key, ut.task_name)
        WHEN 'meeting' THEN format(e.body_tpl, um.meeting_name)
    END AS body,
    CASE e.target_kind
        WHEN 'task' THEN jsonb_build_object(
            'task_id',   ut.task_id,
            'task_key',  ut.task_key,
            'task_name', ut.task_name
        )
        WHEN 'meeting' THEN jsonb_build_object(
            'meeting_id',         um.meeting_id,
            'meeting_name',       um.meeting_name,
            'meeting_start_time', to_char(um.meeting_start_time AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
        )
    END AS payload,
    false AS is_read,
    (TIMESTAMP '2026-04-16 09:00:00'
        - make_interval(
            hours => (abs(hashtext(u.id::text || e.event_type)) % (7 * 24)),
            mins  => (abs(hashtext(u.id::text || e.event_type || 'm')) % 60)
        ))::timestamptz AS created_at
FROM users u
CROSS JOIN event_specs e
JOIN notification_settings ns
    ON ns.user_id = u.id
   AND ns.event_type = e.event_type
   AND ns.in_system = true
LEFT JOIN _user_task    ut ON ut.user_id = u.id
LEFT JOIN _user_meeting um ON um.user_id = u.id
WHERE u.deleted_at IS NULL
  -- Отбрасываем случай, когда для type='task' у юзера вообще нет задач
  -- в БД (теоретически — если БД пуста) и наоборот для meeting.
  AND (
      (e.target_kind = 'task'    AND ut.task_id    IS NOT NULL) OR
      (e.target_kind = 'meeting' AND um.meeting_id IS NOT NULL)
  );

DROP TABLE _user_task;
DROP TABLE _user_meeting;
