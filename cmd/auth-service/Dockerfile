FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o auth-service ./cmd/auth-service

FROM alpine:latest
COPY --from=build /app/auth-service /auth-service
COPY migrations/auth /migrations
CMD ["/auth-service"]