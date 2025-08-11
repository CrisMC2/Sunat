#!/bin/bash

# Configuración
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
DB_PORT=5433
TXT_FILE="rucs_test.txt"

echo "⏳ Insertando RUCs en la tabla 'ruc_pruebas'..."

# Convertimos el txt en un script SQL temporal
TMP_SQL="tmp_insert_ruc_pruebas.sql"
echo "-- Insertando datos en ruc_pruebas" > "$TMP_SQL"

while IFS= read -r ruc; do
    echo "INSERT INTO ruc_pruebas (ruc) VALUES ('$ruc') ON CONFLICT (ruc) DO NOTHING;" >> "$TMP_SQL"
done < "$TXT_FILE"

# Ejecutamos el script
PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h localhost -p "$DB_PORT" -f "$TMP_SQL"

if [ $? -eq 0 ]; then
    echo "✅ RUCs insertados exitosamente en la tabla 'ruc_pruebas'."
    rm "$TMP_SQL"  # Limpieza del archivo temporal
else
    echo "❌ Error al insertar los RUCs."
fi