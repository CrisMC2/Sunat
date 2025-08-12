#!/bin/bash

# =============================================================================
# Script simplificado para ejecutar go run . RUC uno por uno
# SIN LOTES - SIN ARCHIVOS TEMPORALES - SIN COMPLICACIONES
# =============================================================================

# Variables de configuración - BASE DE DATOS
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
DB_HOST="localhost"
DB_PORT="5433"
DATABASE_URL="postgres://postgres:admin123@localhost:5433/sunat?sslmode=disable"

# Directorio del proyecto Go
GO_PROJECT_DIR="$(pwd)/cmd/scraper-completo"

# Variables de control
TIMEOUT_SCRAPER=600    # 10 minutos por RUC
PAUSE_BETWEEN_RUCS=2   # 2 segundos entre RUCs
MAX_REINTENTOS=2       # Máximo 2 intentos por RUC

# Contadores
TOTAL_PROCESSED=0
TOTAL_SUCCESS=0
TOTAL_ERRORS=0
CURRENT_OFFSET=0

# Variable para cursor de RUC
LAST_RUC_PROCESSED=""

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# =============================================================================
# FUNCIONES DE LOGGING (integradas desde logs.sh)
# =============================================================================

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
        query="UPDATE log_consultas SET estado = '$estado', mensaje = '$mensaje_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    else
        query="UPDATE log_consultas SET estado = '$estado', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    fi
    
    # Ejecutar consulta
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
    
    return $?
}

# Función: marcar RUC como exitoso
mark_success() {
    local ruc="$1"
    local mensaje="$2"
    
    # Si no hay mensaje, usar mensaje por defecto
    if [ -z "$mensaje" ]; then
        mensaje="Scraping completado exitosamente"
    fi
    
    update_log_consultas "$ruc" "exitoso" "$mensaje"
}

# Función: marcar RUC como fallido
mark_failed() {
    local ruc="$1"
    local mensaje="$2"
    
    # Si no hay mensaje, usar mensaje por defecto
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

# =============================================================================
# FUNCIONES ORIGINALES
# =============================================================================

# Funciones simples
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Verificar dependencias básicas
check_dependencies() {
    print_info "Verificando dependencias..."
    
    if ! command -v psql &> /dev/null; then
        print_error "psql no está instalado"
        exit 1
    fi
    
    if ! command -v go &> /dev/null; then
        print_error "Go no está instalado"
        exit 1
    fi
    
    # Verificar conexión a BD
    PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        print_error "No se puede conectar a PostgreSQL"
        exit 1
    fi
    
    # Verificar directorio del proyecto
    if [ ! -f "$GO_PROJECT_DIR/main.go" ]; then
        print_error "No se encuentra main.go en: $GO_PROJECT_DIR"
        exit 1
    fi
    
    print_success "Dependencias verificadas"
}

# Obtener total de RUCs - RÁPIDO
get_total_rucs() {
    print_info "Obteniendo conteo aproximado..."
    local query="SELECT COUNT(*) FROM log_consultas WHERE estado IN ('pendiente', 'fallido');"
    
    TOTAL_RUCS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null | xargs)
    
    if [ $? -ne 0 ] || [ -z "$TOTAL_RUCS" ]; then
        print_error "Error al obtener conteo de RUCs"
        exit 1
    fi
    
    print_success "Aproximadamente $TOTAL_RUCS RUCs a procesar"
}

# Obtener SIGUIENTE RUC usando cursor (sin OFFSET)
get_next_ruc() {
    local query=""
    
    if [ -z "$LAST_RUC_PROCESSED" ]; then
        query="SELECT ruc FROM log_consultas WHERE estado IN ('pendiente', 'fallido') ORDER BY ruc LIMIT 1;"
    else
        query="SELECT ruc FROM log_consultas WHERE estado IN ('pendiente', 'fallido') AND ruc > '$LAST_RUC_PROCESSED' ORDER BY ruc LIMIT 1;"
    fi
    
    local ruc=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null | xargs)
    
    if [ $? -ne 0 ] || [ -z "$ruc" ]; then
        return 1
    fi
    
    echo "$ruc"
    return 0
}

