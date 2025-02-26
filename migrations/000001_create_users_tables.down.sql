-- Удаление таблиц в обратном порядке для соблюдения зависимостей
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_history;
DROP TABLE IF EXISTS telegram_users;
DROP TABLE IF EXISTS users;