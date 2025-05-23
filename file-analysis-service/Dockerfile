FROM golang:1.24.0 AS builder

WORKDIR /app
COPY . .
RUN go get github.com/lib/pq
RUN go get github.com/google/uuid
RUN CGO_ENABLED=0 GOOS=linux go build -o file-analysis-service .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/file-analysis-service .
EXPOSE 8082
CMD ["./file-analysis-service"]