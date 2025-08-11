#!/bin/bash

# Check if Go is installed
GO_BIN=""
if command -v go &> /dev/null; then
    GO_BIN="go"
elif [ -x "/usr/local/go/bin/go" ]; then
    GO_BIN="/usr/local/go/bin/go"
else
    echo "Go is not installed. Please install Go from https://golang.org/dl/"
    exit 1
fi

# Export Go path for this session
export PATH="/usr/local/go/bin:$PATH"

# Download dependencies
echo "Downloading dependencies..."
$GO_BIN mod download

# Run the scraper
echo "Running scraper..."
$GO_BIN run cmd/scraper/main.go $@