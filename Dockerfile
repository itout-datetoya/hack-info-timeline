FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main .

FROM alpine:latest

COPY --from=builder /app/main /app/main

COPY --from=builder /app/migrations ./migrations/

EXPOSE 10000

CMD ["/app/main"]