# Ejecutar scraper para UN RUC
run_scraper() {
    local ruc="$1"
    local attempt="$2"
    
    cd "$GO_PROJECT_DIR" || {
        print_error "No se puede cambiar al directorio: $GO_PROJECT_DIR"
        return 1
    }
    
    export DATABASE_URL="$DATABASE_URL"
    
    print_info "[$((TOTAL_PROCESSED + 1))] Procesando RUC: $ruc (intento $attempt)"
    
    # Marcar como procesando en log_consultas
    mark_processing "$ruc" "$attempt"
    
    # Crear archivo temporal para capturar salida completa
    local temp_output="/tmp/scraper_output_$$"
    
    # Ejecutar go run . RUC y capturar tanto stdout como stderr
    timeout ${TIMEOUT_SCRAPER}s go run . "$ruc" > "$temp_output" 2>&1
    local exit_code=$?
    
    # Leer la salida completa del comando
    local raw_output=""
    if [ -f "$temp_output" ]; then
        raw_output=$(cat "$temp_output")
        rm -f "$temp_output"
    fi
    
    case $exit_code in
        0)
            print_success "✅ RUC $ruc procesado exitosamente"
            mark_success "$ruc" "$raw_output"
            return 0
            ;;
        124)
            print_error "⏱️  RUC $ruc: Timeout (>${TIMEOUT_SCRAPER}s)"
            mark_timeout "$ruc" "$TIMEOUT_SCRAPER"
            return 2
            ;;
        *)
            print_error "❌ RUC $ruc: Error (código: $exit_code)"
            mark_failed "$ruc" "$raw_output"
            return 1
            ;;
    esac
}

# Procesar UN RUC con reintentos
process_single_ruc() {
    local ruc="$1"
    
    for attempt in $(seq 1 $MAX_REINTENTOS); do
        run_scraper "$ruc" "$attempt"
        local result=$?
        
        if [ $result -eq 0 ]; then
            ((TOTAL_SUCCESS++))
            return 0
        fi
        
        # Si es timeout en el primer intento, no reintentar
        if [ $result -eq 2 ] && [ $attempt -eq 1 ]; then
            break
        fi
        
        # Si no es el último intento, esperar un poco
        if [ $attempt -lt $MAX_REINTENTOS ]; then
            print_warning "Reintentando en 3 segundos..."
            sleep 3
        fi
    done
    
    # Si llegamos aquí, falló
    ((TOTAL_ERRORS++))
    return 1
}

# Guardar progreso
save_progress() {
    echo "TOTAL_PROCESSED=$TOTAL_PROCESSED" > progreso.tmp
    echo "TOTAL_SUCCESS=$TOTAL_SUCCESS" >> progreso.tmp
    echo "TOTAL_ERRORS=$TOTAL_ERRORS" >> progreso.tmp
    echo "LAST_RUC_PROCESSED=$LAST_RUC_PROCESSED" >> progreso.tmp
    echo "FECHA=$(date)" >> progreso.tmp
}

# Reanudar desde progreso guardado
resume_progress() {
    if [ -f "progreso.tmp" ]; then
        print_warning "Se encontró progreso previo:"
        cat progreso.tmp
        echo
        read -p "¿Reanudar desde donde se quedó? (y/n): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            source progreso.tmp
            print_info "Reanudando desde RUC: $LAST_RUC_PROCESSED"
            return 0
        fi
    fi
    return 1
}

