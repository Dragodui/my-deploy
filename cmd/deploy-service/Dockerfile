FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o deploy-service ./cmd/deploy-service

FROM alpine:latest
COPY --from=build /app/deploy-service /deploy-service
COPY migrations/deploy /migrations
CMD ["/deploy-service"]
