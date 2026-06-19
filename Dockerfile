# Stage 1: Build the React SPA frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build the Go Backend
FROM golang:alpine AS builder
WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Copy all source files
COPY . .

# Copy built frontend assets to the backend's static directory
COPY --from=frontend-builder /app/static ./static

# Build the Go web server binary
RUN go build -o server main.go

# Stage 3: Production container runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["./server"]
