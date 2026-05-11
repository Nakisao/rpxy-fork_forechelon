# Multi-stage build for RoProxy Lite
FROM golang:1.21-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o roproxy-lite main.go

FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/roproxy-lite .

EXPOSE 8080
CMD ["./roproxy-lite"]
