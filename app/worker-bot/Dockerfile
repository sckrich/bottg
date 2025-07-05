FROM golang:1.24.4-alpine
WORKDIR /app

COPY . .

WORKDIR /app/worker-bot

RUN go build -o /worker-bot

CMD ["/worker-bot"]