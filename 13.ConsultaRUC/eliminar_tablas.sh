#!/bin/bash

# Variables (ajusta según tu configuración)
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"  # pon tu contraseña real
DB_HOST="localhost"
DB_PORT="5433"

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}⚠️  ATENCIÓN: Este script eliminará TODAS las tablas de la base de datos '$DB_NAME'${NC}"
echo -e "${RED}⚠️  TODOS los datos se perderán permanentemente${NC}"
echo
read -p "¿Estás seguro de continuar? Escribe 'SI' para confirmar: " confirmation

if [ "$confirmation" != "SI" ]; then
    echo -e "${GREEN}Operación cancelada${NC}"
    exit 0
fi

echo -e "${YELLOW}🗑️  Eliminando todas las tablas de la base de datos '$DB_NAME'...${NC}"

# Eliminar todas las tablas
PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -p "$DB_PORT" -c "
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO $DB_USER;
GRANT ALL ON SCHEMA public TO public;
"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Todas las tablas han sido eliminadas exitosamente${NC}"
    
    # Verificar que no queden tablas
    TABLE_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -U "$DB_USER" -d "$DB_NAME" -h "$DB_HOST" -p "$DB_PORT" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
    echo "📊 Tablas restantes: $(echo $TABLE_COUNT | xargs)"
else
    echo -e "${RED}❌ Error al eliminar las tablas${NC}"
    exit 1
fi