#!/bin/bash

# Variables (ajusta según tu configuración)
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
#DB_HOST="localhost"
#DB_PORT="5433"
DB_HOST="192.168.18.16"
DB_PORT="5432"

# Ejecutar el archivo SQL
echo "⏳ Ejecutando el script para crear la tabla 'ruc_pruebas' en la base de datos '$DB_NAME'..."

PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h localhost -p 5433 -f "$SCHEMA_FILE"

if [ $? -eq 0 ]; then
    echo "✅ Tabla 'ruc_pruebas' creada exitosamente."
else
    echo "❌ Ocurrió un error al crear la tabla 'ruc_pruebas'."
fi