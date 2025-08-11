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
export ROD_BROWSER_BIN="/snap/bin/chromium"

# Show menu
echo "==================================="
echo "  SUNAT RUC Scraper - Completo"
echo "==================================="
echo ""
echo "Opciones:"
echo "1. Consulta básica (solo información principal)"
echo "2. Consulta completa (toda la información disponible)"
echo ""
read -p "Seleccione una opción (1 o 2): " option

# Get RUC
if [ $# -eq 0 ]; then
    read -p "Ingrese el RUC a consultar: " ruc
else
    ruc=$1
fi

# Download dependencies if needed
echo "Verificando dependencias..."
$GO_BIN mod download

# Run the appropriate scraper
case $option in
    1)
        echo "Ejecutando consulta básica..."
        $GO_BIN run cmd/scraper/main.go $ruc
        ;;
    2)
        echo "Ejecutando consulta completa..."
        echo "Esto puede tomar varios minutos ya que se consultarán todos los servicios disponibles..."
        $GO_BIN run cmd/scraper-completo/main.go $ruc
        ;;
    *)
        echo "Opción inválida. Ejecutando consulta básica por defecto..."
        $GO_BIN run cmd/scraper/main.go $ruc
        ;;
esac