# ADR-0011: Слои Clean Architecture и направление зависимостей

- **Статус:** accepted
- **Дата:** 2026-09-21

## Контекст

Проект должен оставаться тестируемым и независимым от транспорта
и хранилища. Нужно определить границы между слоями.

## Решение

Четыре слоя с направлением зависимостей **внутрь**:

cmd/ → transport → usecase → domain
↑
repository (реализация)


### Правила

1. **Домен** — типы `Attempt`, `Decision`, `Rule`, `RuleInfo`.
   Живёт внутри `internal/usecase`. Домен не импортирует ничего,
   кроме стандартной библиотеки.
2. **Usecase** — бизнес-логика. Определяет **интерфейсы**
   `Limiter`, `IPRuleStore`. Не знает о gRPC, HTTP, SQL, Redis.
3. **Transport** (`internal/transport/grpc`, `internal/transport/http`) —
   знает о proto, gRPC, gateway. Знает о usecase. Не знает о
   конкретных хранилищах.
4. **Repository** (`internal/repository/memory`) — реализует интерфейсы
   usecase. Знает о `netutil.List`. Не знает о транспорте.
5. **cmd** — собирает всё вместе.

### Проверка: интерфейсы — в usecase, реализации — снаружи

`usecase.Limiter` определён в usecase. `limiter.Storage` — в пакете
`limiter`, который **не импортируется** usecase'ом. Usecase получает
`*Storage` как `Limiter` через dependency injection.

## Последствия

- **+** Usecase тестируется с моками без gRPC и БД.
- **+** Легко подменить Storage на Redis, транспорт на CLI.
- **+** Циклических импортов быть не может — направление одно.
- **−** Иногда приходится создавать «лишние» интерфейсы с одним
  методом — но это осознанная плата за развязку.