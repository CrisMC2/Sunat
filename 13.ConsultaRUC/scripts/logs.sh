#!/bin/bash

# =============================================================================
# Script para manejar logs en la tabla log_consultas
# Ubicación: ./scripts/logs.sh
# =============================================================================

# Variables de configuración de la base de datos
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
NC='\033[0m'

# Función para logging con colores
log_info() {
    echo -e "${BLUE}[LOGS]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[LOGS]${NC} $1"
}

log_error() {
    echo -e "${RED}[LOGS]${NC} $1"
}

# Función para escapar caracteres especiales en SQL
escape_sql() {
    echo "$1" | sed "s/'/''/g"
}

# Función principal: actualizar estado en log_consultas
update_log_consultas() {
    local ruc="$1"
    local estado="$2"
    local mensaje="$3"
    
    # Validar parámetros
    if [ -z "$ruc" ] || [ -z "$estado" ]; then
        log_error "Parámetros inválidos: RUC y estado son requeridos"
        return 1
    fi
    
    # Escapar mensaje para SQL
    local mensaje_escaped=""
    if [ -n "$mensaje" ]; then
        mensaje_escaped=$(escape_sql "$mensaje")
    fi
    
    # Preparar consulta SQL
    local query=""
    if [ -n "$mensaje" ]; then
        query="
            UPDATE log_consultas 
            SET estado = '$estado', 
                mensaje = '$mensaje_escaped',
                fecha_registro = CURRENT_TIMESTAMP 
            WHERE ruc = '$ruc';
        "
    else
        query="
            UPDATE log_consultas 
            SET estado = '$estado',
                fecha_registro = CURRENT_TIMESTAMP 
            WHERE ruc = '$ruc';
        "
    fi
    
    # Ejecutar consulta
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
    
    if [ $? -eq 0 ]; then
        log_success "Estado actualizado para RUC $ruc: $estado"
        return 0
    else
        log_error "Error actualizando estado para RUC $ruc"
        return 1
    fi
}

# Función: marcar RUC como exitoso
mark_success() {
    local ruc="$1"
    local mensaje="${2:-Scraping completado exitosamente}"
    
    update_log_consultas "$ruc" "exitoso" "$mensaje"
}

# Función: marcar RUC como fallido
mark_failed() {
    local ruc="$1"
    local mensaje="$2"
    
    if [ -z "$mensaje" ]; then
        mensaje="Error durante el scraping"
    fi
    
    update_log_consultas "$ruc" "fallido" "$mensaje"
}

# Función: marcar RUC como en proceso
mark_processing() {
    local ruc="$1"
    local intento="${2:-1}"
    local mensaje="Procesando... (intento $intento)"
    
    update_log_consultas "$ruc" "procesando" "$mensaje"
}

# Función: marcar RUC como timeout
mark_timeout() {
    local ruc="$1"
    local timeout_duration="$2"
    local mensaje="Timeout después de ${timeout_duration}s"
    
    update_log_consultas "$ruc" "fallido" "$mensaje"
}

# Función para obtener estadísticas
get_stats() {
    local query="
        SELECT 
            estado,
            COUNT(*) as cantidad
        FROM log_consultas 
        GROUP BY estado 
        ORDER BY cantidad DESC;
    "
    
    log_info "=== ESTADÍSTICAS DE CONSULTAS ==="
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query"
}

# Función para obtener últimos registros
get_recent_logs() {
    local limit="${1:-10}"
    local query="
        SELECT 
            ruc,
            estado,
            mensaje,
            fecha_registro
        FROM log_consultas 
        ORDER BY fecha_registro DESC 
        LIMIT $limit;
    "
    
    log_info "=== ÚLTIMOS $limit REGISTROS ==="
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query"
}

# Función para verificar conexión a BD
test_connection() {
    PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" >/dev/null 2>&1
    if [ $? -eq 0 ]; then
        log_success "Conexión a base de datos OK"
        return 0
    else
        log_error "Error de conexión a base de datos"
        return 1
    fi
}

# Función principal si se ejecuta directamente
main() {
    case "${1:-help}" in
        "success")
            mark_success "$2" "$3"
            ;;
        "failed")
            mark_failed "$2" "$3"
            ;;
        "processing")
            mark_processing "$2" "$3"
            ;;
        "timeout")
            mark_timeout "$2" "$3"
            ;;
        "stats")
            get_stats
            ;;
        "recent")
            get_recent_logs "$2"
            ;;
        "test")
            test_connection
            ;;
        "help"|*)
            echo "=== SCRIPT DE LOGGING PARA log_consultas ==="
            echo "Uso: $0 [comando] [ruc] [mensaje]"
            echo
            echo "Comandos disponibles:"
            echo "  success RUC [mensaje]    - Marcar como exitoso"
            echo "  failed RUC mensaje       - Marcar como fallido"
            echo "  processing RUC [intento] - Marcar como procesando"
            echo "  timeout RUC segundos     - Marcar como timeout"
            echo "  stats                    - Ver estadísticas"
            echo "  recent [N]               - Ver últimos N registros (default: 10)"
            echo "  test                     - Probar conexión a BD"
            echo
            echo "Ejemplos:"
            echo "  $0 success 20606316977 'Scraping completado'"
            echo "  $0 failed 20606316977 'Error de conexión'"
            echo "  $0 processing 20606316977 2"
            echo "  $0 timeout 20606316977 120"
            echo "  $0 stats"
            echo "  $0 recent 5"
            ;;
    esac
}

# Ejecutar función principal si se llama directamente
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi