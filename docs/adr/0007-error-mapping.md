# ADR-0007: Отображение доменных ошибок в gRPC-коды

- **Статус:** accepted
- **Дата:** 2026-09-21

## Контекст

Домен возвращает типизированные ошибки (`netutil.ErrInvalidCIDR`,
`usecase.ErrInvalidRuleType`). gRPC-клиенты ожидают стандартные
коды (`InvalidArgument`, `NotFound`, `Internal` и т.д.). Нужен слой
перевода.

## Решение

В `internal/transport/grpc/server.go` функция `mapRuleError`:

- `netutil.ErrInvalidCIDR` → `codes.InvalidArgument`
- `usecase.ErrInvalidRuleType` → `codes.InvalidArgument`
- всё остальное → `codes.Internal`

Валидация обязательных полей делается в самом сервере
(`if req.GetLogin() == "" { return InvalidArgument }`).

## Последствия

- **+** Клиент получает стандартные коды, работает с любым gRPC-клиентом.
- **+** Домен не знает о gRPC — зависимость направлена внутрь.
- **−** Каждую новую доменную ошибку нужно явно замапить — не забыть.