#!/bin/bash

# =============================================================================
# Script para ejecutar go run . RUC con 10 procesos en paralelo
# CON CONTROL DE CONCURRENCIA MEJORADO
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

# Variables de control paralelo
MAX_PARALLEL_JOBS=10      # Número máximo de procesos simultáneos
TIMEOUT_SCRAPER=600       # 10 minutos por RUC
PAUSE_BETWEEN_BATCHES=5   # 5 segundos entre lotes
MAX_REINTENTOS=2          # Máximo 2 intentos por RUC
BATCH_SIZE=50             # Tamaño del lote para procesar

# Contadores globales (con archivos para sincronización)
STATS_FILE="/tmp/scraper_stats_$$"
LOCK_FILE="/tmp/scraper_lock_$$"
JOBS_DIR="/tmp/scraper_jobs_$$"
ACTIVE_PIDS_FILE="/tmp/scraper_pids_$$"

# Crear directorio para jobs
mkdir -p "$JOBS_DIR"

# Inicializar contadores
echo "TOTAL_PROCESSED=0" > "$STATS_FILE"
echo "TOTAL_SUCCESS=0" >> "$STATS_FILE"
echo "TOTAL_ERRORS=0" >> "$STATS_FILE"

# Inicializar archivo de PIDs activos
echo "" > "$ACTIVE_PIDS_FILE"

# Variable para cursor de RUC
LAST_RUC_PROCESSED=""

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# =============================================================================
# FUNCIONES DE LOGGING Y CONTROL
# =============================================================================

# Lock para operaciones críticas con timeout mejorado
acquire_lock() {
    local max_attempts=100
    local attempt=0
    
    while ! mkdir "$LOCK_FILE" 2>/dev/null; do
        sleep 0.05
        ((attempt++))
        if [ $attempt -ge $max_attempts ]; then
            echo "ERROR: No se pudo obtener el lock después de $max_attempts intentos" >&2
            return 1
        fi
    done
    return 0
}

release_lock() {
    rmdir "$LOCK_FILE" 2>/dev/null || true
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
    
    [ -z "$ruc" ] || [ -z "$estado" ] && return 1
    
    local mensaje_escaped=""
    [ -n "$mensaje" ] && mensaje_escaped=$(escape_sql "$mensaje")
    
    local query=""
    if [ -n "$mensaje" ]; then
        query="UPDATE log_consultas SET estado = '$estado', mensaje = '$mensaje_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    else
        query="UPDATE log_consultas SET estado = '$estado', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    fi
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
}

# Funciones de marcado de estado
mark_success() {
    update_log_consultas "$1" "exitoso" "${2:-Scraping completado}"
}

mark_failed() {
    update_log_consultas "$1" "fallido" "${2:-Error durante el scraping}"
}

mark_processing() {
    local mensaje="Worker-${2:-1} procesando (intento ${3:-1})"
    update_log_consultas "$1" "procesando" "$mensaje"
}

mark_timeout() {
    local mensaje="Timeout después de ${2}s"
    update_log_consultas "$1" "fallido" "$mensaje"
}

# Actualizar estadísticas globales de forma segura
update_stats() {
    local operation="$1"
    
    acquire_lock || return 1
    
    local TOTAL_PROCESSED=0
    local TOTAL_SUCCESS=0
    local TOTAL_ERRORS=0
    
    [ -f "$STATS_FILE" ] && source "$STATS_FILE"
    
    case "$operation" in
        "processed") TOTAL_PROCESSED=$((TOTAL_PROCESSED + 1)) ;;
        "success") TOTAL_SUCCESS=$((TOTAL_SUCCESS + 1)) ;;
        "error") TOTAL_ERRORS=$((TOTAL_ERRORS + 1)) ;;
    esac
    
    echo "TOTAL_PROCESSED=$TOTAL_PROCESSED" > "$STATS_FILE"
    echo "TOTAL_SUCCESS=$TOTAL_SUCCESS" >> "$STATS_FILE"
    echo "TOTAL_ERRORS=$TOTAL_ERRORS" >> "$STATS_FILE"
    
    release_lock
}

