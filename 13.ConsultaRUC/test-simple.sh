#!/bin/bash

# Test simple con un solo RUC para verificar funcionalidad
echo "=== TEST SIMPLE - UN SOLO RUC ==="
echo ""

export PATH="/usr/local/go/bin:$PATH"

# Probar con un RUC jurídico
echo "Probando con RUC jurídico: 20606454466"
go run cmd/scraper/main.go 20606454466

echo ""
echo "--- Verificando archivo generado ---"
if [ -f "output/ruc_20606454466.json" ]; then
    echo "✅ Archivo generado correctamente"
    echo "Contenido:"
    cat output/ruc_20606454466.json | head -20
else
    echo "❌ No se generó el archivo"
fi