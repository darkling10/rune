# Stage 1: Build the React UI
FROM node:20-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build the Go Backend
FROM golang:1.22-alpine AS backend-builder
WORKDIR /app
# Install git for go-git dependencies
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Note: In a future iteration, we will use go:embed to bundle the React UI.
# For now, we just build the Go binary.
RUN CGO_ENABLED=0 GOOS=linux go build -o rune-server ./cmd/server/main.go

# Stage 3: Final Minimal Image
FROM alpine:3.19
WORKDIR /app

# Install runtime dependencies needed for pipeline execution
# (e.g., git for cloning, curl/sh for shell execution)
RUN apk add --no-cache ca-certificates git curl bash

# Copy the compiled Go binary
COPY --from=backend-builder /app/rune-server .

# Copy the built React static files
COPY --from=frontend-builder /app/web/dist ./web/dist

# Expose the API port
EXPOSE 8080

# Run the server
CMD ["./rune-server"]