# Funciones de output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_worker() {
    echo -e "${CYAN}[W-$1]${NC} $2"
}

# Verificar dependencias
check_dependencies() {
    print_info "Verificando dependencias..."
    
    command -v psql &> /dev/null || { print_error "psql no está instalado"; exit 1; }
    command -v go &> /dev/null || { print_error "Go no está instalado"; exit 1; }
    
    PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" > /dev/null 2>&1 || {
        print_error "No se puede conectar a PostgreSQL"
        exit 1
    }
    
    [ -f "$GO_PROJECT_DIR/main.go" ] || {
        print_error "No se encuentra main.go en: $GO_PROJECT_DIR"
        exit 1
    }
    
    print_success "Dependencias OK"
}

# Obtener lote de RUCs
get_ruc_batch() {
    local batch_size="$1"
    local query=""
    
    if [ -z "$LAST_RUC_PROCESSED" ]; then
        query="SELECT ruc FROM log_consultas WHERE estado NOT IN ('exitoso', 'procesando') ORDER BY ruc DESC LIMIT $batch_size;"
    else
        query="SELECT ruc FROM log_consultas WHERE estado NOT IN ('exitoso', 'procesando') AND ruc < '$LAST_RUC_PROCESSED' ORDER BY ruc DESC LIMIT $batch_size;"
    fi
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null | sed 's/^[ \t]*//;s/[ \t]*$//' | grep -v '^$'
}

# Registrar PID activo
register_pid() {
    local pid="$1"
    acquire_lock || return 1
    echo "$pid" >> "$ACTIVE_PIDS_FILE"
    release_lock
}

# Desregistrar PID
unregister_pid() {
    local pid="$1"
    acquire_lock || return 1
    grep -v "^$pid$" "$ACTIVE_PIDS_FILE" > "${ACTIVE_PIDS_FILE}.tmp" 2>/dev/null || touch "${ACTIVE_PIDS_FILE}.tmp"
    mv "${ACTIVE_PIDS_FILE}.tmp" "$ACTIVE_PIDS_FILE"
    release_lock
}

# Contar procesos activos de forma confiable
count_active_processes() {
    acquire_lock || return 1
    
    local active_count=0
    local temp_file="${ACTIVE_PIDS_FILE}.clean"
    
    : > "$temp_file"
    
    if [ -f "$ACTIVE_PIDS_FILE" ]; then
        while IFS= read -r pid; do
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                echo "$pid" >> "$temp_file"
                ((active_count++))
            fi
        done < "$ACTIVE_PIDS_FILE"
    fi
    
    mv "$temp_file" "$ACTIVE_PIDS_FILE"
    release_lock
    
    echo $active_count
}

# Worker function mejorada
worker_function() {
    local worker_id="$1"
    local ruc="$2"
    local attempt="${3:-1}"
    
    cd "$GO_PROJECT_DIR" || {
        print_worker "$worker_id" "ERROR: No se puede cambiar al directorio"
        return 1
    }
    
    export DATABASE_URL="$DATABASE_URL"
    
    print_worker "$worker_id" "Procesando RUC: $ruc (intento $attempt)"
    mark_processing "$ruc" "$worker_id" "$attempt"
    
    # Ejecutar con timeout
    local temp_output="/tmp/worker_${worker_id}_${ruc}_$$"
    timeout ${TIMEOUT_SCRAPER}s go run . "$ruc" > "$temp_output" 2>&1
    local exit_code=$?
    
    # Leer salida limitada
    local output=""
    [ -f "$temp_output" ] && output=$(head -c 200 "$temp_output" 2>/dev/null)
    rm -f "$temp_output"
    
    case $exit_code in
        0)
            print_worker "$worker_id" "OK: RUC $ruc"
            mark_success "$ruc"
            update_stats "success"
            return 0
            ;;
        124)
            print_worker "$worker_id" "TIMEOUT: RUC $ruc"
            mark_timeout "$ruc" "$TIMEOUT_SCRAPER"
            update_stats "error"
            return 2
            ;;
        *)
            print_worker "$worker_id" "ERROR: RUC $ruc (código: $exit_code)"
            mark_failed "$ruc" "Error $exit_code"
            update_stats "error"
            return 1
            ;;
    esac
}

