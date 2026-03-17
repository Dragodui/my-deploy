FROM golang:1.25 as builder

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download

COPY . .

RUN go build -ldflags="-w -s" -o main ./cmd/main.go

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/internal/templates ./internal/templates

EXPOSE 8080
CMD ["./main"]