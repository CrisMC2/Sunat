#!/bin/bash

# =============================================================================
# Script para ejecutar go run . RUC con 10 procesos en paralelo
# CON CONTROL DE CONCURRENCIA MEJORADO Y LIMPIEZA DE CACHÉ
# =============================================================================

# Variables de configuración - BASE DE DATOS
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
#DB_HOST="localhost"
#DB_PORT="5433"
DB_HOST="192.168.18.16"
DB_PORT="5432"
#DATABASE_URL="postgres://postgres:admin123@localhost:5433/sunat?sslmode=disable"
DATABASE_URL="postgres://postgres:admin123@192.168.18.16:5432/sunat?sslmode=disable"

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

# Inicializar contadores - FIX SINTAXIS
STATS_FILE="/tmp/scraper_stats_$$"
{
    echo "TOTAL_PROCESSED=0"
    echo "TOTAL_SUCCESS=0" 
    echo "TOTAL_ERRORS=0"
    echo "TOTAL_BATCHES=0"
} > "$STATS_FILE"

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

# Función mejorada para matar procesos de Chrome/Chromium con más precisión
kill_all_chrome_processes() {
    print_error "Cerrando todos los procesos de Chrome/Chromium..."
    
    # Lista completa de procesos relacionados con Chrome
    local chrome_processes=(
        "chrome"
        "chromium"
        "Chrome"
        "Chromium" 
        "google-chrome"
        "chromium-browser"
        "chrome_crashpad_handler"
        "chrome_zygote"
        "nacl_helper"
        "chromedriver"
        "chrome --type="
        "chromium --type="
    )
    
    # Primera pasada: SIGTERM (terminación elegante)
    print_error "Enviando SIGTERM a procesos de Chrome..."
    for process in "${chrome_processes[@]}"; do
        pkill -f -TERM "$process" 2>/dev/null || true
        pgrep -f "$process" | while read pid; do
            [ -n "$pid" ] && kill -TERM "$pid" 2>/dev/null || true
        done
    done
    
    # Dar tiempo para terminación elegante
    sleep 3
    
    # Segunda pasada: SIGKILL (forzar terminación)
    print_error "Forzando terminación de procesos Chrome restantes..."
    for process in "${chrome_processes[@]}"; do
        pkill -f -KILL "$process" 2>/dev/null || true
        pgrep -f "$process" | while read pid; do
            [ -n "$pid" ] && kill -KILL "$pid" 2>/dev/null || true
        done
    done
    
    # Verificación final
    local remaining=$(pgrep -c -f "chrome\|chromium" 2>/dev/null | head -1 || echo "0")
    if [ "$remaining" -gt 0 ]; then
        print_warning "Aún quedan $remaining procesos de Chrome/Chromium"
        # Último intento más agresivo
        pkill -9 -f "chrome" 2>/dev/null || true
        pkill -9 -f "chromium" 2>/dev/null || true
    else
        print_success "Todos los procesos de Chrome terminados"
    fi
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

# Función para marcar error con especificación
mark_failed_with_specification() {
    local ruc="$1"
    local mensaje="$2"
    local especificacion="$3"
    
    local mensaje_escaped=""
    local especificacion_escaped=""
    
    [ -n "$mensaje" ] && mensaje_escaped=$(escape_sql "$mensaje")
    [ -n "$especificacion" ] && especificacion_escaped=$(escape_sql "$especificacion")
    
    local query="UPDATE log_consultas SET estado = 'fallido', mensaje = '$mensaje_escaped', especificacion = '$especificacion_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
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

# AGREGAR esta nueva función después de check_dependencies():
init_go_dependencies() {
    print_info "Inicializando dependencias de Go compartidas..."
    
    cd "$GO_PROJECT_DIR" || {
        print_error "No se puede cambiar al directorio: $GO_PROJECT_DIR"
        exit 1
    }
    
    # Descargar dependencias una sola vez
    print_info "Descargando dependencias de Go..."
    go mod download || {
        print_error "Error descargando dependencias"
        exit 1
    }
    
    # Verificar que el módulo funcione
    print_info "Verificando módulo Go..."
    go mod verify || {
        print_error "Error verificando dependencias"
        exit 1
    }
    
    print_success "Dependencias de Go inicializadas correctamente"
}

# Obtener exactamente BATCH_SIZE RUCs no exitosos
get_ruc_batch() {
    local batch_size="$1"
    local query=""
    
    # CAMBIO: Obtener RUCs de contabilidad no exitosos NI revision (incluyendo nunca procesados)
    if [ -z "$LAST_RUC_PROCESSED" ]; then
        # query="SELECT es.ruc 
        #        FROM empresas_sunat es
        #        LEFT JOIN log_consultas lc ON es.ruc::text = lc.ruc
        #        WHERE (lc.estado IS NULL OR lc.estado NOT IN ('exitoso', 'revision'))
        #        AND (es.actividad_economica_ciiu_rev3_principal ILIKE '%contabilidad%' 
        #             OR es.actividad_economica_ciiu_rev3_secundaria ILIKE '%contabilidad%' 
        #             OR es.actividad_economica_ciiu_rev4_principal ILIKE '%contabilidad%')
        #        ORDER BY es.ruc ASC LIMIT $batch_size;"
        query="SELECT es.ruc 
               FROM empresas_sunat es
               LEFT JOIN log_consultas lc ON es.ruc::text = lc.ruc
               WHERE (lc.estado IS NULL OR lc.estado NOT IN ('exitoso', 'revision'))
               AND (es.actividad_economica_ciiu_rev3_principal ILIKE '%contabilidad%' 
                    OR es.actividad_economica_ciiu_rev3_secundaria ILIKE '%contabilidad%' 
                    OR es.actividad_economica_ciiu_rev4_principal ILIKE '%contabilidad%')
               ORDER BY es.ruc DESC LIMIT $batch_size;"
    else
        # query="SELECT es.ruc 
        #        FROM empresas_sunat es
        #        LEFT JOIN log_consultas lc ON es.ruc::text = lc.ruc
        #        WHERE (lc.estado IS NULL OR lc.estado NOT IN ('exitoso', 'revision'))
        #        AND es.ruc < '$LAST_RUC_PROCESSED'
        #        AND (es.actividad_economica_ciiu_rev3_principal ILIKE '%contabilidad%' 
        #             OR es.actividad_economica_ciiu_rev3_secundaria ILIKE '%contabilidad%' 
        #             OR es.actividad_economica_ciiu_rev4_principal ILIKE '%contabilidad%')
        #        ORDER BY es.ruc ASC LIMIT $batch_size;"
        query="SELECT es.ruc 
               FROM empresas_sunat es
               LEFT JOIN log_consultas lc ON es.ruc::text = lc.ruc
               WHERE (lc.estado IS NULL OR lc.estado NOT IN ('exitoso', 'revision'))
               AND es.ruc < '$LAST_RUC_PROCESSED'
               AND (es.actividad_economica_ciiu_rev3_principal ILIKE '%contabilidad%' 
                    OR es.actividad_economica_ciiu_rev3_secundaria ILIKE '%contabilidad%' 
                    OR es.actividad_economica_ciiu_rev4_principal ILIKE '%contabilidad%')
               ORDER BY es.ruc DESC LIMIT $batch_size;"
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
    print_cache "Limpiando caché de Go (preservando dependencias)..."
    
    # Limpiar solo build cache (NO modcache para preservar dependencias)
    go clean -cache 2>/dev/null || true
    
    # Limpiar test cache
    go clean -testcache 2>/dev/null || true
    
    # NO ejecutar: go clean -modcache (esto borra las dependencias descargadas)
    
    print_cache "Caché de Go limpiado (dependencias preservadas)"
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
    
    # Verificar permisos antes de intentar limpiar page cache
    if [ -w "/proc/sys/vm/drop_caches" ] 2>/dev/null; then
        echo 3 > /proc/sys/vm/drop_caches 2>/dev/null && print_cache "✓ Page cache limpiado"
    else
        print_cache "ℹ Page cache no limpiado (requiere permisos de root)"
    fi
    
    # Configurar garbage collection de Go más agresivo
    export GOGC=10
    export GOMEMLIMIT=1GiB
    
    print_cache "Buffers de memoria procesados"
}

# Función mejorada de limpieza profunda después del lote
deep_clean_after_batch() {
    local batch_num="$1"
    
    print_cache "=== INICIANDO LIMPIEZA PROFUNDA POST-LOTE $batch_num CON VERIFICACIÓN DE PERMISOS ==="
    
    local cleanup_start=$(date +%s)
    
    # 1. Esperar que todos los procesos terminen completamente
    print_cache "Esperando finalización completa de procesos..."
    sleep 3
    
    # 2. Limpiar procesos Chrome que puedan haber quedado del lote
    local chrome_count=$(pgrep -c -f "chrome\|chromium" 2>/dev/null | tr -d '\n\r' || echo "0")
    if [ "$chrome_count" -gt 0 ]; then
        print_cache "Encontrados $chrome_count procesos Chrome del lote anterior"
        kill_all_chrome_processes
    fi
    
    # 3. Limpiar caché de Go
    clean_go_cache
    
    # 4. Limpiar archivos temporales con permisos
    clean_system_temp_with_permissions
    
    # 5. Limpiar caché de red
    clean_http_cache
    
    # 6. Limpiar buffers de memoria con permisos
    clean_memory_buffers_with_permissions
    
    # 7. Limpiar directorios específicos registrados con verificación
    if [ -f "$CACHE_DIRS_FILE" ] && [ -r "$CACHE_DIRS_FILE" ]; then
        print_cache "Limpiando directorios específicos..."
        while IFS= read -r cache_dir; do
            if [ -n "$cache_dir" ] && [ -d "$cache_dir" ]; then
                if [ -w "$cache_dir" ] || [ -w "$(dirname "$cache_dir")" ]; then
                    rm -rf "$cache_dir" 2>/dev/null && print_cache "✓ Eliminado: $cache_dir" || print_cache "✗ Error eliminando: $cache_dir"
                else
                    print_cache "✗ Sin permisos para eliminar: $cache_dir"
                fi
            fi
        done < "$CACHE_DIRS_FILE"
        
        # Limpiar el archivo de registro si es posible
        if [ -w "$CACHE_DIRS_FILE" ]; then
            : > "$CACHE_DIRS_FILE"
        fi
    fi
    
    # 8. Forzar sincronización del sistema
    print_cache "Forzando sincronización del sistema..."
    sync
    
    # 9. Verificación final del lote
    local remaining_processes=$(pgrep -c -f "go run\|chrome\|chromium" 2>/dev/null | tr -d '\n\r' || echo "0")
    if [ "$remaining_processes" -gt 0 ]; then
        print_warning "Aún quedan $remaining_processes procesos activos después de la limpieza"
    fi
    
    local cleanup_duration=$(($(date +%s) - cleanup_start))
    print_cache "=== LIMPIEZA PROFUNDA COMPLETADA EN ${cleanup_duration}s ==="
}

# Función mejorada de limpieza de sistema con verificación de permisos
clean_system_temp_with_permissions() {
    print_cache "Limpiando archivos temporales con verificación de permisos..."
    
    # 1. Limpiar /tmp con verificación de permisos
    if [ -w "/tmp" ]; then
        find /tmp -name "*scraper*$$*" -type f -user "$(whoami)" -delete 2>/dev/null || true
        find /tmp -name "go-build*" -type d -user "$(whoami)" -exec rm -rf {} + 2>/dev/null || true
        find /tmp -name "go-*" -type d -user "$(whoami)" -mtime +0 -exec rm -rf {} + 2>/dev/null || true
        print_cache "✓ Limpieza de /tmp completada"
    else
        print_warning "✗ Sin permisos de escritura en /tmp"
    fi
    
    # 2. Limpiar directorio del proyecto
    if [ -w "$GO_PROJECT_DIR" ]; then
        find "$GO_PROJECT_DIR" -name "*.tmp" -user "$(whoami)" -delete 2>/dev/null || true
        find "$GO_PROJECT_DIR" -name "*~" -user "$(whoami)" -delete 2>/dev/null || true
        find "$GO_PROJECT_DIR" -name "*.log" -user "$(whoami)" -delete 2>/dev/null || true
        print_cache "✓ Limpieza del directorio del proyecto completada"
    else
        print_warning "✗ Sin permisos de escritura en directorio del proyecto"
    fi
    
    # 3. Limpiar caché del usuario actual
    local user_cache_dir="${HOME}/.cache"
    if [ -d "$user_cache_dir" ] && [ -w "$user_cache_dir" ]; then
        find "$user_cache_dir" -name "*http*" -type f -user "$(whoami)" -delete 2>/dev/null || true
        find "$user_cache_dir" -name "*chrome*" -type d -user "$(whoami)" -exec rm -rf {} + 2>/dev/null || true
        print_cache "✓ Limpieza de caché de usuario completada"
    fi
}

# Función mejorada de limpieza de memoria con verificación de permisos
clean_memory_buffers_with_permissions() {
    print_cache "Limpiando buffers de memoria con verificación de permisos..."
    
    # Sincronización (no requiere permisos especiales)
    sync 2>/dev/null || true
    
    # Intentar limpiar page cache (requiere permisos de root)
    if [ -w "/proc/sys/vm/drop_caches" ]; then
        echo 3 > /proc/sys/vm/drop_caches 2>/dev/null && print_cache "✓ Page cache limpiado"
    else
        print_cache "ℹ Page cache no limpiado (requiere permisos de root)"
    fi
    
    # Forzar garbage collection en Go si es posible
    export GOGC=1
    print_cache "✓ Garbage collection de Go configurado"
}

# Worker function mejorada con detección de paginación
worker_function() {
    local worker_id="$1"
    local ruc="$2"
    local attempt="${3:-1}"
    
    cd "$GO_PROJECT_DIR" || {
        print_worker "$worker_id" "ERROR: No se puede cambiar al directorio"
        return 1
    }
    
    export DATABASE_URL="$DATABASE_URL"
    
    # Crear directorio temporal con permisos adecuados
    local worker_temp_dir="/tmp/worker_${worker_id}_$$"
    if mkdir -p "$worker_temp_dir" 2>/dev/null; then
        chmod 755 "$worker_temp_dir" 2>/dev/null || true
        register_cache_dir "$worker_temp_dir"
    else
        print_worker "$worker_id" "WARN: No se pudo crear directorio temporal"
        worker_temp_dir="/tmp"
    fi
    
    print_worker "$worker_id" "Procesando RUC: $ruc (intento $attempt)"
    mark_processing "$ruc" "$worker_id" "$attempt"
    
    # Archivos de salida con verificación de permisos
    local temp_output="${worker_temp_dir}/output_${worker_id}.log"
    local temp_error="${worker_temp_dir}/error_${worker_id}.log"
    
    # Ejecutar con timeout mejorado y manejo de señales
    local go_pid=""
    timeout ${TIMEOUT_SCRAPER}s go run . "$ruc" > "$temp_output" 2> "$temp_error" &
    go_pid=$!

    # Registrar el PID del proceso binario para poder matarlo después
    echo "$go_pid" >> "${ACTIVE_PIDS_FILE}.go_processes" 2>/dev/null || true
    
    # Esperar el proceso con manejo de señales
    local exit_code=0
    if wait $go_pid; then
        exit_code=0
    else
        exit_code=$?
        # Si el proceso fue matado por timeout
        if [ $exit_code -eq 124 ]; then
            print_worker "$worker_id" "TIMEOUT: Matando procesos relacionados con RUC $ruc"
            pkill -P $go_pid 2>/dev/null || true
        fi
    fi
    
    # ====================================
    # MODIFICACIÓN PRINCIPAL: CAPTURAR TODO EL TERMINAL
    # ====================================
    
    # Leer salida completa con verificación de archivos
    local output=""
    local error_output=""
    local falla_especifica=""
    local terminal_completo=""

    if [ -r "$temp_output" ]; then
        output=$(cat "$temp_output" 2>/dev/null || echo "")
        # NUEVO: Capturar TODA la salida para especificación
        terminal_completo="$output"
        
        # Buscar la línea específica de falla para mensaje
        falla_especifica=$(echo "$output" | grep -o "🛑 Terminando programa debido a falla en:.*" | head -1)
    fi
    
    if [ -r "$temp_error" ]; then
        error_output=$(cat "$temp_error" 2>/dev/null || echo "")
        # NUEVO: Agregar errores al terminal completo
        if [ -n "$terminal_completo" ] && [ -n "$error_output" ]; then
            terminal_completo="$terminal_completo\n\n=== ERRORES ===\n$error_output"
        elif [ -n "$error_output" ]; then
            terminal_completo="$error_output"
        fi
        
        # Si no encontró en output, buscar en error
        if [ -z "$falla_especifica" ]; then
            falla_especifica=$(echo "$error_output" | grep -o "🛑 Terminando programa debido a falla en:.*" | head -1)
        fi
    fi
    
    # ====================================
    # NUEVA LÓGICA PARA DETECTAR PAGINACIÓN Y GUARDAR TERMINAL COMPLETO
    # ====================================
    
    # Función para detectar paginación en la salida
    detect_pagination() {
        local content="$1"
        local pagination_info=""
        local has_pagination=false
        
        # Buscar la sección de paginación después de "📄 DETECCIÓN DE PAGINACIÓN:"
        local pagination_section=$(echo "$content" | awk '/📄 DETECCIÓN DE PAGINACIÓN:/{flag=1;next}/^[[:space:]]*$/{if(flag) flag=0}flag')
        
        if [ -n "$pagination_section" ]; then
            # Buscar líneas que contengan ✅ Sí
            local pagination_checks=$(echo "$pagination_section" | grep "✅ Sí" | sed 's/^[[:space:]]*//' | sed 's/:[[:space:]]*✅ Sí$//')
            
            if [ -n "$pagination_checks" ]; then
                has_pagination=true
                pagination_info="Paginación detectada en: $(echo "$pagination_checks" | tr '\n' ', ' | sed 's/, $//')"
            fi
        fi
        
        echo "$has_pagination|$pagination_info"
    }
    
    # Limpiar archivos temporales del worker con verificación de permisos
    if [ -d "$worker_temp_dir" ] && [ "$worker_temp_dir" != "/tmp" ]; then
        rm -rf "$worker_temp_dir" 2>/dev/null || {
            print_worker "$worker_id" "WARN: No se pudo limpiar directorio temporal"
            rm -f "$temp_output" "$temp_error" 2>/dev/null || true
        }
    fi
    
    # ====================================
    # MODIFICACIÓN: MANEJO DE CÓDIGOS DE SALIDA CON TERMINAL COMPLETO
    # ====================================
    case $exit_code in
        0)
            # PROCESO EXITOSO - VERIFICAR SI HAY PAGINACIÓN
            local pagination_result=$(detect_pagination "$output")
            local has_pagination=$(echo "$pagination_result" | cut -d'|' -f1)
            local pagination_details=$(echo "$pagination_result" | cut -d'|' -f2)
            
            if [ "$has_pagination" = "true" ]; then
                # HAY PAGINACIÓN - MARCAR COMO REVISIÓN CON TERMINAL COMPLETO
                print_worker "$worker_id" "REVISIÓN: RUC $ruc (paginación detectada)"
                
                # NUEVO: Usar terminal completo en especificación y falla específica en mensaje
                local mensaje_revision="Procesamiento exitoso con paginación detectada"
                if [ -n "$falla_especifica" ]; then
                    mensaje_revision="$mensaje_revision: $falla_especifica"
                fi
                
                mark_revision_with_terminal "$ruc" "$mensaje_revision" "$terminal_completo"
                unmark_ruc_processing "$ruc"
                update_stats "success"
                return 0
            else
                # NO HAY PAGINACIÓN - MARCAR COMO EXITOSO NORMAL CON TERMINAL COMPLETO
                print_worker "$worker_id" "OK: RUC $ruc"
                
                # NUEVO: Usar terminal completo en especificación y falla específica en mensaje
                local mensaje_exitoso="Scraping completado"
                if [ -n "$falla_especifica" ]; then
                    mensaje_exitoso="$mensaje_exitoso: $falla_especifica"
                fi
                
                mark_success_with_terminal "$ruc" "$mensaje_exitoso" "$terminal_completo"
                unmark_ruc_processing "$ruc"
                update_stats "success"
                return 0
            fi
            ;;
        124)
            print_worker "$worker_id" "TIMEOUT: RUC $ruc (${TIMEOUT_SCRAPER}s)"
            
            # NUEVO: También guardar terminal completo para timeouts
            local mensaje_timeout="Timeout después de ${TIMEOUT_SCRAPER}s"
            mark_timeout_with_terminal "$ruc" "$TIMEOUT_SCRAPER" "$terminal_completo"
            unmark_ruc_processing "$ruc"
            update_stats "error"
            return 2
            ;;
        *)
            # NUEVO: Usar terminal completo para errores
            print_worker "$worker_id" "ERROR: RUC $ruc"
            
            # Crear mensaje corto para campo 'mensaje'
            local mensaje_error="Exit code: $exit_code"
            if [ -n "$falla_especifica" ]; then
                mensaje_error="$mensaje_error: $falla_especifica"
            fi
            
            # NUEVO: Guardar terminal completo en especificación para errores
            mark_failed_with_terminal "$ruc" "$mensaje_error" "$terminal_completo"
            
            unmark_ruc_processing "$ruc"
            update_stats "error"
            return 1
            ;;
    esac
}