# Procesar RUC con reintentos
process_ruc_with_retries() {
    local worker_id="$1"
    local ruc="$2"
    
    for attempt in $(seq 1 $MAX_REINTENTOS); do
        worker_function "$worker_id" "$ruc" "$attempt"
        local result=$?
        
        [ $result -eq 0 ] && return 0
        
        # No reintentar timeouts
        [ $result -eq 2 ] && [ $attempt -eq 1 ] && break
        
        # Pausa antes del siguiente intento
        [ $attempt -lt $MAX_REINTENTOS ] && sleep 3
    done
    
    return 1
}

# Esperar hasta que haya espacio para un nuevo worker
wait_for_slot() {
    while [ $(count_active_processes) -ge $MAX_PARALLEL_JOBS ]; do
        sleep 0.5
    done
}

# Procesar lote con control estricto de concurrencia
process_batch_parallel() {
    local rucs_string="$1"
    local batch_num="$2"
    
    read -ra rucs <<< "$rucs_string"
    
    [ ${#rucs[@]} -eq 0 ] && return 1
    
    print_info "Lote $batch_num: ${#rucs[@]} RUCs"
    local batch_start_time=$(date +%s)
    
    local worker_id=1
    
    for ruc in "${rucs[@]}"; do
        # Esperar slot disponible
        wait_for_slot
        
        # Verificar límite una vez más antes de crear proceso
        local current_count=$(count_active_processes)
        if [ $current_count -ge $MAX_PARALLEL_JOBS ]; then
            print_warning "Límite alcanzado ($current_count/$MAX_PARALLEL_JOBS), esperando..."
            wait_for_slot
        fi
        
        # Crear worker en background
        (
            local my_pid=$$
            register_pid "$my_pid"
            
            # Trap para asegurar limpieza
            trap "unregister_pid $my_pid" EXIT
            
            process_ruc_with_retries "$worker_id" "$ruc"
            update_stats "processed"
        ) &
        
        local job_pid=$!
        register_pid "$job_pid"
        
        ((worker_id++))
        
        # Pausa mínima para evitar race conditions
        sleep 0.1
    done
    
    # Esperar a que terminen todos los trabajos del lote
    print_info "Esperando workers del lote $batch_num..."
    
    local last_report=0
    while [ $(count_active_processes) -gt 0 ]; do
        local current_time=$(date +%s)
        if [ $((current_time - last_report)) -ge 30 ]; then
            local active=$(count_active_processes)
            [ -f "$STATS_FILE" ] && source "$STATS_FILE"
            print_info "Activos: $active | Procesados: $TOTAL_PROCESSED | OK: $TOTAL_SUCCESS | Errores: $TOTAL_ERRORS"
            last_report=$current_time
        fi
        sleep 2
    done
    
    # Actualizar cursor
    [ ${#rucs[@]} -gt 0 ] && LAST_RUC_PROCESSED="${rucs[-1]}"
    
    local batch_duration=$(($(date +%s) - batch_start_time))
    [ -f "$STATS_FILE" ] && source "$STATS_FILE"
    print_success "Lote $batch_num completado en ${batch_duration}s (Procesados=$TOTAL_PROCESSED)"
    
    return 0
}

# Proceso principal
process_all_rucs_parallel() {
    local start_time=$(date +%s)
    local batch_num=1
    
    print_info "=== INICIANDO PROCESAMIENTO PARALELO ==="
    print_info "Workers máx: $MAX_PARALLEL_JOBS | Lote: $BATCH_SIZE | Timeout: ${TIMEOUT_SCRAPER}s"
    
    while true; do
        print_info "Obteniendo lote $batch_num..."
        
        local rucs_raw=$(get_ruc_batch "$BATCH_SIZE")
        
        if [ -z "$rucs_raw" ]; then
            print_success "No hay más RUCs pendientes"
            break
        fi
        
        local rucs=()
        while IFS= read -r line; do
            [ -n "$line" ] && rucs+=("$line")
        done <<< "$rucs_raw"
        
        if [ ${#rucs[@]} -eq 0 ]; then
            print_success "No hay más RUCs válidos"
            break
        fi
        
        process_batch_parallel "${rucs[*]}" "$batch_num"
        
        ((batch_num++))
        
        [ $PAUSE_BETWEEN_BATCHES -gt 0 ] && sleep $PAUSE_BETWEEN_BATCHES
    done
    
    # Resumen final
    local total_time=$(($(date +%s) - start_time))
    
    if [ -f "$STATS_FILE" ]; then
        source "$STATS_FILE"
        
        print_success "=== PROCESAMIENTO COMPLETADO ==="
        print_success "Total procesados: $TOTAL_PROCESSED"
        print_success "Exitosos: $TOTAL_SUCCESS"
        print_success "Errores: $TOTAL_ERRORS"
        
        local hours=$((total_time / 3600))
        local minutes=$(( (total_time % 3600) / 60 ))
        local seconds=$((total_time % 60))
        print_success "Tiempo total: ${hours}h ${minutes}m ${seconds}s"
        
        if [ $TOTAL_PROCESSED -gt 0 ]; then
            local success_rate=$(( (TOTAL_SUCCESS * 100) / TOTAL_PROCESSED ))
            print_success "Tasa de éxito: ${success_rate}%"
            
            local avg_per_ruc=$((total_time / TOTAL_PROCESSED))
            print_success "Promedio por RUC: ${avg_per_ruc}s"
            
            [ $total_time -gt 0 ] && {
                local rucs_per_minute=$(( (TOTAL_PROCESSED * 60) / total_time ))
                print_success "RUCs por minuto: ${rucs_per_minute}"
            }
        fi
    fi
}

# Kill all active processes
kill_all_workers() {
    print_info "Terminando workers activos..."
    
    if [ -f "$ACTIVE_PIDS_FILE" ]; then
        while IFS= read -r pid; do
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                kill -TERM "$pid" 2>/dev/null
                sleep 0.1
                kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null
            fi
        done < "$ACTIVE_PIDS_FILE"
    fi
    
    # Asegurar que no queden procesos go run
    pkill -f "go run.*RUC" 2>/dev/null || true
}

# Limpieza mejorada
cleanup() {
    print_info "Iniciando limpieza..."
    
    kill_all_workers
    
    rm -rf "$JOBS_DIR" 2>/dev/null || true
    rm -f "$STATS_FILE" 2>/dev/null || true
    rm -f "$ACTIVE_PIDS_FILE" 2>/dev/null || true
    rm -f "${ACTIVE_PIDS_FILE}.tmp" 2>/dev/null || true
    rm -f "${ACTIVE_PIDS_FILE}.clean" 2>/dev/null || true
    release_lock
    
    print_info "Limpieza completada"
}

trap cleanup EXIT INT TERM

# =============================================================================
# SCRIPT PRINCIPAL
# =============================================================================

echo "======================================================================"
echo "SCRAPER PARALELO - CONTROL DE CONCURRENCIA MEJORADO"
echo "======================================================================"

check_dependencies

print_info "=== CONFIGURACIÓN ==="
print_info "Workers máximos: $MAX_PARALLEL_JOBS"
print_info "Tamaño de lote: $BATCH_SIZE RUCs"
print_info "Timeout por RUC: ${TIMEOUT_SCRAPER}s"
print_info "Reintentos: $MAX_REINTENTOS"
print_info "Directorio: $GO_PROJECT_DIR"

print_warning "Se ejecutarán hasta $MAX_PARALLEL_JOBS procesos simultáneos"
read -p "¿Continuar? (y/n): " -n 1 -r
echo

[[ ! $REPLY =~ ^[Yy]$ ]] && { print_error "Operación cancelada"; exit 0; }

process_all_rucs_parallel

print_success "Proceso paralelo completado!"

print_info "=== COMANDOS ÚTILES ==="
print_info "Ver total: psql '$DATABASE_URL' -c 'SELECT COUNT(*) FROM ruc_completo;'"
print_info "Ver estadísticas: psql '$DATABASE_URL' -c 'SELECT estado, COUNT(*) FROM log_consultas GROUP BY estado;'"