# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем статический бинарь
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/anti-bruteforce ./cmd/antibruteforce
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/cli ./cmd/cli

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/anti-bruteforce /anti-bruteforce
COPY --from=builder /out/cli /cli

USER nonroot:nonroot
EXPOSE 8080 9090

ENTRYPOINT ["/anti-bruteforce"]