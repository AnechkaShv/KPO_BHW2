FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o file-storing-service .

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/file-storing-service .

EXPOSE 8081

CMD ["./file-storing-service"]