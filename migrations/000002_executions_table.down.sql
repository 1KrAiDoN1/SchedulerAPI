-- Откат миграции: удаление таблицы executions
-- Индексы удаляются автоматически при удалении таблицы
DROP TABLE IF EXISTS executions;