# Proceso principal - UNO POR UNO
process_all_rucs() {
    local start_time=$(date +%s)
    
    print_info "=== INICIANDO PROCESAMIENTO UNO POR UNO ==="
    print_info "Procesando RUCs secuencialmente hasta completar todos"
    print_info "Timeout por RUC: ${TIMEOUT_SCRAPER}s"
    print_info "Pausa entre RUCs: ${PAUSE_BETWEEN_RUCS}s"
    echo
    
    while true; do
        # Obtener el siguiente RUC usando cursor
        ruc=$(get_next_ruc)
        
        if [ $? -ne 0 ] || [ -z "$ruc" ]; then
            print_success "✅ No hay más RUCs pendientes - procesamiento completado!"
            break
        fi
        
        # Procesar este RUC
        process_single_ruc "$ruc"
        
        # Actualizar cursor y contadores
        LAST_RUC_PROCESSED="$ruc"
        ((TOTAL_PROCESSED++))
        
        # Guardar progreso cada 10 RUCs
        if [ $((TOTAL_PROCESSED % 10)) -eq 0 ]; then
            save_progress
            
            # Mostrar estadísticas
            elapsed=$(($(date +%s) - start_time))
            avg_time_per_ruc=$((elapsed / TOTAL_PROCESSED))
            success_rate=0
            if [ $TOTAL_PROCESSED -gt 0 ]; then
                success_rate=$(( (TOTAL_SUCCESS * 100) / TOTAL_PROCESSED ))
            fi
            
            print_info "=== PROGRESO ==="
            print_info "Procesados: $TOTAL_PROCESSED (${success_rate}% éxito)"
            print_info "Exitosos: $TOTAL_SUCCESS | Errores: $TOTAL_ERRORS"
            print_info "Tiempo promedio: ${avg_time_per_ruc}s por RUC"
            print_info "Último RUC: $LAST_RUC_PROCESSED"
            echo
        fi
        
        # Pausa entre RUCs
        sleep $PAUSE_BETWEEN_RUCS
    done
    
    # Resumen final
    total_time=$(($(date +%s) - start_time))
    print_success "=== PROCESAMIENTO COMPLETADO ==="
    print_success "Total procesados: $TOTAL_PROCESSED"
    print_success "Exitosos: $TOTAL_SUCCESS"
    print_success "Errores: $TOTAL_ERRORS"
    print_success "Tiempo total: $((total_time / 3600))h $((total_time % 3600 / 60))m"
    if [ $TOTAL_PROCESSED -gt 0 ]; then
        print_success "Tasa de éxito: $(( (TOTAL_SUCCESS * 100) / TOTAL_PROCESSED ))%"
        print_success "Promedio por RUC: $((total_time / TOTAL_PROCESSED))s"
    fi
}

# =============================================================================
# SCRIPT PRINCIPAL - SIMPLE Y DIRECTO
# =============================================================================

echo "======================================================================"
echo "🎯 SCRAPER SIMPLE - UNO POR UNO"
echo "======================================================================"
echo "✨ Sin lotes, sin archivos temporales, solo: go run . RUC"
echo

# Verificaciones básicas
check_dependencies

# Obtener total de RUCs
get_total_rucs

# Verificar si hay progreso previo
if ! resume_progress; then
    TOTAL_PROCESSED=0
    TOTAL_SUCCESS=0
    TOTAL_ERRORS=0
    LAST_RUC_PROCESSED=""
fi

# Mostrar configuración
print_info "=== CONFIGURACIÓN ==="
print_info "Comando: go run . RUC"
print_info "Directorio: $GO_PROJECT_DIR"
print_info "Timeout: ${TIMEOUT_SCRAPER}s por RUC"
print_info "Reintentos: $MAX_REINTENTOS"
print_info "Total aproximado: $TOTAL_RUCS RUCs"
if [ -n "$LAST_RUC_PROCESSED" ]; then
    print_info "Reanudando desde RUC: $LAST_RUC_PROCESSED"
fi
echo

# Confirmación
print_warning "⚠️  Procesando RUCs secuencialmente hasta completar todos los pendientes"
read -p "¿Continuar? (y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_error "Operación cancelada"
    exit 0
fi

# Procesar todos los RUCs uno por uno
process_all_rucs

print_success "🎉 ¡Proceso completado!"
print_success "📊 Los datos están en la tabla 'ruc_completo'"

# Limpiar archivo de progreso temporal
rm -f progreso.tmp

echo
print_info "=== COMANDOS ÚTILES ==="
print_info "Ver registros en BD: psql '$DATABASE_URL' -c 'SELECT COUNT(*) FROM ruc_completo;'"
print_info "Ver últimos procesados: psql '$DATABASE_URL' -c 'SELECT ruc, razon_social FROM ruc_completo ORDER BY fecha_consulta DESC LIMIT 5;'"