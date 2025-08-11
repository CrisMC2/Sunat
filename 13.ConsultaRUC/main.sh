#!/bin/bash

# =============================================================================
# Script para ejecutar el scraper con RUCs obtenidos de la base de datos
# =============================================================================

# Variables de configuración
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
DB_HOST="localhost"
DB_PORT="5433"

# Directorio del proyecto Go
GO_PROJECT_DIR="$(pwd)"  # Asume que estás en el directorio raíz del proyecto
SCRAPER_PATH="cmd/scraper-completo/main.go"

# Directorio para resultados
RESULTS_DIR="resultados_scraping"
LOG_FILE="$RESULTS_DIR/scraping_log_$(date +%Y%m%d_%H%M%S).log"

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Funciones de utilidad
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Verificar dependencias
check_dependencies() {
    print_info "Verificando dependencias..."
    
    # Verificar PostgreSQL
    if ! command -v psql &> /dev/null; then
        print_error "psql no está instalado o no está en el PATH"
        exit 1
    fi
    
    # Verificar Go
    if ! command -v go &> /dev/null; then
        print_error "Go no está instalado o no está en el PATH"
        exit 1
    fi
    
    # Verificar conexión a la base de datos
    PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        print_error "No se puede conectar a PostgreSQL"
        exit 1
    fi
    
    # Verificar que existe el scraper
    if [ ! -f "$SCRAPER_PATH" ]; then
        print_error "No se encuentra el archivo del scraper: $SCRAPER_PATH"
        exit 1
    fi
    
    print_success "Todas las dependencias están disponibles"
}

# Obtener RUCs de la base de datos
get_rucs_from_db() {
    print_info "Obteniendo RUCs de la tabla ruc_pruebas..."
    
    # Query para obtener RUCs - ajusta según tu estructura de tabla
    # Asumiendo que la tabla tiene una columna 'ruc'
    local query="SELECT DISTINCT ruc FROM ruc_pruebas WHERE ruc IS NOT NULL AND LENGTH(ruc) = 11 ORDER BY ruc;"
    
    # Ejecutar query y obtener resultados
    local rucs_result=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null)
    
    if [ $? -ne 0 ]; then
        print_error "Error al consultar la base de datos"
        return 1
    fi
    
    # Limpiar espacios y crear array
    RUCS_ARRAY=()
    while IFS= read -r line; do
        ruc=$(echo "$line" | xargs)  # Eliminar espacios
        if [ -n "$ruc" ]; then
            RUCS_ARRAY+=("$ruc")
        fi
    done <<< "$rucs_result"
    
    local count=${#RUCS_ARRAY[@]}
    if [ $count -eq 0 ]; then
        print_warning "No se encontraron RUCs en la tabla ruc_pruebas"
        return 1
    fi
    
    print_success "Se encontraron $count RUCs para procesar"
    return 0
}

# Ejecutar scraper para un RUC
run_scraper_for_ruc() {
    local ruc="$1"
    local index="$2"
    local total="$3"
    
    print_info "[$index/$total] Procesando RUC: $ruc"
    
    # Cambiar al directorio del proyecto
    cd "$GO_PROJECT_DIR" || {
        print_error "No se puede cambiar al directorio del proyecto: $GO_PROJECT_DIR"
        return 1
    }
    
    # Ejecutar el scraper
    local start_time=$(date +%s)
    
    # Redirigir output del scraper al log
    timeout 300s go run "$SCRAPER_PATH" "$ruc" >> "$LOG_FILE" 2>&1
    local exit_code=$?
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    case $exit_code in
        0)
            print_success "[$index/$total] RUC $ruc procesado exitosamente (${duration}s)"
            
            # Verificar si se generó el archivo JSON
            local json_file="ruc_completo_${ruc}.json"
            if [ -f "$json_file" ]; then
                # Mover archivo al directorio de resultados
                mv "$json_file" "$RESULTS_DIR/" 2>/dev/null
                print_info "Archivo JSON movido a: $RESULTS_DIR/$json_file"
            fi
            ;;
        124)
            print_error "[$index/$total] Timeout procesando RUC $ruc (>300s)"
            ;;
        *)
            print_error "[$index/$total] Error procesando RUC $ruc (código: $exit_code, tiempo: ${duration}s)"
            ;;
    esac
    
    return $exit_code
}

