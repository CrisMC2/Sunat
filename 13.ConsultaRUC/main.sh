#!/bin/bash

# =============================================================================
# Script para ejecutar go run . RUC con 10 procesos en paralelo
# CON CONTROL DE CONCURRENCIA MEJORADO Y LIMPIEZA DE CACHÉ
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
PAUSE_BETWEEN_BATCHES=10  # 10 segundos entre lotes (aumentado para limpieza)
MAX_REINTENTOS=2          # Máximo 2 intentos por RUC
BATCH_SIZE=100            # FIJO: 100 RUCs por lote
DEEP_CLEAN_ENABLED=true   # Habilitar limpieza profunda

# Contadores globales (con archivos para sincronización)
STATS_FILE="/tmp/scraper_stats_$$"
LOCK_FILE="/tmp/scraper_lock_$$"
JOBS_DIR="/tmp/scraper_jobs_$$"
ACTIVE_PIDS_FILE="/tmp/scraper_pids_$$"
CACHE_DIRS_FILE="/tmp/scraper_cache_dirs_$$"

# Crear directorio para jobs
mkdir -p "$JOBS_DIR"

# Inicializar contadores
echo "TOTAL_PROCESSED=0" > "$STATS_FILE"
echo "TOTAL_SUCCESS=0" >> "$STATS_FILE"
echo "TOTAL_ERRORS=0" >> "$STATS_FILE"
echo "TOTAL_BATCHES=0" >> "$STATS_FILE"

# Inicializar archivo de PIDs activos
echo "" > "$ACTIVE_PIDS_FILE"

# Inicializar archivo de directorios de caché
echo "" > "$CACHE_DIRS_FILE"

# Archivo para rastrear RUCs en proceso
PROCESSING_RUCS_FILE="/tmp/scraper_processing_rucs_$"
echo "" > "$PROCESSING_RUCS_FILE"

# Variable para cursor de RUC
LAST_RUC_PROCESSED=""

# Variable para controlar terminación
TERMINATION_REQUESTED=false

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
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

# Función para marcar RUC como en procesamiento
mark_ruc_processing() {
    local ruc="$1"
    local worker_id="$2"
    
    acquire_lock || return 1
    echo "${ruc}:${worker_id}:$(date +%s)" >> "$PROCESSING_RUCS_FILE"
    release_lock
}

# Función para desmarcar RUC de procesamiento
unmark_ruc_processing() {
    local ruc="$1"
    
    acquire_lock || return 1
    grep -v "^${ruc}:" "$PROCESSING_RUCS_FILE" > "${PROCESSING_RUCS_FILE}.tmp" 2>/dev/null || touch "${PROCESSING_RUCS_FILE}.tmp"
    mv "${PROCESSING_RUCS_FILE}.tmp" "$PROCESSING_RUCS_FILE"
    release_lock
}

# Función para marcar todos los RUCs en proceso como error terminal
mark_all_processing_as_terminal_error() {
    print_error "Marcando RUCs en procesamiento como 'error_terminal'..."
    
    if [ -f "$PROCESSING_RUCS_FILE" ]; then
        while IFS=':' read -r ruc worker_id timestamp; do
            if [ -n "$ruc" ]; then
                print_error "Marcando RUC $ruc como error_terminal (era worker $worker_id)"
                update_log_consultas "$ruc" "error_terminal" "Proceso terminado por usuario (CTRL+C/KILL)"
            fi
        done < "$PROCESSING_RUCS_FILE"
    fi
}

