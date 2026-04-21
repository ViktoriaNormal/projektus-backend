-- =============================================================================
-- 000027: Доработка сид-данных
--   1. Добавить вторую доску «Тестирование» к каждому проекту
--      с колонками и дорожками (для Kanban).
--   2. Сгенерировать ≈25 QA-задач на каждой тестовой доске с тематическими
--      названиями, исполнителями, статусами и датами создания в рабочих
--      днях пн–пт периода 02.02.2026–16.04.2026.
--   3. Сформировать task_status_history для всех задач (основной доски и
--      тестовой), чтобы ожили CFD / Cycle Time / Throughput / Velocity /
--      Burndown.
-- =============================================================================

SET client_encoding = 'UTF8';

-- ======================== 1. Вторые доски «Тестирование» ========================

-- Scrum MOB test
INSERT INTO boards (id, project_id, name, description, sort_order, is_default, priority_type, estimation_unit, swimlane_group_by, priority_options) VALUES
    ('d0000000-0000-0001-0001-000000000000', 'c0000000-0000-0000-0001-000000000000',
     'Тестирование', 'Доска команды QA: ручные проверки, автотесты, баги', 2, false, 'priority', 'story_points', '',
     '["Низкий","Средний","Высокий","Критичный"]'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO columns (id, board_id, name, system_type, sort_order, is_locked) VALUES
    ('d0000000-0000-0001-0001-000000000001', 'd0000000-0000-0001-0001-000000000000', 'К тестированию', 'initial',     1, false),
    ('d0000000-0000-0001-0001-000000000002', 'd0000000-0000-0001-0001-000000000000', 'В тесте',        'in_progress', 2, false),
    ('d0000000-0000-0001-0001-000000000003', 'd0000000-0000-0001-0001-000000000000', 'Баг',            'in_progress', 3, false),
    ('d0000000-0000-0001-0001-000000000004', 'd0000000-0000-0001-0001-000000000000', 'Проверено',      'completed',   4, false)
ON CONFLICT (id) DO NOTHING;

-- Scrum MKT test
INSERT INTO boards (id, project_id, name, description, sort_order, is_default, priority_type, estimation_unit, swimlane_group_by, priority_options) VALUES
    ('d0000000-0000-0001-0002-000000000000', 'c0000000-0000-0000-0002-000000000000',
     'Тестирование', 'Доска команды QA: ручные проверки, автотесты, баги', 2, false, 'priority', 'story_points', '',
     '["Низкий","Средний","Высокий","Критичный"]'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO columns (id, board_id, name, system_type, sort_order, is_locked) VALUES
    ('d0000000-0000-0001-0002-000000000001', 'd0000000-0000-0001-0002-000000000000', 'К тестированию', 'initial',     1, false),
    ('d0000000-0000-0001-0002-000000000002', 'd0000000-0000-0001-0002-000000000000', 'В тесте',        'in_progress', 2, false),
    ('d0000000-0000-0001-0002-000000000003', 'd0000000-0000-0001-0002-000000000000', 'Баг',            'in_progress', 3, false),
    ('d0000000-0000-0001-0002-000000000004', 'd0000000-0000-0001-0002-000000000000', 'Проверено',      'completed',   4, false)
ON CONFLICT (id) DO NOTHING;

-- Kanban INF test
INSERT INTO boards (id, project_id, name, description, sort_order, is_default, priority_type, estimation_unit, swimlane_group_by, priority_options) VALUES
    ('d0000000-0000-0001-0003-000000000000', 'c0000000-0000-0000-0003-000000000000',
     'Тестирование', 'QA-доска инфраструктуры: smoke, chaos, нагрузки', 2, false, 'service_class', 'time',
     '00000000-0000-0000-0001-000000000008',
     '["Ускоренный","С фиксированной датой","Стандартный","Нематериальный"]'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO columns (id, board_id, name, system_type, wip_limit, sort_order, is_locked) VALUES
    ('d0000000-0000-0001-0003-000000000001', 'd0000000-0000-0001-0003-000000000000', 'К тестированию', 'initial',     NULL, 1, false),
    ('d0000000-0000-0001-0003-000000000002', 'd0000000-0000-0001-0003-000000000000', 'Готово к прогону','initial',    6,    2, false),
    ('d0000000-0000-0001-0003-000000000003', 'd0000000-0000-0001-0003-000000000000', 'В тесте',        'in_progress', 5,    3, false),
    ('d0000000-0000-0001-0003-000000000004', 'd0000000-0000-0001-0003-000000000000', 'Баг',            'in_progress', 4,    4, false),
    ('d0000000-0000-0001-0003-000000000005', 'd0000000-0000-0001-0003-000000000000', 'Проверено',      'completed',   NULL, 5, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO swimlanes (id, board_id, name, sort_order) VALUES
    ('d0000000-0000-0001-0003-000000000011', 'd0000000-0000-0001-0003-000000000000', 'Ускоренный',            1),
    ('d0000000-0000-0001-0003-000000000012', 'd0000000-0000-0001-0003-000000000000', 'С фиксированной датой', 2),
    ('d0000000-0000-0001-0003-000000000013', 'd0000000-0000-0001-0003-000000000000', 'Стандартный',           3),
    ('d0000000-0000-0001-0003-000000000014', 'd0000000-0000-0001-0003-000000000000', 'Нематериальный',        4)
ON CONFLICT (id) DO NOTHING;

-- Kanban SITE test
INSERT INTO boards (id, project_id, name, description, sort_order, is_default, priority_type, estimation_unit, swimlane_group_by, priority_options) VALUES
    ('d0000000-0000-0001-0004-000000000000', 'c0000000-0000-0000-0004-000000000000',
     'Тестирование', 'QA-доска сайта: кроссбраузерность, SEO, Lighthouse', 2, false, 'service_class', 'time',
     '00000000-0000-0000-0001-000000000008',
     '["Ускоренный","С фиксированной датой","Стандартный","Нематериальный"]'::jsonb)
ON CONFLICT (id) DO NOTHING;

INSERT INTO columns (id, board_id, name, system_type, wip_limit, sort_order, is_locked) VALUES
    ('d0000000-0000-0001-0004-000000000001', 'd0000000-0000-0001-0004-000000000000', 'К тестированию', 'initial',     NULL, 1, false),
    ('d0000000-0000-0001-0004-000000000002', 'd0000000-0000-0001-0004-000000000000', 'Готово к прогону','initial',    5,    2, false),
    ('d0000000-0000-0001-0004-000000000003', 'd0000000-0000-0001-0004-000000000000', 'В тесте',        'in_progress', 4,    3, false),
    ('d0000000-0000-0001-0004-000000000004', 'd0000000-0000-0001-0004-000000000000', 'Баг',            'in_progress', 3,    4, false),
    ('d0000000-0000-0001-0004-000000000005', 'd0000000-0000-0001-0004-000000000000', 'Проверено',      'completed',   NULL, 5, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO swimlanes (id, board_id, name, sort_order) VALUES
    ('d0000000-0000-0001-0004-000000000011', 'd0000000-0000-0001-0004-000000000000', 'Ускоренный',            1),
    ('d0000000-0000-0001-0004-000000000012', 'd0000000-0000-0001-0004-000000000000', 'С фиксированной датой', 2),
    ('d0000000-0000-0001-0004-000000000013', 'd0000000-0000-0001-0004-000000000000', 'Стандартный',           3),
    ('d0000000-0000-0001-0004-000000000014', 'd0000000-0000-0001-0004-000000000000', 'Нематериальный',        4)
ON CONFLICT (id) DO NOTHING;

-- ======================== 3. QA-задачи на тестовых досках ========================
-- 25 тематических названий на каждый проект. Исполнители выбираются из списка
-- участников проекта. Распределение статусов: 40% проверено, 25% в тесте,
-- 20% баг, 15% к тестированию (по хэшу id).

CREATE TEMP TABLE _qa_topics (
    project_id UUID,
    topic_n INT,
    topic_name TEXT
);

-- MOB
INSERT INTO _qa_topics VALUES
    ('c0000000-0000-0000-0001-000000000000'::uuid, 1,  'Написать автотесты на экран входа'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 2,  'Регрессионное тестирование авторизации'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 3,  'Проверить Face ID на iPhone 13/14/15'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 4,  'Проверить Touch ID на Android-устройствах'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 5,  'Баг: при переводе между своими счетами списывается комиссия'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 6,  'Smoke-тест критичных сценариев переводов'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 7,  'Проверить платёж по СБП на разных банках'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 8,  'Баг: при оплате ЖКХ теряется часть комментария'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 9,  'Проверить push-уведомления в оффлайне'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 10, 'Нагрузочное тестирование авторизации (1000 RPS)'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 11, 'Написать e2e-тест открытия вклада'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 12, 'Проверить экспорт выписки в PDF на больших периодах'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 13, 'Баг: краш при переключении на iPad в landscape'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 14, 'Проверить блокировку карты из приложения'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 15, 'Smoke-тест релиза v2.3'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 16, 'Регресс шаблонов платежей'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 17, 'Тест сценария с ошибкой сети при переводе'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 18, 'Проверить чат с поддержкой: отправка файлов'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 19, 'Баг: неверный остаток при быстром переключении счетов'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 20, 'Проверить установку лимитов по карте'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 21, 'A/B-тестирование нового дизайна главного экрана'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 22, 'Автоматизация проверки аналитики расходов'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 23, 'Совместимость с iOS 17 и Android 14'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 24, 'Тест уведомлений с заблокированным экраном'),
    ('c0000000-0000-0000-0001-000000000000'::uuid, 25, 'Регресс всех сценариев перед мажорным релизом');

-- MKT
INSERT INTO _qa_topics VALUES
    ('c0000000-0000-0000-0002-000000000000'::uuid, 1,  'Автотесты API каталога товаров'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 2,  'Проверить массовую загрузку XLSX: 10К карточек'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 3,  'Баг: при редактировании характеристик товара теряется галерея'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 4,  'Регресс модерации карточек'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 5,  'E2E: полный сценарий от регистрации продавца до продажи'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 6,  'Нагрузка API вебхуков (500 RPS)'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 7,  'Проверить синхронизацию остатков с 1С'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 8,  'Баг: промо-код не применяется при повторном входе в корзину'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 9,  'Тест отчёта по продажам за длинный период'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 10, 'Проверить выгрузку для налогового учёта'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 11, 'Автотесты рейтинга и отзывов'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 12, 'Баг: задваивается ответ продавца на отзыв'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 13, 'Проверить ограничения прав ролей продавца'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 14, 'E2E: возврат товара с претензией'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 15, 'Нагрузка массовых импортов цен'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 16, 'Баг: сбой биллинга при смене тарифа'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 17, 'Проверить квоты API на превышение'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 18, 'Smoke-тест после каждого деплоя'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 19, 'Тест печати этикеток для разных принтеров'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 20, 'Баг: некорректные остатки при одновременных заказах'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 21, 'Автоматизация отчётов по конверсии'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 22, 'Проверить работу аналитики на больших объёмах'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 23, 'Интеграционные тесты с тикет-системой'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 24, 'Регресс всех сценариев оплаты комиссий'),
    ('c0000000-0000-0000-0002-000000000000'::uuid, 25, 'Финальный прогон перед релизом');

-- INF
INSERT INTO _qa_topics VALUES
    ('c0000000-0000-0000-0003-000000000000'::uuid, 1,  'Smoke-тест кластера после обновления K8s'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 2,  'Проверить работоспособность Ingress после деплоя'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 3,  'Тест автопродления сертификатов Let''s Encrypt'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 4,  'Chaos: выключение worker-ноды'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 5,  'Chaos: сбой сетевой политики'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 6,  'Нагрузка 10К RPS на payments-api'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 7,  'Проверить HPA на росте нагрузки'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 8,  'Баг: Prometheus теряет метрики при рестарте'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 9,  'Тест backup etcd и восстановление'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 10, 'Проверить алерты по SLO'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 11, 'Баг: дублирующиеся события в Loki'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 12, 'Тест canary-деплоя с роллбэком'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 13, 'Проверить работу ArgoCD sync'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 14, 'Баг: tracing пропадает при пиковой нагрузке'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 15, 'Тест NetworkPolicy между неймспейсами'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 16, 'Smoke-проверка Vault после перезапуска'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 17, 'Нагрузочное тестирование auth-api'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 18, 'Проверить сохранение состояния в stateful-сервисах'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 19, 'Баг: pod crash loop после миграции'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 20, 'Тест failover для PostgreSQL в кластере'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 21, 'Проверить корректность логов access/error'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 22, 'Тест spot-нод для stateless-воркеров'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 23, 'Аудит безопасности кластера: CIS Benchmark'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 24, 'Баг: утечка памяти в notifications-worker'),
    ('c0000000-0000-0000-0003-000000000000'::uuid, 25, 'Финальный приёмочный тест миграции');

-- SITE
INSERT INTO _qa_topics VALUES
    ('c0000000-0000-0000-0004-000000000000'::uuid, 1,  'Cross-browser тест главной страницы'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 2,  'Lighthouse-аудит Core Web Vitals'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 3,  'Валидация микроразметки Schema.org'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 4,  'Тест формы обратной связи на спам'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 5,  'Баг: форма отклика на вакансию не принимает PDF > 5MB'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 6,  'Проверить sitemap.xml для всех разделов'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 7,  'Тест отзывчивости вёрстки на мобильных'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 8,  'Smoke-тест CMS: создание и публикация новости'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 9,  'Баг: WYSIWYG теряет форматирование при вставке из Word'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 10, 'Тест прав доступа редакторов в CMS'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 11, 'Проверить локализацию EN на всех страницах'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 12, 'Баг: некорректный hreflang для EN/RU'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 13, 'Тест скорости загрузки с CDN'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 14, 'Проверить корректность событий Яндекс.Метрики'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 15, 'Баг: cookie-банер не закрывается в Firefox'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 16, 'Регресс SEO-метатегов на всех страницах'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 17, 'Тест версионирования черновиков в CMS'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 18, 'Smoke-тест рассылки по контактам'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 19, 'Баг: 404 при переходе на страницу услуги из поиска'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 20, 'Тест доступности (a11y): WCAG 2.1 AA'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 21, 'Проверить работу поиска по сайту'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 22, 'Баг: длинные заголовки ломают сетку на iPad'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 23, 'Тест производительности медиатеки при 1000+ файлов'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 24, 'Регресс политики cookies после обновления'),
    ('c0000000-0000-0000-0004-000000000000'::uuid, 25, 'Финальный приёмочный тест дизайна');

-- Генерация QA-задач
CREATE TEMP TABLE _qa_tasks (
    id UUID,
    key TEXT,
    project_id UUID,
    board_id UUID,
    owner_member_n INT,
    executor_member_n INT,
    name TEXT,
    column_id UUID,
    swimlane_id UUID,
    priority TEXT,
    estimation TEXT,
    created_at TIMESTAMPTZ,
    status_bucket TEXT  -- 'backlog'|'inprogress'|'review'|'done'
);

-- MOB QA (25 задач, проект 30 участников → member_n 1..30)
INSERT INTO _qa_tasks
SELECT
    ('e2000000-0000-0000-0001-' || lpad(n::text, 12, '0'))::uuid,
    'MOB-' || (200 + n),
    'c0000000-0000-0000-0001-000000000000'::uuid,
    'd0000000-0000-0001-0001-000000000000'::uuid,
    ((n * 7) % 30) + 1,
    ((n * 13) % 30) + 1,
    tt.topic_name,
    NULL::uuid, NULL::uuid,
    (ARRAY['Низкий','Средний','Высокий','Критичный'])[((n * 3) % 4) + 1],
    ((n * 2) % 8 + 1)::text,
    NULL::timestamptz, NULL::text
FROM generate_series(1, 25) AS n
JOIN _qa_topics tt ON tt.project_id = 'c0000000-0000-0000-0001-000000000000'::uuid AND tt.topic_n = n;

-- MKT QA (35 участников)
INSERT INTO _qa_tasks
SELECT
    ('e2000000-0000-0000-0002-' || lpad(n::text, 12, '0'))::uuid,
    'MKT-' || (200 + n),
    'c0000000-0000-0000-0002-000000000000'::uuid,
    'd0000000-0000-0001-0002-000000000000'::uuid,
    ((n * 7) % 35) + 1,
    ((n * 13) % 35) + 1,
    tt.topic_name,
    NULL::uuid, NULL::uuid,
    (ARRAY['Низкий','Средний','Высокий','Критичный'])[((n * 3) % 4) + 1],
    ((n * 2) % 8 + 1)::text,
    NULL::timestamptz, NULL::text
FROM generate_series(1, 25) AS n
JOIN _qa_topics tt ON tt.project_id = 'c0000000-0000-0000-0002-000000000000'::uuid AND tt.topic_n = n;

-- INF QA (26 участников)
INSERT INTO _qa_tasks
SELECT
    ('e2000000-0000-0000-0003-' || lpad(n::text, 12, '0'))::uuid,
    'INF-' || (200 + n),
    'c0000000-0000-0000-0003-000000000000'::uuid,
    'd0000000-0000-0001-0003-000000000000'::uuid,
    ((n * 7) % 25) + 1,
    ((n * 13) % 25) + 1,
    tt.topic_name,
    NULL::uuid, NULL::uuid,
    (ARRAY['Ускоренный','С фиксированной датой','Стандартный','Нематериальный'])[((n * 3) % 4) + 1],
    ((n * 3) % 6 + 1)::text || 'ч',
    NULL::timestamptz, NULL::text
FROM generate_series(1, 25) AS n
JOIN _qa_topics tt ON tt.project_id = 'c0000000-0000-0000-0003-000000000000'::uuid AND tt.topic_n = n;

-- SITE QA (28 участников)
INSERT INTO _qa_tasks
SELECT
    ('e2000000-0000-0000-0004-' || lpad(n::text, 12, '0'))::uuid,
    'SITE-' || (200 + n),
    'c0000000-0000-0000-0004-000000000000'::uuid,
    'd0000000-0000-0001-0004-000000000000'::uuid,
    ((n * 7) % 28) + 1,
    ((n * 13) % 28) + 1,
    tt.topic_name,
    NULL::uuid, NULL::uuid,
    (ARRAY['Ускоренный','С фиксированной датой','Стандартный','Нематериальный'])[((n * 3) % 4) + 1],
    ((n * 3) % 6 + 1)::text || 'ч',
    NULL::timestamptz, NULL::text
FROM generate_series(1, 25) AS n
JOIN _qa_topics tt ON tt.project_id = 'c0000000-0000-0000-0004-000000000000'::uuid AND tt.topic_n = n;

-- Распределить статусы по хэшу: 40% done, 25% inprogress, 20% review(=bug), 15% backlog
UPDATE _qa_tasks
SET status_bucket = CASE
    WHEN (abs(hashtext(id::text)) % 100) < 40 THEN 'done'
    WHEN (abs(hashtext(id::text)) % 100) < 65 THEN 'inprogress'
    WHEN (abs(hashtext(id::text)) % 100) < 85 THEN 'review'
    ELSE 'backlog'
  END;

-- Даты создания — рабочие дни периода
UPDATE _qa_tasks t
SET created_at = (
    (DATE '2026-02-02' + ((abs(hashtext(t.id::text)) % 74) || ' days')::interval)::timestamptz
    + make_interval(hours => 9 + ((abs(hashtext(t.id::text)) / 7) % 9),
                    mins  => ((abs(hashtext(t.id::text)) / 11) % 60))
);
UPDATE _qa_tasks SET created_at = created_at + INTERVAL '2 days' WHERE EXTRACT(DOW FROM created_at) = 6;
UPDATE _qa_tasks SET created_at = created_at + INTERVAL '1 day'  WHERE EXTRACT(DOW FROM created_at) = 0;

-- column_id / swimlane_id по статусу и типу доски
-- Scrum MOB test
UPDATE _qa_tasks
SET column_id = CASE status_bucket
    WHEN 'backlog'    THEN 'd0000000-0000-0001-0001-000000000001'::uuid
    WHEN 'inprogress' THEN 'd0000000-0000-0001-0001-000000000002'::uuid
    WHEN 'review'     THEN 'd0000000-0000-0001-0001-000000000003'::uuid
    WHEN 'done'       THEN 'd0000000-0000-0001-0001-000000000004'::uuid
  END
WHERE project_id = 'c0000000-0000-0000-0001-000000000000'::uuid;

-- Scrum MKT test
UPDATE _qa_tasks
SET column_id = CASE status_bucket
    WHEN 'backlog'    THEN 'd0000000-0000-0001-0002-000000000001'::uuid
    WHEN 'inprogress' THEN 'd0000000-0000-0001-0002-000000000002'::uuid
    WHEN 'review'     THEN 'd0000000-0000-0001-0002-000000000003'::uuid
    WHEN 'done'       THEN 'd0000000-0000-0001-0002-000000000004'::uuid
  END
WHERE project_id = 'c0000000-0000-0000-0002-000000000000'::uuid;

-- Kanban INF test
UPDATE _qa_tasks
SET column_id = CASE status_bucket
    WHEN 'backlog'    THEN 'd0000000-0000-0001-0003-000000000001'::uuid
    WHEN 'inprogress' THEN 'd0000000-0000-0001-0003-000000000003'::uuid
    WHEN 'review'     THEN 'd0000000-0000-0001-0003-000000000004'::uuid
    WHEN 'done'       THEN 'd0000000-0000-0001-0003-000000000005'::uuid
  END,
  swimlane_id = CASE priority
    WHEN 'Ускоренный'            THEN 'd0000000-0000-0001-0003-000000000011'::uuid
    WHEN 'С фиксированной датой' THEN 'd0000000-0000-0001-0003-000000000012'::uuid
    WHEN 'Стандартный'           THEN 'd0000000-0000-0001-0003-000000000013'::uuid
    WHEN 'Нематериальный'        THEN 'd0000000-0000-0001-0003-000000000014'::uuid
  END
WHERE project_id = 'c0000000-0000-0000-0003-000000000000'::uuid;

-- Kanban SITE test
UPDATE _qa_tasks
SET column_id = CASE status_bucket
    WHEN 'backlog'    THEN 'd0000000-0000-0001-0004-000000000001'::uuid
    WHEN 'inprogress' THEN 'd0000000-0000-0001-0004-000000000003'::uuid
    WHEN 'review'     THEN 'd0000000-0000-0001-0004-000000000004'::uuid
    WHEN 'done'       THEN 'd0000000-0000-0001-0004-000000000005'::uuid
  END,
  swimlane_id = CASE priority
    WHEN 'Ускоренный'            THEN 'd0000000-0000-0001-0004-000000000011'::uuid
    WHEN 'С фиксированной датой' THEN 'd0000000-0000-0001-0004-000000000012'::uuid
    WHEN 'Стандартный'           THEN 'd0000000-0000-0001-0004-000000000013'::uuid
    WHEN 'Нематериальный'        THEN 'd0000000-0000-0001-0004-000000000014'::uuid
  END
WHERE project_id = 'c0000000-0000-0000-0004-000000000000'::uuid;

-- Вставляем QA-задачи
INSERT INTO tasks (id, key, project_id, board_id, owner_id, executor_id, name, column_id, swimlane_id, priority, estimation, created_at)
SELECT
    t.id, t.key, t.project_id, t.board_id,
    ('c1000000-0000-0000-' || lpad(split_part(t.project_id::text, '-', 4), 4, '0') || '-' || lpad(t.owner_member_n::text, 12, '0'))::uuid,
    CASE WHEN (abs(hashtext(t.id::text)) % 5) = 0 THEN NULL
         ELSE ('c1000000-0000-0000-' || lpad(split_part(t.project_id::text, '-', 4), 4, '0') || '-' || lpad(t.executor_member_n::text, 12, '0'))::uuid
    END,
    t.name, t.column_id, t.swimlane_id, t.priority, t.estimation, t.created_at
FROM _qa_tasks t
ON CONFLICT (id) DO NOTHING;

-- ======================== 4. task_status_history для всех задач ========================
-- Генерируем цепочку переходов колонок:
--   * backlog    → 1 запись (initial, left_at=NULL)
--   * inprogress → initial → in_progress (последняя left_at=NULL)
--   * review     → initial → in_progress-1 → in_progress-2/review (последняя left_at=NULL)
--   * done       → initial → in_progress-1 → in_progress-2 → completed

WITH task_board_cols AS (
    SELECT
        t.id AS task_id,
        t.created_at,
        t.board_id,
        t.column_id AS current_column,
        (SELECT id FROM columns WHERE board_id = t.board_id AND system_type = 'initial'
             ORDER BY sort_order LIMIT 1) AS col_initial,
        (SELECT id FROM columns WHERE board_id = t.board_id AND system_type = 'in_progress'
             ORDER BY sort_order LIMIT 1) AS col_inprog1,
        (SELECT id FROM columns WHERE board_id = t.board_id AND system_type = 'in_progress'
             ORDER BY sort_order DESC LIMIT 1) AS col_inprog2,
        (SELECT id FROM columns WHERE board_id = t.board_id AND system_type = 'completed'
             ORDER BY sort_order LIMIT 1) AS col_completed
    FROM tasks t
),
task_stage AS (
    SELECT
        tb.*,
        CASE
            WHEN tb.current_column = tb.col_completed                                   THEN 3  -- done
            WHEN tb.current_column = tb.col_inprog2 AND tb.col_inprog2 <> tb.col_inprog1 THEN 2 -- review/bug
            WHEN tb.current_column = tb.col_inprog1                                     THEN 1  -- inprogress
            ELSE 0                                                                            -- backlog
        END AS stage
    FROM task_board_cols tb
)
INSERT INTO task_status_history (id, task_id, column_id, entered_at, left_at)
SELECT gen_random_uuid(), task_id, column_id, entered_at, left_at FROM (
    -- initial (всегда одна запись)
    SELECT
        ts.task_id,
        ts.col_initial AS column_id,
        ts.created_at AS entered_at,
        CASE WHEN ts.stage >= 1 THEN ts.created_at + INTERVAL '1 day' ELSE NULL END AS left_at
    FROM task_stage ts
    WHERE ts.col_initial IS NOT NULL

    UNION ALL

    -- in_progress-1 (for stage >= 1)
    SELECT
        ts.task_id,
        ts.col_inprog1,
        ts.created_at + INTERVAL '1 day',
        CASE WHEN ts.stage >= 2 THEN ts.created_at + INTERVAL '5 days'
             WHEN ts.stage >= 3 AND ts.col_inprog2 = ts.col_inprog1 THEN ts.created_at + INTERVAL '7 days'
             ELSE NULL END
    FROM task_stage ts
    WHERE ts.stage >= 1 AND ts.col_inprog1 IS NOT NULL

    UNION ALL

    -- in_progress-2 / review (only if different from in_progress-1, for stage >= 2)
    SELECT
        ts.task_id,
        ts.col_inprog2,
        ts.created_at + INTERVAL '5 days',
        CASE WHEN ts.stage >= 3 THEN ts.created_at + INTERVAL '7 days' ELSE NULL END
    FROM task_stage ts
    WHERE ts.stage >= 2 AND ts.col_inprog2 IS NOT NULL AND ts.col_inprog2 <> ts.col_inprog1

    UNION ALL

    -- completed (stage = 3)
    SELECT
        ts.task_id,
        ts.col_completed,
        ts.created_at + INTERVAL '7 days',
        NULL
    FROM task_stage ts
    WHERE ts.stage >= 3 AND ts.col_completed IS NOT NULL
) h
ON CONFLICT DO NOTHING;

-- Очистка
DROP TABLE IF EXISTS _qa_tasks;
DROP TABLE IF EXISTS _qa_topics;