# Función principal de procesamiento
process_all_rucs() {
    local total=${#RUCS_ARRAY[@]}
    local success_count=0
    local error_count=0
    local start_time=$(date +%s)
    
    print_info "Iniciando procesamiento de $total RUCs..."
    
    for i in "${!RUCS_ARRAY[@]}"; do
        local ruc="${RUCS_ARRAY[$i]}"
        local index=$((i + 1))
        
        run_scraper_for_ruc "$ruc" "$index" "$total"
        
        if [ $? -eq 0 ]; then
            ((success_count++))
        else
            ((error_count++))
        fi
        
        # Pausa opcional entre requests para no sobrecargar
        if [ $index -lt $total ]; then
            sleep 2
        fi
        
        # Mostrar progreso cada 10 RUCs
        if [ $((index % 10)) -eq 0 ] || [ $index -eq $total ]; then
            local elapsed=$(($(date +%s) - start_time))
            local avg_time=$((elapsed / index))
            local eta=$(((total - index) * avg_time))
            print_info "Progreso: $index/$total (${success_count} éxitos, ${error_count} errores) - ETA: ${eta}s"
        fi
    done
    
    # Resumen final
    local total_time=$(($(date +%s) - start_time))
    print_success "=== RESUMEN FINAL ==="
    print_success "Total procesados: $total"
    print_success "Exitosos: $success_count"
    print_success "Errores: $error_count"
    print_success "Tiempo total: ${total_time}s"
    print_success "Promedio por RUC: $((total_time / total))s"
    print_success "Archivos generados en: $RESULTS_DIR/"
}

# Generar reporte de archivos generados
generate_report() {
    print_info "Generando reporte de archivos..."
    
    local report_file="$RESULTS_DIR/reporte_$(date +%Y%m%d_%H%M%S).txt"
    
    {
        echo "REPORTE DE SCRAPING - $(date)"
        echo "=================================="
        echo
        echo "Archivos JSON generados:"
        ls -la "$RESULTS_DIR"/*.json 2>/dev/null | wc -l
        echo
        echo "Lista de archivos:"
        ls -la "$RESULTS_DIR"/*.json 2>/dev/null || echo "No se generaron archivos JSON"
        echo
        echo "Tamaño total:"
        du -sh "$RESULTS_DIR" 2>/dev/null
    } > "$report_file"
    
    print_success "Reporte generado en: $report_file"
}

# =============================================================================
# SCRIPT PRINCIPAL
# =============================================================================

echo "======================================================================"
echo "🤖 SCRAPER BATCH - PROCESAMIENTO MASIVO DE RUCs"
echo "======================================================================"
echo

# Crear directorio de resultados
mkdir -p "$RESULTS_DIR"

# Inicializar log
echo "=== INICIO DE SCRAPING - $(date) ===" > "$LOG_FILE"

# Verificaciones previas
check_dependencies

# Obtener RUCs de la base de datos
if ! get_rucs_from_db; then
    print_error "No se pudieron obtener RUCs de la base de datos"
    exit 1
fi

# Mostrar RUCs a procesar
print_info "RUCs a procesar:"
printf '%s\n' "${RUCS_ARRAY[@]}" | head -10 | tee -a "$LOG_FILE"
if [ ${#RUCS_ARRAY[@]} -gt 10 ]; then
    print_info "... y $((${#RUCS_ARRAY[@]} - 10)) más"
fi

# Confirmación del usuario
echo
print_warning "⚠️  Se procesarán ${#RUCS_ARRAY[@]} RUCs"
print_warning "⚠️  Esto puede tomar varios minutos/horas"
read -p "¿Continuar? (y/n): " -n 1 -r
echo
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_error "Operación cancelada por el usuario"
    exit 1
fi

# Procesar todos los RUCs
process_all_rucs

# Generar reporte final
generate_report

print_success "🎉 Procesamiento completado. Revisa los archivos en: $RESULTS_DIR/"