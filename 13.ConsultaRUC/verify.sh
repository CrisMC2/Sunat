#!/bin/bash

# ============================================
# SCRIPT DE VERIFICACIÓN DE DATOS IMPORTADOS
# ============================================

# Configuración de la base de datos
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
DB_HOST="localhost"
DB_PORT="5433"

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para ejecutar consulta y mostrar resultado
run_query() {
    local query="$1"
    local title="$2"
    
    export PGPASSWORD="$DB_PASSWORD"
    
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$title${NC}"
    echo -e "${BLUE}========================================${NC}"
    
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query"
}

echo -e "${GREEN}🔍 VERIFICANDO DATOS IMPORTADOS${NC}"
echo -e "${GREEN}================================${NC}"

# 1. Contar registros totales por tabla
run_query "
SELECT 
    'ruc_info' as tabla, 
    COUNT(*) as total_registros,
    COUNT(DISTINCT ruc) as rucs_unicos
FROM ruc_info
UNION ALL
SELECT 
    'ruc_actividades_economicas' as tabla, 
    COUNT(*) as total_registros,
    COUNT(DISTINCT ruc) as rucs_unicos
FROM ruc_actividades_economicas
UNION ALL
SELECT 
    'ruc_comprobantes_electronicos' as tabla, 
    COUNT(*) as total_registros,
    COUNT(DISTINCT ruc) as rucs_unicos
FROM ruc_comprobantes_electronicos
UNION ALL
SELECT 
    'representante_legal' as tabla, 
    COUNT(*) as total_registros,
    COUNT(DISTINCT ruc) as rucs_unicos
FROM representante_legal
ORDER BY tabla;
" "📊 RESUMEN GENERAL DE REGISTROS"

# 2. Mostrar algunos RUCs de ejemplo
run_query "
SELECT 
    ruc,
    razon_social,
    estado,
    condicion,
    fecha_inscripcion,
    updated_at
FROM ruc_info 
ORDER BY updated_at DESC 
LIMIT 5;
" "📋 ÚLTIMOS 5 RUCs INSERTADOS"

# 3. Verificar actividades económicas
run_query "
SELECT 
    ri.ruc,
    ri.razon_social,
    COUNT(ae.actividad_economica) as cant_actividades
FROM ruc_info ri
LEFT JOIN ruc_actividades_economicas ae ON ri.ruc = ae.ruc
GROUP BY ri.ruc, ri.razon_social
HAVING COUNT(ae.actividad_economica) > 0
ORDER BY cant_actividades DESC
LIMIT 5;
" "🏭 RUCs CON MÁS ACTIVIDADES ECONÓMICAS"

# 4. Verificar representantes legales
run_query "
SELECT 
    ri.ruc,
    ri.razon_social,
    rl.nombre_completo,
    rl.cargo,
    rl.vigente
FROM ruc_info ri
INNER JOIN representante_legal rl ON ri.ruc = rl.ruc
WHERE rl.vigente = true
ORDER BY ri.updated_at DESC
LIMIT 5;
" "👥 REPRESENTANTES LEGALES VIGENTES"

# 5. Verificar estados y condiciones
run_query "
SELECT 
    estado,
    condicion,
    COUNT(*) as cantidad
FROM ruc_info
GROUP BY estado, condicion
ORDER BY cantidad DESC;
" "📈 DISTRIBUCIÓN POR ESTADO Y CONDICIÓN"

# 6. Verificar fechas nulas o problemáticas
run_query "
SELECT 
    COUNT(*) as total_registros,
    COUNT(fecha_inscripcion) as con_fecha_inscripcion,
    COUNT(fecha_inicio_actividades) as con_fecha_inicio,
    COUNT(*) - COUNT(fecha_inscripcion) as sin_fecha_inscripcion,
    COUNT(*) - COUNT(fecha_inicio_actividades) as sin_fecha_inicio
FROM ruc_info;
" "📅 VERIFICACIÓN DE FECHAS"

# 7. Buscar posibles errores en datos
run_query "
SELECT 
    'Campos vacíos en razón social' as problema,
    COUNT(*) as cantidad
FROM ruc_info 
WHERE razon_social IS NULL OR razon_social = '' OR razon_social = '-'
UNION ALL
SELECT 
    'RUCs con formato incorrecto' as problema,
    COUNT(*) as cantidad
FROM ruc_info 
WHERE LENGTH(ruc) != 11 OR ruc !~ '^[0-9]+$'
UNION ALL
SELECT 
    'Estados no estándar' as problema,
    COUNT(*) as cantidad
FROM ruc_info 
WHERE estado NOT IN ('ACTIVO', 'INACTIVO', 'SUSPENDIDO');
" "⚠️ POSIBLES PROBLEMAS EN DATOS"

# 8. Verificar un RUC específico con todos sus datos relacionados
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}🔍 DETALLE COMPLETO DE UN RUC EJEMPLO${NC}"
echo -e "${BLUE}========================================${NC}"

# Obtener un RUC para mostrar como ejemplo
RUC_EJEMPLO=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT ruc FROM ruc_info LIMIT 1;" 2>/dev/null | tr -d ' ')

if [[ -n "$RUC_EJEMPLO" ]]; then
    echo -e "${YELLOW}Mostrando datos completos para RUC: $RUC_EJEMPLO${NC}"
    
    run_query "
    SELECT 
        'INFORMACIÓN BÁSICA' as seccion,
        ruc,
        razon_social,
        tipo_contribuyente,
        estado,
        condicion,
        domicilio_fiscal
    FROM ruc_info 
    WHERE ruc = '$RUC_EJEMPLO';
    " "INFORMACIÓN BÁSICA"
    
    run_query "
    SELECT 
        'ACTIVIDAD ECONÓMICA' as tipo,
        actividad_economica
    FROM ruc_actividades_economicas 
    WHERE ruc = '$RUC_EJEMPLO';
    " "ACTIVIDADES ECONÓMICAS"
    
    run_query "
    SELECT 
        'COMPROBANTE ELECTRÓNICO' as tipo,
        comprobante_electronico
    FROM ruc_comprobantes_electronicos 
    WHERE ruc = '$RUC_EJEMPLO';
    " "COMPROBANTES ELECTRÓNICOS"
    
    run_query "
    SELECT 
        nombre_completo,
        cargo,
        tipo_documento,
        numero_documento,
        vigente
    FROM representante_legal 
    WHERE ruc = '$RUC_EJEMPLO';
    " "REPRESENTANTES LEGALES"
else
    echo -e "${RED}No se encontraron RUCs en la base de datos${NC}"
fi

echo -e "\n${GREEN}✅ VERIFICACIÓN COMPLETADA${NC}"
echo -e "${GREEN}=========================${NC}"