# Build stage
FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o file-analysis-service .

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/file-analysis-service .

# Install dependencies for plagiarism detection
RUN apk add --no-cache gcc musl-dev

EXPOSE 8082

CMD ["./file-analysis-service"]