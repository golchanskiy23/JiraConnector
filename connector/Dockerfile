FROM golang:1.24.7 AS builder

WORKDIR /app

# Сначала копируем только файлы зависимостей для кеширования
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект в контейнер
COPY . .

# Собираем main.go из пакета connector/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -o /connector ./connector/cmd

# Открываем порт
EXPOSE 8567

# Запуск бинарника
CMD ["/connector"]
