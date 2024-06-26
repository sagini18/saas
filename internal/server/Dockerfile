FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY ../../../go.mod ../../../go.sum ./
RUN go mod download

COPY ../../../ .

RUN go build -o main ./internal/server/server.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/main .

RUN chmod +x ./main

CMD ["./main"]
