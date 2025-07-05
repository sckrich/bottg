FROM golang:1.24.4-alpine


WORKDIR /app

# Копируем ВСЕ файлы проекта
COPY . .

# Переходим в папку бота
WORKDIR /app/admin-bot

# Собираем
RUN go build -o /admin-bot

CMD ["/admin-bot"]