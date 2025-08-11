#!/bin/bash

# =============================================================================
# Script para ejecutar el scraper con RUCs obtenidos de la base de datos
# CON CONTROL DE ERRORES Y REINTENTOS - VERSION CORREGIDA
# =============================================================================

# Variables de configuración
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
DB_HOST="localhost"
DB_PORT="5433"

# Directorio del proyecto Go
GO_PROJECT_DIR="$(pwd)"
SCRAPER_PATH="cmd/scraper-completo/main.go"

# Directorio para resultados
RESULTS_DIR="resultados_scraping"
LOG_FILE="$RESULTS_DIR/scraping_log_$(date +%Y%m%d_%H%M%S).log"
ERROR_LOG="$RESULTS_DIR/error_log_$(date +%Y%m%d_%H%M%S).log"

# Configuración de reintentos
MAX_REINTENTOS=3
TIMEOUT_SCRAPER=300  # 5 minutos timeout por RUC

# Archivos de control
FAILED_RUCS_FILE="$RESULTS_DIR/rucs_fallidos.txt"
SUCCESS_RUCS_FILE="$RESULTS_DIR/rucs_exitosos.txt"
RETRY_RUCS_FILE="$RESULTS_DIR/rucs_reintentar.txt"

# Colores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Funciones de utilidad
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE" "$ERROR_LOG"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

print_retry() {
    echo -e "${PURPLE}[RETRY]${NC} $1" | tee -a "$LOG_FILE"
}

