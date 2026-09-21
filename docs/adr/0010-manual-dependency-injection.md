# ADR-0010: Ручное внедрение зависимостей без DI-фреймворка

- **Статус:** accepted
- **Дата:** 2026-09-21

## Контекст

В проекте несколько слоёв и зависимостей: Storage → Limiter → Usecase →
gRPC-сервер. Нужен способ собрать всё это вместе в `main.go`.

## Рассмотренные варианты

1. **Google Wire** — кодогенерация. Мощно, но добавляет build-time
   зависимость и `wire_gen.go` файл.
2. **Uber Fx / Dig** — runtime-рефлексия. Гибко, но добавляет рантайм-
   магию и сложнее отлаживать.
3. **Ручное внедрение** — обычные вызовы конструкторов в `main.go`.

## Решение

Выбран **вариант 3**: явные конструкторы (`NewStorage`, `NewIPRules`,
`NewCheckAttempt`, `NewServer`) и сборка в `cmd/antibruteforce/main.go`.

Пример:
```go
storage := limiter.NewStorage(cfg)
rules := memory.NewIPRules()
checkUC := usecase.NewCheckAttempt(storage, rules)
server := grpcsrv.NewServer(checkUC, resetUC, rulesUC, log)
```

## Последствия

- **+** Никакой магии, всё видно в одном месте.
- **+** Ошибки сборки — обычные ошибки компиляции.
- **+** Нет лишних зависимостей.
- **-** При росте графа зависимостей main.go растёт. Сейчас это
  ~30 строк — приемлемо.