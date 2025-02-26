FROM golang:1.23-alpine AS builder

WORKDIR /app

# Установка необходимых зависимостей
RUN apk add --no-cache gcc musl-dev

# Копирование и кэширование зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копирование всего проекта
COPY . .

# Компиляция приложения
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o auth-service ./cmd/auth-service

# Финальный образ
FROM alpine:latest

WORKDIR /app

# Установка необходимых runtime зависимостей
RUN apk --no-cache add ca-certificates tzdata

# Копирование бинарного файла из builder
COPY --from=builder /app/auth-service .

# Установка переменных окружения
ENV TZ=Europe/Moscow

# Открытие порта
EXPOSE 8080

# Запуск приложения
CMD ["./auth-service"]