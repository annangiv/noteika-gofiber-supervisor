# Stage 1: Build the React SPA frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
# Copy source only — node_modules stays from npm ci above
COPY frontend/index.html frontend/vite.config.js frontend/eslint.config.js ./
COPY frontend/public ./public
COPY frontend/src ./src
RUN npm run build
# Vite outDir (../static) => /app/static

# Stage 2: Build the Go Backend
FROM golang:alpine AS builder
WORKDIR /app

# Copy dependency files first
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source (static/ comes from frontend-builder, not the host)
COPY actor/ actor/
COPY db/ db/
COPY handlers/ handlers/
COPY loadbalancer/ loadbalancer/
COPY supervisor/ supervisor/
COPY utils/ utils/
COPY web/ web/
COPY main.go ./

# Overwrite any host static with the Docker-built React bundle
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
