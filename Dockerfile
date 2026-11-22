FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/internal/i18n ./internal/i18n
COPY --from=builder /app/swagger.yaml .

RUN mkdir -p uploads

EXPOSE 8080

CMD ["./main"]