# =============================================================================
# MODIFICACIONES ESPECÍFICAS PARA CAPTURAR TERMINAL COMPLETO
# =============================================================================

# 1. MODIFICAR la función worker_function - CAMBIOS EN LÍNEAS ~580-750
worker_function() {
    local worker_id="$1"
    local ruc="$2"
    local attempt="${3:-1}"
    
    cd "$GO_PROJECT_DIR" || {
        print_worker "$worker_id" "ERROR: No se puede cambiar al directorio"
        return 1
    }
    
    export DATABASE_URL="$DATABASE_URL"
    
    # Crear directorio temporal con permisos adecuados
    local worker_temp_dir="/tmp/worker_${worker_id}_$$"
    if mkdir -p "$worker_temp_dir" 2>/dev/null; then
        chmod 755 "$worker_temp_dir" 2>/dev/null || true
        register_cache_dir "$worker_temp_dir"
    else
        print_worker "$worker_id" "WARN: No se pudo crear directorio temporal"
        worker_temp_dir="/tmp"
    fi
    
    print_worker "$worker_id" "Procesando RUC: $ruc (intento $attempt)"
    mark_processing "$ruc" "$worker_id" "$attempt"
    
    # Archivos de salida con verificación de permisos
    local temp_output="${worker_temp_dir}/output_${worker_id}.log"
    local temp_error="${worker_temp_dir}/error_${worker_id}.log"
    
    # Ejecutar con timeout mejorado y manejo de señales
    local go_pid=""
    timeout ${TIMEOUT_SCRAPER}s go run . "$ruc" > "$temp_output" 2> "$temp_error" &
    go_pid=$!

    # Registrar el PID del proceso binario para poder matarlo después
    echo "$go_pid" >> "${ACTIVE_PIDS_FILE}.go_processes" 2>/dev/null || true
    
    # Esperar el proceso con manejo de señales
    local exit_code=0
    if wait $go_pid; then
        exit_code=0
    else
        exit_code=$?
        # Si el proceso fue matado por timeout
        if [ $exit_code -eq 124 ]; then
            print_worker "$worker_id" "TIMEOUT: Matando procesos relacionados con RUC $ruc"
            pkill -P $go_pid 2>/dev/null || true
        fi
    fi
    
    # ====================================
    # MODIFICACIÓN PRINCIPAL: CAPTURAR TODO EL TERMINAL
    # ====================================
    
    # Leer salida completa con verificación de archivos
    local output=""
    local error_output=""
    local falla_especifica=""
    local terminal_completo=""

    if [ -r "$temp_output" ]; then
        output=$(cat "$temp_output" 2>/dev/null || echo "")
        # NUEVO: Capturar TODA la salida para especificación
        terminal_completo="$output"
        
        # Buscar la línea específica de falla para mensaje
        falla_especifica=$(echo "$output" | grep -o "🛑 Terminando programa debido a falla en:.*" | head -1)
    fi
    
    if [ -r "$temp_error" ]; then
        error_output=$(cat "$temp_error" 2>/dev/null || echo "")
        # NUEVO: Agregar errores al terminal completo
        if [ -n "$terminal_completo" ] && [ -n "$error_output" ]; then
            terminal_completo="$terminal_completo\n\n=== ERRORES ===\n$error_output"
        elif [ -n "$error_output" ]; then
            terminal_completo="$error_output"
        fi
        
        # Si no encontró en output, buscar en error
        if [ -z "$falla_especifica" ]; then
            falla_especifica=$(echo "$error_output" | grep -o "🛑 Terminando programa debido a falla en:.*" | head -1)
        fi
    fi
    
    # ====================================
    # NUEVA LÓGICA PARA DETECTAR PAGINACIÓN Y GUARDAR TERMINAL COMPLETO
    # ====================================
    
    # Función para detectar paginación en la salida
    detect_pagination() {
        local content="$1"
        local pagination_info=""
        local has_pagination=false
        
        # Buscar la sección de paginación después de "📄 DETECCIÓN DE PAGINACIÓN:"
        local pagination_section=$(echo "$content" | awk '/📄 DETECCIÓN DE PAGINACIÓN:/{flag=1;next}/^[[:space:]]*$/{if(flag) flag=0}flag')
        
        if [ -n "$pagination_section" ]; then
            # Buscar líneas que contengan ✅ Sí
            local pagination_checks=$(echo "$pagination_section" | grep "✅ Sí" | sed 's/^[[:space:]]*//' | sed 's/:[[:space:]]*✅ Sí$//')
            
            if [ -n "$pagination_checks" ]; then
                has_pagination=true
                pagination_info="Paginación detectada en: $(echo "$pagination_checks" | tr '\n' ', ' | sed 's/, $//')"
            fi
        fi
        
        echo "$has_pagination|$pagination_info"
    }
    
    # Limpiar archivos temporales del worker con verificación de permisos
    if [ -d "$worker_temp_dir" ] && [ "$worker_temp_dir" != "/tmp" ]; then
        rm -rf "$worker_temp_dir" 2>/dev/null || {
            print_worker "$worker_id" "WARN: No se pudo limpiar directorio temporal"
            rm -f "$temp_output" "$temp_error" 2>/dev/null || true
        }
    fi
    
    # ====================================
    # MODIFICACIÓN: MANEJO DE CÓDIGOS DE SALIDA CON TERMINAL COMPLETO
    # ====================================
    case $exit_code in
        0)
            # PROCESO EXITOSO - VERIFICAR SI HAY PAGINACIÓN
            local pagination_result=$(detect_pagination "$output")
            local has_pagination=$(echo "$pagination_result" | cut -d'|' -f1)
            local pagination_details=$(echo "$pagination_result" | cut -d'|' -f2)
            
            if [ "$has_pagination" = "true" ]; then
                # HAY PAGINACIÓN - MARCAR COMO REVISIÓN CON TERMINAL COMPLETO
                print_worker "$worker_id" "REVISIÓN: RUC $ruc (paginación detectada)"
                
                # NUEVO: Usar terminal completo en especificación y falla específica en mensaje
                local mensaje_revision="Procesamiento exitoso con paginación detectada"
                if [ -n "$falla_especifica" ]; then
                    mensaje_revision="$mensaje_revision: $falla_especifica"
                fi
                
                mark_revision_with_terminal "$ruc" "$mensaje_revision" "$terminal_completo"
                unmark_ruc_processing "$ruc"
                update_stats "success"
                return 0
            else
                # NO HAY PAGINACIÓN - MARCAR COMO EXITOSO NORMAL CON TERMINAL COMPLETO
                print_worker "$worker_id" "OK: RUC $ruc"
                
                # NUEVO: Usar terminal completo en especificación y falla específica en mensaje
                local mensaje_exitoso="Scraping completado"
                if [ -n "$falla_especifica" ]; then
                    mensaje_exitoso="$mensaje_exitoso: $falla_especifica"
                fi
                
                mark_success_with_terminal "$ruc" "$mensaje_exitoso" "$terminal_completo"
                unmark_ruc_processing "$ruc"
                update_stats "success"
                return 0
            fi
            ;;
        124)
            print_worker "$worker_id" "TIMEOUT: RUC $ruc (${TIMEOUT_SCRAPER}s)"
            
            # NUEVO: También guardar terminal completo para timeouts
            local mensaje_timeout="Timeout después de ${TIMEOUT_SCRAPER}s"
            mark_timeout_with_terminal "$ruc" "$TIMEOUT_SCRAPER" "$terminal_completo"
            unmark_ruc_processing "$ruc"
            update_stats "error"
            return 2
            ;;
        *)
            # NUEVO: Usar terminal completo para errores
            print_worker "$worker_id" "ERROR: RUC $ruc"
            
            # Crear mensaje corto para campo 'mensaje'
            local mensaje_error="Exit code: $exit_code"
            if [ -n "$falla_especifica" ]; then
                mensaje_error="$mensaje_error: $falla_especifica"
            fi
            
            # NUEVO: Guardar terminal completo en especificación para errores
            mark_failed_with_terminal "$ruc" "$mensaje_error" "$terminal_completo"
            
            unmark_ruc_processing "$ruc"
            update_stats "error"
            return 1
            ;;
    esac
}