# Función para matar todos los procesos de Chrome/Chromium
kill_all_chrome_processes() {
    print_error "Cerrando todos los procesos de Chrome/Chromium..."
    
    # Matar Chrome por nombre de proceso
    pkill -f -TERM chrome 2>/dev/null || true
    pkill -f -TERM chromium 2>/dev/null || true
    pkill -f -TERM Chrome 2>/dev/null || true
    pkill -f -TERM Chromium 2>/dev/null || true
    
    # Matar procesos específicos de Chrome
    pkill -f -TERM "google-chrome" 2>/dev/null || true
    pkill -f -TERM "chromium-browser" 2>/dev/null || true
    
    # Dar tiempo para que terminen graciosamente
    sleep 2
    
    # Si no terminaron, forzar con KILL
    pkill -f -KILL chrome 2>/dev/null || true
    pkill -f -KILL chromium 2>/dev/null || true
    pkill -f -KILL Chrome 2>/dev/null || true
    pkill -f -KILL Chromium 2>/dev/null || true
    pkill -f -KILL "google-chrome" 2>/dev/null || true
    pkill -f -KILL "chromium-browser" 2>/dev/null || true
    
    # También matar procesos de Selenium WebDriver si existen
    pkill -f -TERM chromedriver 2>/dev/null || true
    pkill -f -KILL chromedriver 2>/dev/null || true
    
    print_error "Procesos de Chrome terminados"
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
    local ruc="$1"
    local worker_id="${2:-1}"
    local attempt="${3:-1}"
    local mensaje="Worker-${worker_id} procesando (intento ${attempt})"
    
    # Marcar en la base de datos
    update_log_consultas "$ruc" "procesando" "$mensaje"
    
    # Registrar en el archivo de procesamiento
    mark_ruc_processing "$ruc" "$worker_id"
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
    local TOTAL_BATCHES=0
    
    [ -f "$STATS_FILE" ] && source "$STATS_FILE"
    
    case "$operation" in
        "processed") TOTAL_PROCESSED=$((TOTAL_PROCESSED + 1)) ;;
        "success") TOTAL_SUCCESS=$((TOTAL_SUCCESS + 1)) ;;
        "error") TOTAL_ERRORS=$((TOTAL_ERRORS + 1)) ;;
        "batch") TOTAL_BATCHES=$((TOTAL_BATCHES + 1)) ;;
    esac
    
    echo "TOTAL_PROCESSED=$TOTAL_PROCESSED" > "$STATS_FILE"
    echo "TOTAL_SUCCESS=$TOTAL_SUCCESS" >> "$STATS_FILE"
    echo "TOTAL_ERRORS=$TOTAL_ERRORS" >> "$STATS_FILE"
    echo "TOTAL_BATCHES=$TOTAL_BATCHES" >> "$STATS_FILE"
    
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

