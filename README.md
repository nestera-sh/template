# Trip Service

Сервис управления поездками TripGo. Лабораторная работа 1: HTTP API и PostgreSQL.

## Требования

- Go 1.26+
- Docker (для окружения `tripgoctl`)
- `tripgoctl`
- `psql` (опционально, для проверки БД)

## Запуск

1. Поднять локальное окружение (PostgreSQL):
```bash
cp environment.toml .
tripgoctl cluster start
tripgoctl environment start

2. Запустить миграции: 
```bash
make migrate

3. Запустить лабу:
```bash
make run

4. Проверка работоспособности (должен быть статус "ок")
```bash
curl -i localhost:8080/health
curl -i localhost:8080/ready


##Переменные окружения
Все параметры читаются из переменных окружения. Локально они подхватываются из файла `.env` (через библиотеку `godotenv`). Шаблон лежит в `.env.example`.



- `HTTP_ADDR` — адрес HTTP-сервера — по умолч.: `:8080`
- `LOG_LEVEL` — уровень логирования — по умолч.: `info`
- `SHUTDOWN_TIMEOUT` — таймаут graceful shutdown — по умолчанию: `10s`
- `DATABASE_URL` — DSN для подключения к PostgreSQL
- `DATABASE_MAX_CONNS` — максимум соединений в пуле — по умолч.: `10`
- `DATABASE_MIN_CONNS` — минимум соединений в пуле — по умолч.: `1`
- `DATABASE_CONNECT_TIMEOUT` — таймаут подключения к БД — по умолч.: `5s`
- `DATABASE_QUERY_TIMEOUT` — таймаут выполнения запроса — по умолч.: `5s`
- `DATABASE_MAX_CONN_LIFETIME` — время жизни соединения — по умолч.: `1h`


## Что сделано в этой работе

- Реализованы миграции для таблиц `trips` и `trip_status_history` (с накатом и откатом).
- Подключение к PostgreSQL через `pgxpool` с настройкой пула из env. При недоступности БД сервис не стартует.
- HTTP-сервер на `net/http` с роутером `chi`, настроены таймауты и graceful shutdown по `SIGINT`/`SIGTERM`.
- Эндпоинты `/health` и `/ready`.
- Три хэндлера по контракту: `POST /api/v1/trips`, `GET /api/v1/trips/{tripId}`, `POST /api/v1/trips/{tripId}/finish`.
- Код сгенерирован по OpenAPI.
- Репозитории на `pgx` + `squirrel`, параметризованные SQL-запросы.
- Контекстный менеджер транзакций для атомарной записи в две таблицы.
- Все ошибки отдаются в формате `application/problem+json`.


## Решения

**1. Менеджер транзакций**
Реализован контекстный менеджер транзакций (`internal/txmanager`).
- `TxManager.Do(ctx, fn)` открывает транзакцию и кладёт её в `context.Value`.
- Репозитории при каждом вызове смотрят в `ctx` через `txmanager.TxFromContext(ctx)`. Если там есть `pgx.Tx` — работают через неё, если нет — через `pgxpool.Pool`.
- Вложенный `Do` переиспользует уже открытую транзакцию. Это гарантирует, что `INSERT trips` и `INSERT trip_status_history` выполняются атомарно в рамках одного вызова `CreateTrip`.

**2. Запрет двух активных поездок**
Обеспечивается на уровне БД с помощью частичного уникального индекса:
`CREATE UNIQUE INDEX trips_one_active_per_driver_idx ON trips(driver_id) WHERE status = 'active';`
При попытке создать вторую активную поездку для того же водителя БД возвращает ошибку `23505` (unique violation). Репозиторий ловит её и превращает в ошибку `ErrDriverBusy`, т.е. `409 driver_busy`. Это гарантирует корректность даже при одновременных запросах (проверено тестом с 20 параллельными запросами: 1x201, 19x409).

**3. Уровень изоляции транзакций**
Выбран `ReadCommitted` (по умолчанию в PostgreSQL). Защита от гонок обеспечена на уровне схемы и SQL-запросов, а не уровня изоляции. Запрет двух активных поездок — partial unique index. Завершение поездки — один `UPDATE trips SET ... WHERE id = $1 AND status = 'active'`. PostgreSQL сам блокирует строку, второй параллельный запрос получает `0 rows affected` и обрабатывается как `409 trip_completed`.

**4. Обработка ошибок**
Все ошибки возвращаются в формате `application/problem+json`. Ошибки парсинга path-параметров также перехватываются и возвращаются в `problem+json` через кастомный обработчик ошибок `ErrorHandlerFunc`.