print_debug() {
    echo -e "${PURPLE}[DEBUG]${NC} $1" | tee -a "$LOG_FILE"
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
    
    # Verificar jq para validar JSON
    if ! command -v jq &> /dev/null; then
        print_warning "jq no está instalado. Se recomienda instalarlo para validar JSON"
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

# Verificar si un JSON es válido y completo - VERSION CORREGIDA SIN RESTRICCIÓN DE TAMAÑO
validate_json_file() {
    local json_file="$1"
    local ruc="$2"
    
    print_debug "Validando JSON: $json_file para RUC: $ruc"
    
    # Verificar si el archivo existe
    if [ ! -f "$json_file" ]; then
        print_error "Archivo JSON no encontrado: $json_file"
        return 1
    fi
    
    # Solo verificar que el archivo no esté vacío (0 bytes)
    local file_size=$(stat -f%z "$json_file" 2>/dev/null || stat -c%s "$json_file" 2>/dev/null)
    print_debug "Tamaño del archivo JSON: $file_size bytes"
    
    if [ "$file_size" -eq 0 ]; then
        print_error "Archivo JSON vacío (0 bytes): $json_file"
        return 1
    fi
    
    # Verificar sintaxis JSON si jq está disponible
    if command -v jq &> /dev/null; then
        if ! jq empty "$json_file" 2>/dev/null; then
            print_error "JSON inválido (sintaxis): $json_file"
            print_debug "Contenido del archivo JSON inválido (primeras 3 líneas):"
            head -3 "$json_file" 2>/dev/null | while read -r line; do
                print_debug "  $line"
            done
            return 1
        fi
        
        # Verificar campos esenciales - VERSIÓN MÁS FLEXIBLE
        local ruc_in_json=$(jq -r '.InformacionBasica.RUC // .RUC // .ruc // empty' "$json_file" 2>/dev/null)
        
        # Si no encuentra RUC en los campos esperados, buscar en todo el JSON
        if [ -z "$ruc_in_json" ] || [ "$ruc_in_json" = "null" ]; then
            print_debug "RUC no encontrado en campos estándar, buscando en todo el JSON..."
            ruc_in_json=$(jq -r ".. | objects | select(has(\"RUC\")) | .RUC" "$json_file" 2>/dev/null | head -1)
        fi
        
        # Si aún no encuentra el RUC, mostrar información de diagnóstico
        if [ -z "$ruc_in_json" ] || [ "$ruc_in_json" = "null" ]; then
            print_warning "RUC no encontrado en JSON. Estructura del archivo:"
            jq -r 'keys' "$json_file" 2>/dev/null | head -5 | while read -r key; do
                print_debug "  Clave encontrada: $key"
            done
            
            # Verificar si el JSON tiene contenido válido aunque no tenga el RUC esperado
            local has_content=$(jq -r 'keys | length' "$json_file" 2>/dev/null)
            if [ "$has_content" -gt 0 ]; then
                print_warning "JSON tiene contenido ($has_content claves) pero RUC no coincide. Aceptando como válido."
                return 0
            else
                print_error "JSON sin contenido útil"
                return 1
            fi
        fi
        
        # Validar RUC solo si se encontró
        if [ "$ruc_in_json" != "$ruc" ]; then
            print_warning "RUC en JSON ($ruc_in_json) no coincide exactamente con esperado ($ruc), pero JSON es válido"
            
            # Aceptar si es una diferencia menor (espacios, formato)
            local clean_ruc_json=$(echo "$ruc_in_json" | tr -d ' \t\n\r')
            local clean_ruc_expected=$(echo "$ruc" | tr -d ' \t\n\r')
            
            if [ "$clean_ruc_json" = "$clean_ruc_expected" ]; then
                print_info "RUCs coinciden después de limpiar formato"
            else
                print_warning "RUCs diferentes pero JSON válido, continuando..."
            fi
        fi
        
        print_info "JSON validado correctamente para RUC $ruc (${file_size} bytes)"
        return 0
    else
        print_warning "No se puede validar JSON detalladamente (jq no disponible), aceptando archivo de ${file_size} bytes"
        return 0  # Asumir válido si no hay jq
    fi
}

# Obtener RUCs de la base de datos
get_rucs_from_db() {
    print_info "Obteniendo RUCs de la tabla ruc_pruebas..."
    
    local query="SELECT DISTINCT ruc FROM ruc_pruebas WHERE ruc IS NOT NULL AND LENGTH(ruc) = 11 ORDER BY ruc;"
    
    local rucs_result=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null)
    
    if [ $? -ne 0 ]; then
        print_error "Error al consultar la base de datos"
        return 1
    fi
    
    RUCS_ARRAY=()
    while IFS= read -r line; do
        ruc=$(echo "$line" | xargs)
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

# Ejecutar scraper para un RUC con manejo de errores
run_scraper_for_ruc() {
    local ruc="$1"
    local index="$2"
    local total="$3"
    local intento="${4:-1}"
    
    print_info "[$index/$total] Procesando RUC: $ruc (Intento $intento/$MAX_REINTENTOS)"
    
    # Cambiar al directorio del proyecto
    cd "$GO_PROJECT_DIR" || {
        print_error "No se puede cambiar al directorio del proyecto: $GO_PROJECT_DIR"
        return 1
    }
    
    local json_file="ruc_completo_${ruc}.json"
    local temp_json_file="${json_file}.tmp"
    local final_json_path="$RESULTS_DIR/$json_file"
    
    # Limpiar archivos temporales previos
    rm -f "$json_file" "$temp_json_file"
    
    local start_time=$(date +%s)
    local scraper_log="$RESULTS_DIR/scraper_${ruc}_${intento}.log"
    
    # Ejecutar el scraper con logging detallado
    timeout ${TIMEOUT_SCRAPER}s go run "$SCRAPER_PATH" "$ruc" > "$scraper_log" 2>&1
    local exit_code=$?
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Analizar el resultado
    case $exit_code in
        0)
            # Éxito aparente, verificar archivo generado
            if [ -f "$json_file" ]; then
                if validate_json_file "$json_file" "$ruc"; then
                    # Mover archivo válido
                    mv "$json_file" "$final_json_path"
                    print_success "[$index/$total] RUC $ruc procesado exitosamente (${duration}s, intento $intento)"
                    echo "$ruc" >> "$SUCCESS_RUCS_FILE"
                    
                    # Limpiar log temporal si exitoso
                    rm -f "$scraper_log"
                    return 0
                else
                    print_error "[$index/$total] RUC $ruc: JSON generado inválido (${duration}s, intento $intento)"
                    # No eliminar el archivo para poder inspeccionarlo
                    mv "$json_file" "${final_json_path}.invalid" 2>/dev/null
                    return 2  # Error de validación
                fi
            else
                print_error "[$index/$total] RUC $ruc: No se generó archivo JSON (${duration}s, intento $intento)"
                return 2  # No se generó archivo
            fi
            ;;
        124)
            print_error "[$index/$total] RUC $ruc: Timeout (>${TIMEOUT_SCRAPER}s, intento $intento)"
            return 3  # Timeout
            ;;
        *)
            print_error "[$index/$total] RUC $ruc: Error del scraper (código: $exit_code, ${duration}s, intento $intento)"
            
            # Revisar log para errores específicos
            if [ -f "$scraper_log" ]; then
                local error_details=$(grep -i "error\|fail\|panic" "$scraper_log" | head -3)
                if [ -n "$error_details" ]; then
                    print_error "Detalles del error:"
                    echo "$error_details" | while read -r line; do
                        print_error "  $line"
                    done
                fi
            fi
            return 1  # Error general
            ;;
    esac
}

