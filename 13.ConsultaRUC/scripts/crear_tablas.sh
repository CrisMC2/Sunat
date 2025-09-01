#!/bin/bash

# Configuración de la base de datos
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
#DB_HOST="localhost"
#DB_PORT="5433"
DB_HOST="192.168.18.16"
DB_PORT="5432"

# Ruta del archivo SQL
SQL_FILE="./database/prueba.sql"

# Verificar si el archivo SQL existe
if [ ! -f "$SQL_FILE" ]; then
    echo "Error: El archivo $SQL_FILE no existe"
    exit 1
fi

echo "Ejecutando script SQL: $SQL_FILE"
echo "Conectando a la base de datos: $DB_NAME en $DB_HOST:$DB_PORT"

# Ejecutar el script SQL
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $SQL_FILE

# Verificar si la ejecución fue exitosa
if [ $? -eq 0 ]; then
    echo "Script ejecutado exitosamente!"
else
    echo "Error al ejecutar el script SQL"
    exit 1
fi