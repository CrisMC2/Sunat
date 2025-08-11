#!/bin/bash

echo "=== VERIFICACIÓN RÁPIDA DE RUCs ==="
echo "Solo información básica"
echo ""

export PATH="/usr/local/go/bin:$PATH"

# Procesar cada RUC individualmente
while IFS= read -r ruc
do
    if [[ ! -z "$ruc" ]]; then
        echo "Procesando: $ruc"
        go run cmd/scraper/main.go "$ruc"
        echo "---"
        sleep 1
    fi
done < rucs_test.txt

echo ""
echo "✅ Verificación completada"
echo ""
echo "Archivos generados:"
ls -la ruc_*.json