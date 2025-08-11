#!/bin/bash

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker from https://docker.com"
    exit 1
fi

echo "✅ Docker encontrado en: $(command -v docker)"
echo "🔧 Construyendo imagen Docker..."
docker build -t sunat-scraper .

echo "🚀 Ejecutando scraper dentro de Docker..."
if [ $# -eq 0 ]; then
    echo "ℹ️ Ejecutando sin RUC específico (modo interactivo o predeterminado)"
    docker run --rm -v "$(pwd)/output":/app/output sunat-scraper
else
    echo "🧾 Ejecutando con RUC(s): $@"
    docker run --rm -v "$(pwd)/output":/app/output sunat-scraper "$@"
fi
