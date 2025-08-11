#!/bin/bash

# Test rápido con solo algunos RUCs
echo "=== TEST RÁPIDO DEL SCRAPER ==="
echo ""
echo "Probando con 3 RUCs de cada tipo..."
echo ""

# Crear archivo temporal con RUCs de prueba
cat > rucs_prueba.txt << EOF
20606454466
20393261162
20600656288
10719706288
10775397131
10420242986
EOF

# Ejecutar test
export PATH="/usr/local/go/bin:$PATH"
go run cmd/test-masivo/main.go rucs_prueba.txt

# Limpiar
rm rucs_prueba.txt