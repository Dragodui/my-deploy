FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o template-service ./cmd/template-service

FROM alpine:latest
COPY --from=build /app/template-service /template-service
COPY templates/*.yaml /templates/
CMD ["/template-service"]
