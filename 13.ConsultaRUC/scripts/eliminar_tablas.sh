#!/bin/bash

# Variables (ajusta según tu configuración)
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
#DB_HOST="localhost"
#DB_PORT="5433"
DB_HOST="192.168.18.16"
DB_PORT="5432"

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}⚠️  ATENCIÓN: Este script eliminará TODAS las tablas de la base de datos '$DB_NAME' excepto 'empresas_sunat'${NC}"
echo -e "${RED}⚠️  TODOS los datos de las demás tablas se perderán permanentemente${NC}"
echo
read -p "¿Estás seguro de continuar? Escribe 'SI' para confirmar: " confirmation

if [ "$confirmation" != "SI" ]; then
    echo -e "${GREEN}Operación cancelada${NC}"
    exit 0
fi

echo -e "${YELLOW}🗑️  Eliminando todas las tablas excepto 'empresas_sunat'...${NC}"

# Obtener todas las tablas en esquema public excepto empresas_sunat
TABLES=$(PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -p "$DB_PORT" -t -c "
SELECT tablename FROM pg_tables
WHERE schemaname = 'public' AND tablename <> 'empresas_sunat';
")

if [ -z "$TABLES" ]; then
    echo -e "${GREEN}No hay tablas para eliminar excepto 'empresas_sunat'${NC}"
    exit 0
fi

# Recorrer las tablas y eliminarlas
for TABLE in $TABLES; do
    echo "Eliminando tabla: $TABLE"
    PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -p "$DB_PORT" -c "DROP TABLE IF EXISTS public.\"$TABLE\" CASCADE;"
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error eliminando la tabla $TABLE${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✅ Tablas eliminadas correctamente, excepto 'empresas_sunat'${NC}"

# Verificar tablas restantes
TABLE_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -p "$DB_PORT" -t -c "SELECT COUNT(*) FROM pg_tables WHERE schemaname = 'public';")
echo "📊 Tablas restantes en 'public': $(echo $TABLE_COUNT | xargs)"
