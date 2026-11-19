# Trip Service

Сервис управления поездками **TripGo**.

**Лабораторная работа № 3: Архитектура, качество кода и observability.**

## Требования

- Go 1.26+
- Docker
- `tripgoctl`
- `psql` (опционально)

## Запуск

### 1. Поднять локальное окружение

```bash
cp environment.toml .
tripgoctl cluster start
tripgoctl environment start
```

### 2. Накатить миграции

```bash
make migrate
```

### 3. Запустить сервис

```bash
make run
```

### 4. Проверить работоспособность

```bash
curl -i localhost:8080/health
curl -i localhost:8080/ready
```

## Переменные окружения

Настройки читаются из переменных окружения. Локально используются `.env` и шаблон `.env.example`.

| Переменная | Описание |
|---|---|
| `HTTP_ADDR` | Адрес HTTP-сервера |
| `DATABASE_URL` | Подключение к PostgreSQL |
| `SHUTDOWN_TIMEOUT` | Таймаут graceful shutdown |
| `PUSH_HTTP_URL` | Адрес Push Service |
| `POSITION_CHECK_INTERVAL` | Интервал проверки устаревших поездок |
| `POSITION_STALE_AFTER` | Время устаревания поездки |
| `POSITION_WORKERS` | Количество фоновых workers |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Адрес OTel Collector |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | Протокол экспорта телеметрии |

## Архитектура

Код разделён на слои с однонаправленными зависимостями:

| Слой | Назначение |
|---|---|
| `internal/domain` | Бизнес-логика и доменные сущности |
| `internal/usecase` | Сценарии работы с поездками |
| `internal/http` | HTTP-хендлеры и middleware |
| `internal/repository` | Работа с PostgreSQL |
| `internal/infra` | Инфраструктура и OpenTelemetry |
| `internal/container` | Сборка зависимостей |
| `internal/worker` | Фоновая обработка поездок |
| `internal/gateway` | Взаимодействие с внешними сервисами |

## Фоновая обработка

- Scanner периодически находит устаревшие поездки.
- Worker pool обрабатывает задачи параллельно.
- Уведомления отправляются через Push Gateway.
- При поступлении новых координат задача устаревания отменяется.
- Добавлены метрики очереди и работы workers.

## Тесты

Unit-тесты:

```bash
make test
```

Интеграционные тесты:

```bash
make test-integration
```

Проверка линтером:

```bash
make lint
```

## Что сделано в этой работе

- Реализована фоновая обработка устаревших поездок.
- Добавлен worker pool с ограниченной очередью.
- Реализована отправка уведомлений через Push Gateway.
- Добавлена отмена устаревших задач при обновлении координат.
- Добавлены метрики фоновой обработки.
- Реализован graceful shutdown.
- Добавлены и обновлены тесты.
- Сохранена интеграция с OpenTelemetry.