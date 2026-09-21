# Anti-Bruteforce

Сервис защиты от подбора паролей (brute-force) для бэкенда авторизации.
Проверяет каждую попытку входа по логину, паролю и IP-адресу и принимает
решение: разрешить её или заблокировать.

Обслуживается через gRPC и REST API, имеет CLI для администрирования.
Готов к запуску в Docker, покрыт юнит- и интеграционными тестами,
проходит строгий `golangci-lint`.

## Возможности

- **Leaky bucket** rate limiting по трём измерениям:
  - логин — 10 попыток в минуту,
  - пароль — 100 попыток в минуту,
  - IP-адрес — 1000 попыток в минуту.
- **Whitelist / blacklist** CIDR-подсетей (IPv4). Whitelist имеет приоритет.
- **REST API** на `/v1/...` (через grpc-gateway) и **gRPC API** на `:9090`.
- **CLI** (`abf-cli`) для администрирования: правила, сброс вёдер, проверка попыток.
- **Автоматическая очистка** неактивных вёдер по TTL.
- **Graceful shutdown** по `SIGINT`/`SIGTERM`.
- **Структурированные логи** (`slog`) и **интерсепторы** gRPC (recovery, logging).
- **Health-check** `/healthz`.

## Быстрый старт

### Требования

- Go **1.26+** (см. `go.mod`)
- Docker и Docker Compose
- `make`

### Запуск через Docker Compose

```bash
make run
```

Сервис будет доступен:
- REST: `http://localhost:8080`
- gRPC: `localhost:9090`

### Локальный запуск без Docker

```bash
make build
./bin/anti-bruteforce
```

### Проверка health-check

```bash
curl -s http://localhost:8080/healthz
# → ok
```

## Использование

### REST API

#### Проверка попытки авторизации

```bash
curl -s -X POST http://localhost:8080/v1/attempts:check \
  -H 'Content-Type: application/json' \
  -d '{"login":"alice","password":"secret","ip":"1.2.3.4"}'
# → {"ok":true}
```

#### Сброс вёдер

```bash
curl -s -X POST http://localhost:8080/v1/buckets:reset \
  -H 'Content-Type: application/json' \
  -d '{"login":"alice","ip":"1.2.3.4"}'
```

#### Управление whitelist/blacklist

```bash
# Добавить whitelist-правило
curl -s -X POST http://localhost:8080/v1/ipRules \
  -H 'Content-Type: application/json' \
  -d '{"cidr":"10.0.0.0/8","type":"IP_RULE_TYPE_WHITELIST","description":"office"}'

# Список правил
curl -s http://localhost:8080/v1/ipRules

# Удалить правило
curl -s -X DELETE http://localhost:8080/v1/ipRules/IP_RULE_TYPE_WHITELIST/10.0.0.0/8
```

### CLI

`abf-cli` работает через gRPC. Адрес сервера задаётся флагом `--server`
или переменной окружения `ABF_SERVER` (по умолчанию `localhost:9090`).

```bash
# Проверка попытки
./bin/cli check --login alice --password secret --ip 1.2.3.4
# → ALLOWED

# Управление whitelist
./bin/cli whitelist add --cidr 10.0.0.0/8 --description office
./bin/cli whitelist list
./bin/cli whitelist remove --cidr 10.0.0.0/8

# Управление blacklist
./bin/cli blacklist add --cidr 6.6.6.0/24 --description bad

# Сброс ведра
./bin/cli bucket reset --login alice --ip 1.2.3.4

# Указать другой сервер
./bin/cli --server prod.example.com:9090 whitelist list
# или
ABF_SERVER=prod.example.com:9090 ./bin/cli whitelist list
```

## Конфигурация

