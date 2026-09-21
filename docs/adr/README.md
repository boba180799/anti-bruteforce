# Architecture Decision Records

Каталог ADR описывает ключевые архитектурные решения проекта
«Анти-брутфорс». Формат — [Майкл Найгард](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions).

## Список решений

| № | Название | Статус | Дата |
|---|----------|--------|------|
| [0000](0000-template.md) | Шаблон ADR | — | — |
| [0001](0001-use-grpc-gateway-for-rest.md) | REST через grpc-gateway | accepted | 2026-09-21 |
| [0002](0002-chain-of-responsibility.md) | Chain of Responsibility для проверки попыток | accepted | 2026-09-21 |
| [0003](0003-in-memory-storage.md) | In-memory хранилище с интерфейсами | accepted | 2026-09-21 |
| [0004](0004-whitelist-over-blacklist.md) | Whitelist приоритетнее blacklist | accepted | 2026-09-21 |
| [0005](0005-leaky-bucket.md) | Leaky bucket для rate limiting | accepted | 2026-09-21 |
| [0006](0006-ttl-cleanup.md) | TTL-очистка неактивных вёдер | accepted | 2026-09-21 |
| [0007](0007-error-mapping.md) | Доменные ошибки → gRPC-коды | accepted | 2026-09-21 |
| [0008](0008-testing-strategy.md) | Стратегия тестирования | accepted | 2026-09-21 |
| [0009](0009-configuration-via-env.md) | Конфигурация через env | accepted | 2026-09-21 |
| [0010](0010-manual-dependency-injection.md) | Ручное DI без фреймворка | accepted | 2026-09-21 |
| [0011](0011-clean-architecture-layers.md) | Слои Clean Architecture | accepted | 2026-09-21 |
| [0012](0012-distroless-image.md) | Distroless как базовый образ | accepted | 2026-09-21 |

## Что не покрыто ADR (задел на будущее)

- **Redis-репозиторий** — план миграции на распределённое хранилище.
- **Prometheus-метрики** — формат и набор метрик.
- **OpenTelemetry** — трейсинг между сервисами.
- **Валидация конфига** — `Config.Validate()`.
- **Rate limit с другим алгоритмом** (token bucket) — как альтернатива.