print_cache() {
    echo -e "${PURPLE}[CACHE]${NC} $1"
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

# Obtener exactamente BATCH_SIZE RUCs no exitosos
get_ruc_batch() {
    local batch_size="$1"
    local query=""
    
    # CAMBIO PRINCIPAL: Siempre obtener exactamente batch_size RUCs no exitosos
    if [ -z "$LAST_RUC_PROCESSED" ]; then
        query="SELECT ruc FROM log_consultas WHERE estado != 'exitoso' ORDER BY ruc DESC LIMIT $batch_size;"
    else
        query="SELECT ruc FROM log_consultas WHERE estado != 'exitoso' AND ruc < '$LAST_RUC_PROCESSED' ORDER BY ruc DESC LIMIT $batch_size;"
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

# =============================================================================
# FUNCIONES DE LIMPIEZA DE CACHÉ
# =============================================================================

# Registrar directorio de caché usado
register_cache_dir() {
    local cache_dir="$1"
    [ -z "$cache_dir" ] && return 1
    
    acquire_lock || return 1
    echo "$cache_dir" >> "$CACHE_DIRS_FILE"
    release_lock
}

# Limpiar caché del sistema Go
clean_go_cache() {
    print_cache "Limpiando caché de Go..."
    
    # Limpiar build cache
    go clean -cache 2>/dev/null || true
    
    # Limpiar module cache (más agresivo)
    go clean -modcache 2>/dev/null || true
    
    # Limpiar test cache
    go clean -testcache 2>/dev/null || true
    
    print_cache "Caché de Go limpiado"
}

# Limpiar archivos temporales del sistema
clean_system_temp() {
    print_cache "Limpiando archivos temporales del sistema..."
    
    # Limpiar /tmp de archivos relacionados con el scraper (excluyendo los actuales)
    find /tmp -name "*scraper*" -type f -not -path "${STATS_FILE}*" -not -path "${LOCK_FILE}*" -not -path "${ACTIVE_PIDS_FILE}*" -not -path "${CACHE_DIRS_FILE}*" -delete 2>/dev/null || true
    
    # Limpiar archivos temporales de go
    find /tmp -name "go-*" -type d -exec rm -rf {} + 2>/dev/null || true
    
    # Limpiar archivos build temporales
    find "$GO_PROJECT_DIR" -name "*.tmp" -delete 2>/dev/null || true
    find "$GO_PROJECT_DIR" -name "*~" -delete 2>/dev/null || true
    
    print_cache "Archivos temporales limpiados"
}

# Limpiar caché de red/HTTP
clean_http_cache() {
    print_cache "Limpiando caché de red..."
    
    # Variables de entorno que pueden contener caché HTTP
    unset HTTP_PROXY
    unset HTTPS_PROXY
    unset NO_PROXY
    
    # Limpiar posibles cachés de HTTP client de Go
    local home_dir="${HOME:-/tmp}"
    [ -d "$home_dir/.cache" ] && find "$home_dir/.cache" -name "*http*" -type f -delete 2>/dev/null || true
    
    print_cache "Caché de red limpiado"
}

# Limpiar memoria y buffers (si es posible)
clean_memory_buffers() {
    print_cache "Intentando limpiar buffers de memoria..."
    
    # Forzar garbage collection si es posible
    sync 2>/dev/null || true
    
    # Limpiar page cache, dentries e inodes (requiere permisos)
    echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || print_cache "No se pudo limpiar page cache (permisos)"
    
    print_cache "Buffers de memoria procesados"
}

# Función principal de limpieza profunda
deep_clean_after_batch() {
    local batch_num="$1"
    
    print_cache "=== INICIANDO LIMPIEZA PROFUNDA POST-LOTE $batch_num ==="
    
    local cleanup_start=$(date +%s)
    
    # 1. Esperar que todos los procesos terminen completamente
    print_cache "Esperando finalización completa de procesos..."
    sleep 2
    
    # 2. Limpiar caché de Go
    clean_go_cache
    
    # 3. Limpiar archivos temporales
    clean_system_temp
    
    # 4. Limpiar caché de red
    clean_http_cache
    
    # 5. Limpiar buffers de memoria
    clean_memory_buffers
    
    # 6. Limpiar directorios específicos registrados
    if [ -f "$CACHE_DIRS_FILE" ]; then
        print_cache "Limpiando directorios específicos..."
        while IFS= read -r cache_dir; do
            if [ -n "$cache_dir" ] && [ -d "$cache_dir" ]; then
                rm -rf "$cache_dir" 2>/dev/null || true
                print_cache "Eliminado: $cache_dir"
            fi
        done < "$CACHE_DIRS_FILE"
        
        # Limpiar el archivo de registro
        : > "$CACHE_DIRS_FILE"
    fi
    
    # 7. Forzar recolección de basura del sistema
    print_cache "Forzando sincronización del sistema..."
    sync
    
    local cleanup_duration=$(($(date +%s) - cleanup_start))
    print_cache "=== LIMPIEZA PROFUNDA COMPLETADA EN ${cleanup_duration}s ==="
}

# Worker function mejorada con registro de caché
worker_function() {
    local worker_id="$1"
    local ruc="$2"
    local attempt="${3:-1}"
    
    cd "$GO_PROJECT_DIR" || {
        print_worker "$worker_id" "ERROR: No se puede cambiar al directorio"
        return 1
    }
    
    export DATABASE_URL="$DATABASE_URL"
    
    # Registrar posibles directorios de caché que se generen
    local worker_temp_dir="/tmp/worker_${worker_id}_$$"
    mkdir -p "$worker_temp_dir"
    register_cache_dir "$worker_temp_dir"
    
    print_worker "$worker_id" "Procesando RUC: $ruc (intento $attempt)"
    mark_processing "$ruc" "$worker_id" "$attempt"
    
    # Ejecutar con timeout y redirección mejorada
    local temp_output="${worker_temp_dir}/output.log"
    local temp_error="${worker_temp_dir}/error.log"
    
    timeout ${TIMEOUT_SCRAPER}s go run . "$ruc" > "$temp_output" 2> "$temp_error"
    local exit_code=$?
    
    # Leer salida limitada
    local output=""
    local error_output=""
    [ -f "$temp_output" ] && output=$(head -c 200 "$temp_output" 2>/dev/null)
    [ -f "$temp_error" ] && error_output=$(head -c 100 "$temp_error" 2>/dev/null)
    
    # Limpiar archivos temporales del worker
    rm -rf "$worker_temp_dir" 2>/dev/null || true
    
    case $exit_code in
        0)
            print_worker "$worker_id" "OK: RUC $ruc"
            mark_success "$ruc"
            unmark_ruc_processing "$ruc"
            update_stats "success"
            return 0
            ;;
        124)
            print_worker "$worker_id" "TIMEOUT: RUC $ruc"
            mark_timeout "$ruc" "$TIMEOUT_SCRAPER"
            unmark_ruc_processing "$ruc"
            update_stats "error"
            return 2
            ;;
        *)
            local error_msg="Error $exit_code"
            [ -n "$error_output" ] && error_msg="$error_msg: $error_output"
            print_worker "$worker_id" "ERROR: RUC $ruc (código: $exit_code)"
            mark_failed "$ruc" "$error_msg"
            unmark_ruc_processing "$ruc"
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
    
    # VALIDACIÓN: Debe tener exactamente BATCH_SIZE RUCs
    if [ ${#rucs[@]} -eq 0 ]; then
        print_warning "Lote $batch_num está vacío"
        return 1
    fi
    
    if [ ${#rucs[@]} -lt $BATCH_SIZE ]; then
        print_warning "Lote $batch_num tiene solo ${#rucs[@]} RUCs (esperado: $BATCH_SIZE)"
        print_info "Esto podría indicar que quedan pocos RUCs pendientes"
    fi
    
    print_info "=== LOTE $batch_num: ${#rucs[@]} RUCs ==="
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
        # Verificar si se solicitó terminación
        if [ "$TERMINATION_REQUESTED" = true ]; then
            print_warning "Terminación solicitada, esperando que terminen los workers actuales..."
            break
        fi
        
        local current_time=$(date +%s)
        if [ $((current_time - last_report)) -ge 30 ]; then
            local active=$(count_active_processes)
            [ -f "$STATS_FILE" ] && source "$STATS_FILE"
            print_info "Activos: $active | Procesados: $TOTAL_PROCESSED | OK: $TOTAL_SUCCESS | Errores: $TOTAL_ERRORS"
            last_report=$current_time
        fi
        sleep 2
    done
    
    # Actualizar cursor para el siguiente lote
    [ ${#rucs[@]} -gt 0 ] && LAST_RUC_PROCESSED="${rucs[-1]}"
    
    local batch_duration=$(($(date +%s) - batch_start_time))
    [ -f "$STATS_FILE" ] && source "$STATS_FILE"
    update_stats "batch"
    print_success "Lote $batch_num completado en ${batch_duration}s (Procesados=$TOTAL_PROCESSED)"
    
    # LIMPIEZA PROFUNDA DESPUÉS DEL LOTE
    if [ "$DEEP_CLEAN_ENABLED" = true ]; then
        deep_clean_after_batch "$batch_num"
    fi
    
    return 0
}

# Proceso principal
process_all_rucs_parallel() {
    local start_time=$(date +%s)
    local batch_num=1
    
    print_info "=== INICIANDO PROCESAMIENTO PARALELO CON LIMPIEZA DE CACHÉ ==="
    print_info "Workers máx: $MAX_PARALLEL_JOBS | Lote fijo: $BATCH_SIZE RUCs | Timeout: ${TIMEOUT_SCRAPER}s"
    print_info "Limpieza profunda: ${DEEP_CLEAN_ENABLED}"
    
    while true; do
        # Verificar si se solicitó terminación
        if [ "$TERMINATION_REQUESTED" = true ]; then
            print_warning "Terminación solicitada, deteniendo procesamiento de nuevos lotes"
            break
        fi
        
        print_info "=== Obteniendo lote $batch_num (exactamente $BATCH_SIZE RUCs no exitosos) ==="
        
        local rucs_raw=$(get_ruc_batch "$BATCH_SIZE")
        
        if [ -z "$rucs_raw" ]; then
            print_success "No hay más RUCs no exitosos para procesar"
            break
        fi
        
        local rucs=()
        while IFS= read -r line; do
            [ -n "$line" ] && rucs+=("$line")
        done <<< "$rucs_raw"
        
        if [ ${#rucs[@]} -eq 0 ]; then
            print_success "No se encontraron RUCs válidos"
            break
        fi
        
        # Procesar el lote (incluye limpieza automática)
        if ! process_batch_parallel "${rucs[*]}" "$batch_num"; then
            print_error "Error procesando lote $batch_num"
            break
        fi
        
        # Verificar nuevamente si se solicitó terminación
        if [ "$TERMINATION_REQUESTED" = true ]; then
            print_warning "Terminación solicitada, finalizando proceso"
            break
        fi
        
        ((batch_num++))
        
        # Pausa aumentada entre lotes para permitir que la limpieza sea efectiva
        if [ $PAUSE_BETWEEN_BATCHES -gt 0 ]; then
            print_info "Pausa de ${PAUSE_BETWEEN_BATCHES}s antes del siguiente lote..."
            sleep $PAUSE_BETWEEN_BATCHES
        fi
        
        # Mostrar progreso
        [ -f "$STATS_FILE" ] && source "$STATS_FILE"
        print_info "Progreso: Lotes completados: $TOTAL_BATCHES | RUCs procesados: $TOTAL_PROCESSED"
    done
    
    # Resumen final
    local total_time=$(($(date +%s) - start_time))
    
    if [ -f "$STATS_FILE" ]; then
        source "$STATS_FILE"
        
        print_success "=== PROCESAMIENTO COMPLETADO ==="
        print_success "Total de lotes: $TOTAL_BATCHES"
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
            
            if [ $TOTAL_BATCHES -gt 0 ]; then
                local avg_per_batch=$((total_time / TOTAL_BATCHES))
                local rucs_per_batch=$(( TOTAL_PROCESSED / TOTAL_BATCHES ))
                print_success "Promedio por lote: ${avg_per_batch}s (${rucs_per_batch} RUCs/lote)"
            fi
            
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

# Limpieza mejorada con manejo de terminación
cleanup() {
    print_error "=== INICIANDO LIMPIEZA DE TERMINACIÓN ==="
    
    # Marcar que se solicitó terminación
    TERMINATION_REQUESTED=true
    
    # 1. Marcar todos los RUCs en proceso como error_terminal
    mark_all_processing_as_terminal_error
    
    # 2. Matar todos los procesos de Chrome
    kill_all_chrome_processes
    
    # 3. Terminar workers activos
    kill_all_workers
    
    # 4. Limpiar archivos de control
    rm -rf "$JOBS_DIR" 2>/dev/null || true
    rm -f "$STATS_FILE" 2>/dev/null || true
    rm -f "$ACTIVE_PIDS_FILE" 2>/dev/null || true
    rm -f "${ACTIVE_PIDS_FILE}.tmp" 2>/dev/null || true
    rm -f "${ACTIVE_PIDS_FILE}.clean" 2>/dev/null || true
    rm -f "$CACHE_DIRS_FILE" 2>/dev/null || true
    rm -f "$PROCESSING_RUCS_FILE" 2>/dev/null || true
    rm -f "${PROCESSING_RUCS_FILE}.tmp" 2>/dev/null || true
    release_lock
    
    # 5. Limpieza final de caché si estaba habilitada
    if [ "$DEEP_CLEAN_ENABLED" = true ]; then
        print_cache "Limpieza final de caché..."
        clean_go_cache
        clean_system_temp
    fi
    
    # 6. Matar cualquier proceso Go relacionado que pueda haber quedado
    pkill -f "go run" 2>/dev/null || true
    pkill -f "scraper-completo" 2>/dev/null || true
    
    # 7. Verificar procesos de Chrome restantes
    local chrome_count=$(pgrep -c -f "chrome\|chromium" 2>/dev/null || echo "0")
    if [ "$chrome_count" -gt 0 ]; then
        print_warning "Aún quedan $chrome_count procesos de Chrome. Forzando terminación final..."
        pkill -f -KILL "chrome" 2>/dev/null || true
        pkill -f -KILL "chromium" 2>/dev/null || true
    fi
    
    print_error "=== LIMPIEZA COMPLETADA ==="
    
    # Mostrar estadísticas finales si están disponibles
    if [ -f "$STATS_FILE" ] 2>/dev/null; then
        source "$STATS_FILE" 2>/dev/null || true
        print_error "Estadísticas al momento de la terminación:"
        print_error "- Lotes procesados: ${TOTAL_BATCHES:-0}"
        print_error "- RUCs procesados: ${TOTAL_PROCESSED:-0}"
        print_error "- Exitosos: ${TOTAL_SUCCESS:-0}"
        print_error "- Errores: ${TOTAL_ERRORS:-0}"
    fi
}

# Función específica para manejar señales de terminación
handle_termination_signal() {
    print_error ""
    print_error "==================== TERMINACIÓN SOLICITADA ===================="
    print_error "Recibida señal de terminación (CTRL+C o KILL)"
    print_error "Iniciando proceso de terminación segura..."
    print_error "=============================================================="
    
    cleanup
    exit 130  # Código de salida estándar para SIGINT
}

trap handle_termination_signal INT TERM
trap cleanup EXIT

# =============================================================================
# SCRIPT PRINCIPAL
# =============================================================================

echo "======================================================================"
echo "SCRAPER PARALELO - CON LIMPIEZA PROFUNDA DE CACHÉ"
echo "======================================================================"

check_dependencies

print_info "=== CONFIGURACIÓN ==="
print_info "Workers máximos: $MAX_PARALLEL_JOBS"
print_info "RUCs por lote (FIJO): $BATCH_SIZE"
print_info "Timeout por RUC: ${TIMEOUT_SCRAPER}s"
print_info "Reintentos: $MAX_REINTENTOS"
print_info "Pausa entre lotes: ${PAUSE_BETWEEN_BATCHES}s"
print_info "Limpieza profunda: ${DEEP_CLEAN_ENABLED}"
print_info "Directorio: $GO_PROJECT_DIR"

print_warning "Se procesarán EXACTAMENTE $BATCH_SIZE RUCs no exitosos por lote"
print_warning "Se ejecutará limpieza profunda de caché después de cada lote"
print_warning "Se ejecutarán hasta $MAX_PARALLEL_JOBS procesos simultáneos"

read -p "¿Continuar? (y/n): " -n 1 -r
echo

[[ ! $REPLY =~ ^[Yy]$ ]] && { print_error "Operación cancelada"; exit 0; }

process_all_rucs_parallel

print_success "Proceso paralelo con limpieza completado!"

print_info "=== COMANDOS ÚTILES ==="
print_info "Ver total: psql '$DATABASE_URL' -c 'SELECT COUNT(*) FROM ruc_completo;'"
print_info "Ver estadísticas: psql '$DATABASE_URL' -c 'SELECT estado, COUNT(*) FROM log_consultas GROUP BY estado;'"
print_info "Ver últimos procesados: psql '$DATABASE_URL' -c 'SELECT ruc, estado, mensaje, fecha_registro FROM log_consultas ORDER BY fecha_registro DESC LIMIT 10;'"
print_info "Ver errores terminales: psql '$DATABASE_URL' -c 'SELECT COUNT(*) FROM log_consultas WHERE estado = \"error_terminal\";'"