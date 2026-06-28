FROM golang:1.26.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o bin/cat-service ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /app/bin/cat-service .
EXPOSE 8081
CMD ["./cat-service"]
