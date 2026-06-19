FROM golang:alpine AS builder

WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum ./

# Copy all source files
COPY . .

# Run mod tidy to fetch dependencies and build the binary
RUN go mod tidy && go build -o server main.go

# Production stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
# Copy static files if they exist
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["./server"]
