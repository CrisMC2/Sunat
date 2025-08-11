#!/bin/bash

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker from https://docker.com"
    exit 1
fi

# Build and run with Docker
echo "Building Docker image..."
docker build -t sunat-scraper .

echo "Running scraper..."
if [ $# -eq 0 ]; then
    # Run with default RUC
    docker run --rm -v "$(pwd)":/app/output sunat-scraper
else
    # Run with provided RUCs
    docker run --rm -v "$(pwd)":/app/output sunat-scraper "$@"
fi