Все параметры задаются через переменные окружения. Значения по умолчанию
соответствуют требованиям ТЗ.

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `GRPC_ADDR` | адрес gRPC-сервера | `:9090` |
| `HTTP_ADDR` | адрес REST-сервера | `:8080` |
| `LOGIN_LIMIT` | ёмкость ведра логина | `10` |
| `LOGIN_RATE` | скорость восстановления логина (ед/сек) | `10/60` |
| `PASSWORD_LIMIT` | ёмкость ведра пароля | `100` |
| `PASSWORD_RATE` | скорость восстановления пароля | `100/60` |
| `IP_LIMIT` | ёмкость ведра IP | `1000` |
| `IP_RATE` | скорость восстановления IP | `1000/60` |
| `BUCKET_TTL` | время жизни неактивного ведра | `10m` |
| `SHUTDOWN_TIMEOUT` | таймаут graceful shutdown | `10s` |

Пример:

```bash
LOGIN_LIMIT=5 IP_LIMIT=100 ./bin/anti-bruteforce
```

## Архитектура

Проект построен по принципам **Clean Architecture** с разделением
на слои. Зависимости направлены внутрь: транспорт → usecase → домен.

```
cmd/
  antibruteforce/    — точка входа сервиса
  cli/               — CLI на Cobra
api/proto/v1/        — .proto контракт + сгенерированный код
internal/
  domain/            — не здесь, доменные типы живут в usecase
  usecase/           — бизнес-логика (CheckAttempt, ResetBucket, ManageRules)
  limiter/           — leaky bucket + TTL-хранилище вёдер
  netutil/           — CIDR-матчер для whitelist/blacklist
  repository/memory/ — in-memory реализация IPRuleStore
  transport/grpc/    — gRPC-сервер, интерсепторы, интеграционные тесты
  transport/http/    — REST через grpc-gateway
  config/            — загрузка конфигурации из env
third_party/google/  — .proto Google API (annotations, http)
deploy/              — docker-compose.yml
```

### Ключевые паттерны

- **Chain of Responsibility** — `usecase.CheckAttempt` пропускает попытку
  через цепочку правил: whitelist → blacklist → login → password → ip.
- **Repository** — интерфейсы `Limiter`, `IPRuleStore` в usecase,
  реализации в `limiter` и `repository/memory`.
- **Decorator** — gRPC-интерсепторы (`UnaryLoggingInterceptor`,
  `UnaryRecoveryInterceptor`).
- **Strategy** — `StorageConfig` позволяет подменить параметры лимитов
  без изменения кода.

### Решение о приоритете whitelist

Если IP входит одновременно в whitelist и blacklist, **побеждает whitelist**.
Обоснование: явное доверие перекрывает явный запрет. Это позволяет
администраторам быстро разрешать доступ через whitelist даже при
наличии широкого blacklist-правила.

## Тестирование

```bash
# Юнит-тесты (быстрый прогон)
make test

# Полный прогон как в CI: race detector + 100 повторов
make test-full

# Только интеграционные тесты
go test ./internal/transport/grpc/ -v

# С покрытием
go test -cover ./internal/...
```

### Что покрыто тестами

- **`internal/limiter`** — алгоритм leaky bucket, потокобезопасность,
  TTL-очистка вёдер.
- **`internal/netutil`** — CIDR-матчер: точное попадание, границы,
  невалидные входы, /0 и /32.
- **`internal/usecase`** — все правила Chain of Responsibility,
  поведение `ManageRules`, edge cases.
- **`internal/transport/grpc`** — интеграционные тесты через `bufconn`:
  check-сценарии, whitelist/blacklist, CRUD, валидация, IP-лимит.

## Разработка

### Генерация proto

```bash
make proto
```

### Линтер

```bash
make lint
```

Используется [`golangci-lint`](https://golangci-lint.run/) с конфигом
`.golangci.yml`.

### Сборка Docker-образа

```bash
make docker-build
make docker-run
```

## CI/CD

GitHub Actions прогоняет три job'а на каждый push в `master`:

1. **lint** — `golangci-lint` с `.golangci.yml`.
2. **test** — `go test -race -count 100 ./...`.
3. **build** — сборка бинаря `anti-bruteforce` для Go 1.26+.

