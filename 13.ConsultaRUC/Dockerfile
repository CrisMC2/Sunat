FROM golang:1.21-alpine AS builder

# Install chromium and dependencies
RUN apk add --no-cache chromium chromium-chromedriver

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o scraper cmd/scraper/main.go

FROM alpine:latest

# Install chromium
RUN apk add --no-cache chromium chromium-chromedriver

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/scraper .

# Set environment variables for Rod
ENV ROD_BROWSER_PATH=/usr/bin/chromium

ENTRYPOINT ["./scraper"]