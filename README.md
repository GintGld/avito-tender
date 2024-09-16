## Структура проекта
Стек проекта:
- fiber для http сервера
- pgx для запросов к Postgres
- golang-migrate v4 для миграций бд
- mockery v2.45.1 для моков

## Параметры окружения
Кроме заданных в условии параметров окружения, есть также:
- ```OPENAPI_PATH [string]``` - копирует openapi.yml к контейнер, чтобы в последствии спека была доступна по ручке /api/openapi.
- ```HTTP_TIMEOUT [time interval]``` - таймаут http запроса.
- ```HTTP_IDLETIMEOUT [time interval]``` - http idle timeout
- ```PRETTY_LOGGER [bool]``` - флаг для использования более читаемого логгера (для дебага).

## Линтеры
Использовал стандартные инструменты:
- gopls v0.16.2
- staticcheck v0.5.1