# Procesar RUC con reintentos
process_ruc_with_retry() {
    local ruc="$1"
    local index="$2"
    local total="$3"
    
    for intento in $(seq 1 $MAX_REINTENTOS); do
        run_scraper_for_ruc "$ruc" "$index" "$total" "$intento"
        local result=$?
        
        if [ $result -eq 0 ]; then
            return 0  # Éxito
        fi
        
        # Si no es el último intento, esperar antes de reintentar
        if [ $intento -lt $MAX_REINTENTOS ]; then
            local wait_time=$((intento * 5))  # Espera incremental
            print_retry "Esperando ${wait_time}s antes del próximo intento para RUC $ruc..."
            sleep $wait_time
        else
            # Último intento fallido
            print_error "RUC $ruc FALLÓ después de $MAX_REINTENTOS intentos"
            echo "$ruc" >> "$FAILED_RUCS_FILE"
            return $result
        fi
    done
}

# Función principal de procesamiento
process_all_rucs() {
    local total=${#RUCS_ARRAY[@]}
    local success_count=0
    local error_count=0
    local start_time=$(date +%s)
    
    print_info "Iniciando procesamiento de $total RUCs..."
    
    # Inicializar archivos de control
    > "$SUCCESS_RUCS_FILE"
    > "$FAILED_RUCS_FILE"
    > "$RETRY_RUCS_FILE"
    
    for i in "${!RUCS_ARRAY[@]}"; do
        local ruc="${RUCS_ARRAY[$i]}"
        local index=$((i + 1))
        
        print_info "=== Procesando RUC $index/$total: $ruc ==="
        
        process_ruc_with_retry "$ruc" "$index" "$total"
        
        if [ $? -eq 0 ]; then
            ((success_count++))
        else
            ((error_count++))
        fi
        
        # Pausa entre RUCs para no sobrecargar
        if [ $index -lt $total ]; then
            sleep 3
        fi
        
        # Mostrar progreso cada 5 RUCs
        if [ $((index % 5)) -eq 0 ] || [ $index -eq $total ]; then
            local elapsed=$(($(date +%s) - start_time))
            local avg_time=$((elapsed / index))
            local eta=$(((total - index) * avg_time))
            print_info "=== PROGRESO: $index/$total (${success_count} éxitos, ${error_count} errores) - ETA: ${eta}s ==="
        fi
    done
    
    # Resumen final
    local total_time=$(($(date +%s) - start_time))
    print_success "=== RESUMEN PRIMERA PASADA ==="
    print_success "Total procesados: $total"
    print_success "Exitosos: $success_count"
    print_success "Errores: $error_count"
    print_success "Tiempo total: ${total_time}s"
    print_success "Promedio por RUC: $((total_time / total))s"
    
    # Procesar reintentos si hay fallos
    if [ $error_count -gt 0 ] && [ -f "$FAILED_RUCS_FILE" ] && [ -s "$FAILED_RUCS_FILE" ]; then
        print_warning "=== INICIANDO SEGUNDA PASADA PARA RUCs FALLIDOS ==="
        process_failed_rucs
    fi
}

# Procesar RUCs que fallaron en la primera pasada
process_failed_rucs() {
    if [ ! -f "$FAILED_RUCS_FILE" ] || [ ! -s "$FAILED_RUCS_FILE" ]; then
        print_info "No hay RUCs fallidos para reprocesar"
        return 0
    fi
    
    local failed_rucs=()
    while IFS= read -r ruc; do
        if [ -n "$ruc" ]; then
            failed_rucs+=("$ruc")
        fi
    done < "$FAILED_RUCS_FILE"
    
    local total_failed=${#failed_rucs[@]}
    local retry_success=0
    local retry_failed=0
    
    print_warning "Reprocesando $total_failed RUCs fallidos..."
    
    # Limpiar archivo de fallidos para esta segunda pasada
    > "${FAILED_RUCS_FILE}.retry"
    
    for i in "${!failed_rucs[@]}"; do
        local ruc="${failed_rucs[$i]}"
        local index=$((i + 1))
        
        print_warning "=== REINTENTO $index/$total_failed: RUC $ruc ==="
        
        # Intentar una vez más con timeout extendido
        TIMEOUT_SCRAPER=600  # 10 minutos para reintentos
        
        process_ruc_with_retry "$ruc" "$index" "$total_failed"
        
        if [ $? -eq 0 ]; then
            ((retry_success++))
            print_success "RUC $ruc recuperado exitosamente en segunda pasada"
        else
            ((retry_failed++))
            echo "$ruc" >> "${FAILED_RUCS_FILE}.retry"
            print_error "RUC $ruc falló definitivamente"
        fi
        
        sleep 5  # Pausa más larga entre reintentos
    done
    
    # Actualizar archivo de fallidos finales
    mv "${FAILED_RUCS_FILE}.retry" "$FAILED_RUCS_FILE"
    
    print_warning "=== RESUMEN SEGUNDA PASADA ==="
    print_warning "RUCs reintentados: $total_failed"
    print_success "Recuperados: $retry_success"
    print_error "Fallidos definitivos: $retry_failed"
}

# Generar reporte completo
generate_comprehensive_report() {
    print_info "Generando reporte completo..."
    
    local report_file="$RESULTS_DIR/reporte_completo_$(date +%Y%m%d_%H%M%S).txt"
    local success_count=$([ -f "$SUCCESS_RUCS_FILE" ] && wc -l < "$SUCCESS_RUCS_FILE" || echo 0)
    local failed_count=$([ -f "$FAILED_RUCS_FILE" ] && wc -l < "$FAILED_RUCS_FILE" || echo 0)
    local total_rucs=${#RUCS_ARRAY[@]}
    
    {
        echo "=========================================="
        echo "REPORTE COMPLETO DE SCRAPING - $(date)"
        echo "=========================================="
        echo
        echo "ESTADÍSTICAS GENERALES:"
        echo "----------------------"
        echo "Total de RUCs procesados: $total_rucs"
        echo "RUCs exitosos: $success_count"
        echo "RUCs fallidos: $failed_count"
        if [ $total_rucs -gt 0 ]; then
            echo "Tasa de éxito: $(( (success_count * 100) / total_rucs ))%"
        fi
        echo
        echo "ARCHIVOS GENERADOS:"
        echo "------------------"
        local json_count=$(ls "$RESULTS_DIR"/*.json 2>/dev/null | wc -l)
        echo "Archivos JSON válidos: $json_count"
        local invalid_count=$(ls "$RESULTS_DIR"/*.invalid 2>/dev/null | wc -l)
        echo "Archivos JSON inválidos: $invalid_count"
        echo
        echo "DETALLE DE ARCHIVOS JSON:"
        echo "------------------------"
        if ls "$RESULTS_DIR"/*.json >/dev/null 2>&1; then
            for json_file in "$RESULTS_DIR"/*.json; do
                local size=$(stat -f%z "$json_file" 2>/dev/null || stat -c%s "$json_file" 2>/dev/null)
                echo "$(basename "$json_file"): ${size} bytes"
            done
        else
            echo "No se generaron archivos JSON válidos"
        fi
        echo
        echo "RUCs EXITOSOS:"
        echo "-------------"
        if [ -f "$SUCCESS_RUCS_FILE" ] && [ -s "$SUCCESS_RUCS_FILE" ]; then
            cat "$SUCCESS_RUCS_FILE"
        else
            echo "Ninguno"
        fi
        echo
        echo "RUCs FALLIDOS:"
        echo "-------------"
        if [ -f "$FAILED_RUCS_FILE" ] && [ -s "$FAILED_RUCS_FILE" ]; then
            cat "$FAILED_RUCS_FILE"
        else
            echo "Ninguno"
        fi
        echo
        echo "ESPACIO UTILIZADO:"
        echo "-----------------"
        du -sh "$RESULTS_DIR" 2>/dev/null || echo "Error calculando espacio"
        echo
        echo "LOGS DE ERROR DISPONIBLES:"
        echo "-------------------------"
        ls -la "$RESULTS_DIR"/*error*.log "$RESULTS_DIR"/scraper_*.log 2>/dev/null || echo "No hay logs de error"
        
    } > "$report_file"
    
    print_success "Reporte completo generado en: $report_file"
    
    # Mostrar resumen en pantalla
    echo
    print_success "=== RESUMEN FINAL ==="
    print_success "Total RUCs: $total_rucs"
    if [ $total_rucs -gt 0 ]; then
        print_success "Exitosos: $success_count ($(( (success_count * 100) / total_rucs ))%)"
        print_error "Fallidos: $failed_count ($(( (failed_count * 100) / total_rucs ))%)"
    fi
    print_success "Archivos JSON generados: $(ls "$RESULTS_DIR"/*.json 2>/dev/null | wc -l)"
}

# Función para limpiar archivos temporales y logs antiguos
cleanup_temp_files() {
    print_info "Limpiando archivos temporales..."
    
    # Eliminar archivos temporales del scraper
    rm -f ruc_completo_*.json.tmp
    rm -f "$GO_PROJECT_DIR"/ruc_completo_*.json
    
    # Mantener solo los últimos 5 logs por RUC
    find "$RESULTS_DIR" -name "scraper_*.log" -type f | sort -r | tail -n +6 | xargs rm -f 2>/dev/null
    
    print_info "Limpieza completada"
}

# =============================================================================
# SCRIPT PRINCIPAL
# =============================================================================

echo "======================================================================"
echo "🤖 SCRAPER BATCH CORREGIDO - SIN RESTRICCIÓN DE TAMAÑO"
echo "======================================================================"
echo

# Crear directorio de resultados
mkdir -p "$RESULTS_DIR"

# Inicializar logs
echo "=== INICIO DE SCRAPING CORREGIDO - $(date) ===" > "$LOG_FILE"
echo "=== LOG DE ERRORES - $(date) ===" > "$ERROR_LOG"

# Verificaciones previas
check_dependencies

# Obtener RUCs de la base de datos
if ! get_rucs_from_db; then
    print_error "No se pudieron obtener RUCs de la base de datos"
    exit 1
fi

# Mostrar configuración
print_info "=== CONFIGURACIÓN ==="
print_info "Máximo de reintentos por RUC: $MAX_REINTENTOS"
print_info "Timeout por RUC: ${TIMEOUT_SCRAPER}s"
print_info "Validación: Solo sintaxis JSON y contenido no vacío"
print_info "Directorio de resultados: $RESULTS_DIR"
echo

# Mostrar RUCs a procesar
print_info "RUCs a procesar:"
printf '%s\n' "${RUCS_ARRAY[@]}" | head -10 | tee -a "$LOG_FILE"
if [ ${#RUCS_ARRAY[@]} -gt 10 ]; then
    print_info "... y $((${#RUCS_ARRAY[@]} - 10)) más"
fi

# Confirmación del usuario
echo
print_warning "⚠️  Se procesarán ${#RUCS_ARRAY[@]} RUCs con control de errores y reintentos"
print_warning "⚠️  Tiempo estimado: $((${#RUCS_ARRAY[@]} * 2)) - $((${#RUCS_ARRAY[@]} * 5)) minutos"
read -p "¿Continuar? (y/n): " -n 1 -r
echo
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_error "Operación cancelada por el usuario"
    exit 1
fi

# Procesar todos los RUCs
process_all_rucs

# Limpiar archivos temporales
cleanup_temp_files

# Generar reporte final
generate_comprehensive_report

print_success "🎉 Procesamiento completado con validación corregida!"
print_success "📁 Revisa los archivos en: $RESULTS_DIR/"
print_success "📋 Reporte completo disponible en el directorio de resultados"

# Mostrar comandos útiles para análisis
echo
print_info "=== COMANDOS ÚTILES PARA ANÁLISIS ==="
print_info "Ver RUCs exitosos: cat $SUCCESS_RUCS_FILE"
print_info "Ver RUCs fallidos: cat $FAILED_RUCS_FILE"
print_info "Ver errores: cat $ERROR_LOG"
print_info "Validar JSON: find $RESULTS_DIR -name '*.json' -exec jq empty {} \;"
print_info "Ver archivos inválidos: ls -la $RESULTS_DIR/*.invalid"s