# =============================================================================
# 2. AGREGAR NUEVAS FUNCIONES PARA GUARDAR CON TERMINAL COMPLETO
# =============================================================================

# Nueva función para marcar éxito con terminal completo
mark_success_with_terminal() {
    local ruc="$1"
    local mensaje="$2"
    local terminal_completo="$3"
    
    local mensaje_escaped=""
    local terminal_escaped=""
    
    [ -n "$mensaje" ] && mensaje_escaped=$(escape_sql "$mensaje")
    [ -n "$terminal_completo" ] && terminal_escaped=$(escape_sql "$terminal_completo")
    
    local query="UPDATE log_consultas SET estado = 'exitoso', mensaje = '$mensaje_escaped', especificacion = '$terminal_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
}

# Nueva función para marcar revisión con terminal completo
mark_revision_with_terminal() {
    local ruc="$1"
    local mensaje="$2"
    local terminal_completo="$3"
    
    local mensaje_escaped=""
    local terminal_escaped=""
    
    [ -n "$mensaje" ] && mensaje_escaped=$(escape_sql "$mensaje")
    [ -n "$terminal_completo" ] && terminal_escaped=$(escape_sql "$terminal_completo")
    
    local query="UPDATE log_consultas SET estado = 'revision', mensaje = '$mensaje_escaped', especificacion = '$terminal_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
}

# Nueva función para marcar timeout con terminal completo
mark_timeout_with_terminal() {
    local ruc="$1"
    local timeout_seconds="$2"
    local terminal_completo="$3"
    
    local mensaje="Timeout después de ${timeout_seconds}s"
    local mensaje_escaped=$(escape_sql "$mensaje")
    local terminal_escaped=""
    
    [ -n "$terminal_completo" ] && terminal_escaped=$(escape_sql "$terminal_completo")
    
    local query="UPDATE log_consultas SET estado = 'fallido', mensaje = '$mensaje_escaped', especificacion = '$terminal_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
}

# Nueva función para marcar falla con terminal completo
mark_failed_with_terminal() {
    local ruc="$1"
    local mensaje="$2"
    local terminal_completo="$3"
    
    local mensaje_escaped=""
    local terminal_escaped=""
    
    [ -n "$mensaje" ] && mensaje_escaped=$(escape_sql "$mensaje")
    [ -n "$terminal_completo" ] && terminal_escaped=$(escape_sql "$terminal_completo")
    
    local query="UPDATE log_consultas SET estado = 'fallido', mensaje = '$mensaje_escaped', especificacion = '$terminal_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
}

# Nueva función para marcar como revisión
mark_revision() {
    local ruc="$1"
    local mensaje="$2"
    local especificacion="$3"
    
    local mensaje_escaped=""
    local especificacion_escaped=""
    
    [ -n "$mensaje" ] && mensaje_escaped=$(escape_sql "$mensaje")
    [ -n "$especificacion" ] && especificacion_escaped=$(escape_sql "$especificacion")
    
    local query=""
    if [ -n "$especificacion" ]; then
        query="UPDATE log_consultas SET estado = 'revision', mensaje = '$mensaje_escaped', especificacion = '$especificacion_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    else
        query="UPDATE log_consultas SET estado = 'revision', mensaje = '$mensaje_escaped', fecha_registro = CURRENT_TIMESTAMP WHERE ruc = '$ruc';"
    fi
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" >/dev/null 2>&1
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
            # Configurar manejo de señales en el subshell
            trap 'exit 130' INT TERM
            
            local my_pid=$$
            register_pid "$my_pid"
            
            # Trap para asegurar limpieza
            trap "unregister_pid $my_pid; exit" EXIT
            
            # Verificar si se solicitó terminación antes de procesar
            if [ "$TERMINATION_REQUESTED" = true ]; then
                unregister_pid "$my_pid"
                exit 130
            fi
            
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
    
    # RECUENTO TOTAL DESPUÉS DE CADA LOTE
    print_info "=== RECUENTO TOTAL DESPUÉS DEL LOTE $batch_num ==="

    # Consultar totales desde la base de datos
    local total_exitoso=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM log_consultas WHERE estado = 'exitoso';" 2>/dev/null | tr -d ' \t\n\r' || echo "0")
    local total_revision=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM log_consultas WHERE estado = 'revision';" 2>/dev/null | tr -d ' \t\n\r' || echo "0")
    local total_fallido=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM log_consultas WHERE estado = 'fallido';" 2>/dev/null | tr -d ' \t\n\r' || echo "0")
    local total_procesando=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM log_consultas WHERE estado = 'procesando';" 2>/dev/null | tr -d ' \t\n\r' || echo "0")
    local total_error_terminal=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM log_consultas WHERE estado = 'error_terminal';" 2>/dev/null | tr -d ' \t\n\r' || echo "0")
    local total_general=$((total_exitoso + total_revision + total_fallido + total_procesando + total_error_terminal))

    print_success "✅ EXITOSOS: $total_exitoso"
    print_info "🔄 REVISIÓN (paginación): $total_revision" 
    print_error "❌ FALLIDOS: $total_fallido"
    print_warning "⏳ PROCESANDO: $total_procesando"
    print_error "💀 ERROR TERMINAL: $total_error_terminal"
    print_info "📊 TOTAL PROCESADOS: $total_general"

    # Calcular porcentajes
    if [ $total_general -gt 0 ]; then
        local pct_exitoso=$(( (total_exitoso * 100) / total_general ))
        local pct_fallido=$(( (total_fallido * 100) / total_general ))
        print_info "📈 Tasa de éxito: ${pct_exitoso}% | Tasa de fallo: ${pct_fallido}%"
    fi
    print_info "================================================="

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

# Función mejorada para matar workers con timeout y procesos Go
kill_all_workers() {
    print_info "Terminando workers y procesos relacionados..."
    
    # 1. Terminar workers registrados con kill más agresivo
    if [ -f "$ACTIVE_PIDS_FILE" ]; then
        print_info "Terminando workers registrados..."
        while IFS= read -r pid; do
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                print_info "Terminando worker PID: $pid"
                # Terminar el proceso y toda su jerarquía
                kill -TERM "$pid" 2>/dev/null || true
                # Terminar procesos hijos
                pkill -P "$pid" -TERM 2>/dev/null || true
            fi
        done < "$ACTIVE_PIDS_FILE"
        
        # Esperar menos tiempo para terminación elegante
        sleep 1
        
        # Forzar terminación más agresiva
        while IFS= read -r pid; do
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                print_warning "Forzando terminación de worker PID: $pid"
                kill -KILL "$pid" 2>/dev/null || true
                # Matar procesos hijos también
                pkill -P "$pid" -KILL 2>/dev/null || true
            fi
        done < "$ACTIVE_PIDS_FILE"
    fi
    
    # 2. Terminar TODOS los procesos Go de forma más agresiva
    print_info "Terminando procesos Go relacionados..."
    
    # Obtener PIDs específicos antes de matarlos
    local go_pids=($(pgrep -f "go run.*RUC" 2>/dev/null || true))
    local timeout_pids=($(pgrep -f "timeout.*go run" 2>/dev/null || true))
    local scraper_pids=($(pgrep -f "scraper-completo" 2>/dev/null || true))
    
    # Matar con SIGTERM primero
    for pid in "${go_pids[@]}" "${timeout_pids[@]}" "${scraper_pids[@]}"; do
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            print_info "Terminando proceso Go PID: $pid"
            kill -TERM "$pid" 2>/dev/null || true
            pkill -P "$pid" -TERM 2>/dev/null || true  # Hijos también
        fi
    done
    
    # Esperar muy poco tiempo
    sleep 0.5
    
    # Matar con SIGKILL inmediatamente
    for pid in "${go_pids[@]}" "${timeout_pids[@]}" "${scraper_pids[@]}"; do
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            print_warning "Forzando terminación de proceso Go PID: $pid"
            kill -KILL "$pid" 2>/dev/null || true
            pkill -P "$pid" -KILL 2>/dev/null || true  # Hijos también
        fi
    done
    
    # 3. Usar pkill más agresivo para patrones específicos
    print_info "Eliminando procesos por patrón..."
    
    # Matar por patrón con SIGKILL inmediatamente
    pkill -9 -f "go run.*RUC" 2>/dev/null || true
    pkill -9 -f "timeout.*go run" 2>/dev/null || true
    pkill -9 -f "scraper-completo" 2>/dev/null || true
    pkill -9 -f "main.*RUC" 2>/dev/null || true
    
    # 4. Limpiar jobs del shell
    print_info "Limpiando jobs del shell..."
    jobs -p | xargs -r kill -KILL 2>/dev/null || true
    
    # 5. Verificación y limpieza final
    sleep 0.5
    local remaining_go=$(pgrep -c -f "go run\|scraper-completo\|timeout.*go" 2>/dev/null | head -1 || echo "0")
    if [ "$remaining_go" -gt 0 ]; then
        print_warning "Aún quedan $remaining_go procesos Go, eliminación final..."
        # Última pasada ultra-agresiva
        killall -9 go 2>/dev/null || true
        pkill -9 -f "RUC" 2>/dev/null || true
    fi
    
    print_success "Workers y procesos relacionados terminados"
}

cleanup_files_with_permissions() {
    print_info "Limpiando archivos de control..."
    
    local files_to_clean=(
        "$STATS_FILE"
        "$ACTIVE_PIDS_FILE"
        "${ACTIVE_PIDS_FILE}.go_processes"
        "$CACHE_DIRS_FILE"
        "$PROCESSING_RUCS_FILE"
        "${PROCESSING_RUCS_FILE}.tmp"
        "${ACTIVE_PIDS_FILE}.tmp"
        "${ACTIVE_PIDS_FILE}.clean"
    )
    
    for file in "${files_to_clean[@]}"; do
        if [ -f "$file" ] && [ -w "$file" ]; then
            rm -f "$file" 2>/dev/null || true
        fi
    done
    
    # Limpiar directorio de jobs
    if [ -d "$JOBS_DIR" ] && [ -w "$JOBS_DIR" ]; then
        rm -rf "$JOBS_DIR" 2>/dev/null || true
    fi
    
    print_info "Archivos de control limpiados"
}

# Función principal de limpieza mejorada
cleanup() {
    print_error "=== INICIANDO LIMPIEZA DE TERMINACIÓN MEJORADA ==="
    
    # Marcar que se solicitó terminación
    TERMINATION_REQUESTED=true
    
    # 1. Marcar todos los RUCs en proceso como error_terminal
    mark_all_processing_as_terminal_error
    
    # 2. Terminar workers y procesos Go (ORDEN IMPORTANTE)
    kill_all_workers
    
    # 3. Matar todos los procesos de Chrome (después de los workers)
    kill_all_chrome_processes
    
    # 4. Limpiar archivos de control con verificación de permisos
    cleanup_files_with_permissions
    
    # 5. Liberar lock
    release_lock
    
    # 6. Limpieza final de caché si estaba habilitada
    if [ "$DEEP_CLEAN_ENABLED" = true ]; then
        print_cache "Limpieza final de caché con verificación de permisos..."
        clean_go_cache
        clean_system_temp_with_permissions
        clean_memory_buffers_with_permissions
    fi
    
    # 7. Verificación final de procesos
    print_info "Verificando procesos restantes..."
    
    local go_remaining=$(pgrep -c -f "go run\|scraper-completo" 2>/dev/null | head -1 || echo "0")
    if [ "$go_remaining" -gt 0 ]; then
        print_warning "Quedan $go_remaining procesos Go. Terminación final..."
        pkill -9 -f "go run" 2>/dev/null || true
        pkill -9 -f "scraper-completo" 2>/dev/null || true
    fi
    
    local chrome_remaining=$(pgrep -c -f "chrome\|chromium" 2>/dev/null | head -1 || echo "0")
    if [ "$chrome_remaining" -gt 0 ]; then
        print_warning "Quedan $chrome_remaining procesos Chrome. Terminación final..."
        pkill -9 -f "chrome\|chromium" 2>/dev/null || true
    fi
    
    # 8. Limpiar jobs del shell actual
    jobs -p | xargs -r kill -9 2>/dev/null || true
    
    # Limpiar procesos Go específicos registrados
    if [ -f "${ACTIVE_PIDS_FILE}.go_processes" ]; then
        print_info "Eliminando procesos Go registrados..."
        while IFS= read -r pid; do
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                kill -KILL "$pid" 2>/dev/null || true
            fi
        done < "${ACTIVE_PIDS_FILE}.go_processes"
        rm -f "${ACTIVE_PIDS_FILE}.go_processes" 2>/dev/null || true
    fi

    print_error "=== LIMPIEZA COMPLETADA ==="

    # Mostrar estadísticas finales si están disponibles
    if [ -f "$STATS_FILE" ] 2>/dev/null && [ -r "$STATS_FILE" ]; then
        source "$STATS_FILE" 2>/dev/null || true
        print_error "Estadísticas al momento de la terminación:"
        print_error "- Lotes procesados: ${TOTAL_BATCHES:-0}"
        print_error "- RUCs procesados: ${TOTAL_PROCESSED:-0}"
        print_error "- Exitosos: ${TOTAL_SUCCESS:-0}"
        print_error "- Errores: ${TOTAL_ERRORS:-0}"
    fi
}

# Variable para evitar múltiples cleanups
CLEANUP_EXECUTED=false

# Función mejorada para manejar terminación
handle_termination_signal() {
    [ "$CLEANUP_EXECUTED" = true ] && exit 130
    CLEANUP_EXECUTED=true
    
    print_error ""
    print_error "==================== TERMINACIÓN SOLICITADA ===================="
    print_error "Recibida señal de terminación (CTRL+C o KILL)"
    print_error "Iniciando proceso de terminación segura..."
    print_error "=============================================================="
    
    cleanup
    exit 130
}

# Solo un trap para terminación
trap handle_termination_signal INT TERM

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