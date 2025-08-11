#!/bin/bash

# ============================================
# SCRIPT DE IMPORTACIÓN JSON A POSTGRESQL - VERSIÓN CORREGIDA
# ============================================

# Configuración de la base de datos
DB_USER="postgres"
DB_NAME="sunat"
DB_PASSWORD="admin123"
DB_HOST="localhost"
DB_PORT="5433"

# Directorio de archivos JSON
JSON_DIR="./resultados_scraping/"

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Contadores globales
TOTAL_PROCESSED=0
TOTAL_ERRORS=0
TOTAL_SKIPPED=0

# Función para logging
log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR $(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" >&2
}

warning() {
    echo -e "${YELLOW}[WARNING $(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO $(date '+%Y-%m-%d %H:%M:%S')]${NC} $1"
}

# Función para verificar dependencias
check_dependencies() {
    if ! command -v jq &> /dev/null; then
        error "jq no está instalado. Por favor instálalo con: sudo apt-get install jq"
        exit 1
    fi
    
    if ! command -v psql &> /dev/null; then
        error "psql no está instalado. Por favor instala postgresql-client"
        exit 1
    fi
    
    log "✓ Dependencias verificadas"
}

# Función mejorada para convertir fechas
convert_date() {
    local date_str="$1"
    
    # Casos donde retornamos NULL
    if [[ -z "$date_str" ]] || [[ "$date_str" == "null" ]] || [[ "$date_str" == "-" ]] || [[ "$date_str" == "" ]]; then
        echo "NULL"
        return 0
    fi
    
    # Verificar formato DD/MM/YYYY
    if [[ "$date_str" =~ ^[0-9]{2}/[0-9]{2}/[0-9]{4}$ ]]; then
        # Convertir DD/MM/YYYY a YYYY-MM-DD usando sed
        local converted=$(echo "$date_str" | sed 's|\([0-9][0-9]\)/\([0-9][0-9]\)/\([0-9][0-9][0-9][0-9]\)|\3-\2-\1|')
        echo "'$converted'"
    else
        # Si no es el formato esperado, retornar NULL
        echo "NULL"
    fi
}

# Función mejorada para convertir timestamps ISO
convert_timestamp() {
    local timestamp="$1"
    
    if [[ -z "$timestamp" ]] || [[ "$timestamp" == "null" ]] || [[ "$timestamp" == "-" ]]; then
        echo "CURRENT_TIMESTAMP"
    else
        # Convertir timestamp ISO a formato PostgreSQL
        # Ejemplo: "2025-08-08T15:34:08.054166525-05:00" -> timestamp
        local clean_timestamp=$(echo "$timestamp" | sed 's/T/ /' | sed 's/\.[0-9]*-/ -/' | sed 's/-05:00//')
        echo "'$clean_timestamp'"
    fi
}

# Función mejorada para escapar SQL
escape_sql() {
    local str="$1"
    if [[ -z "$str" ]] || [[ "$str" == "null" ]] || [[ "$str" == "-" ]]; then
        echo "NULL"
    else
        # Escapar comillas simples y backslashes
        local escaped=$(echo "$str" | sed "s/'/''/g" | sed 's/\\/\\\\/g')
        echo "'$escaped'"
    fi
}

# Función para verificar conexión a la base de datos
check_db_connection() {
    log "Verificando conexión a la base de datos..."
    export PGPASSWORD="$DB_PASSWORD"
    
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
        error "No se puede conectar a la base de datos"
        exit 1
    fi
    
    log "✓ Conexión a la base de datos exitosa"
}

# Función para ejecutar SQL con mejor manejo de errores
execute_sql() {
    local sql="$1"
    local description="$2"
    local show_error="${3:-true}"
    
    export PGPASSWORD="$DB_PASSWORD"
    
    local result=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$sql" 2>&1)
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        return 0
    else
        if [[ "$show_error" == "true" ]]; then
            error "Error ejecutando: $description"
            error "SQL Error: $result"
        fi
        return 1
    fi
}

# Función para verificar si un valor de jq es null o empty
is_null_or_empty() {
    local value="$1"
    if [[ -z "$value" ]] || [[ "$value" == "null" ]] || [[ "$value" == "empty" ]]; then
        return 0  # true
    else
        return 1  # false
    fi
}

# Función para obtener array de forma segura
get_safe_array() {
    local json_file="$1"
    local jq_path="$2"
    
    local result=$(jq -r "if $jq_path == null then empty else $jq_path[]? end" "$json_file" 2>/dev/null)
    echo "$result"
}

# Función para procesar información básica MEJORADA
process_basic_info() {
    local json_file="$1"
    local ruc=$(jq -r '.informacion_basica.ruc // empty' "$json_file")
    
    if [[ -z "$ruc" ]]; then
        error "RUC no encontrado en $json_file"
        return 1
    fi
    
    info "Procesando información básica para RUC: $ruc"
    
    # Extraer datos básicos con manejo seguro de null
    local razon_social=$(jq -r '.informacion_basica.razon_social // ""' "$json_file")
    local tipo_contribuyente=$(jq -r '.informacion_basica.tipo_contribuyente // ""' "$json_file")
    local nombre_comercial=$(jq -r '.informacion_basica.nombre_comercial // ""' "$json_file")
    local fecha_inscripcion=$(jq -r '.informacion_basica.fecha_inscripcion // ""' "$json_file")
    local fecha_inicio_actividades=$(jq -r '.informacion_basica.fecha_inicio_actividades // ""' "$json_file")
    local estado=$(jq -r '.informacion_basica.estado // ""' "$json_file")
    local condicion=$(jq -r '.informacion_basica.condicion // ""' "$json_file")
    local domicilio_fiscal=$(jq -r '.informacion_basica.domicilio_fiscal // ""' "$json_file")
    local sistema_emision=$(jq -r '.informacion_basica.sistema_emision // ""' "$json_file")
    local actividad_comercio_exterior=$(jq -r '.informacion_basica.actividad_comercio_exterior // ""' "$json_file")
    local sistema_contabilidad=$(jq -r '.informacion_basica.sistema_contabilidad // ""' "$json_file")
    local sistema_emision_electronica=$(jq -r '.informacion_basica.sistema_emision_electronica // ""' "$json_file")
    local emisor_electronico_desde=$(jq -r '.informacion_basica.emisor_electronico_desde // ""' "$json_file")
    local afiliado_ple=$(jq -r '.informacion_basica.afiliado_ple // ""' "$json_file")
    
    # Convertir fechas
    local fecha_inscripcion_sql=$(convert_date "$fecha_inscripcion")
    local fecha_inicio_actividades_sql=$(convert_date "$fecha_inicio_actividades")
    local emisor_electronico_desde_sql=$(convert_date "$emisor_electronico_desde")
    
    # Verificar si el RUC ya existe
    local ruc_exists=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM ruc_info WHERE ruc = '$ruc';" 2>/dev/null | tr -d ' ')
    
    local sql
    if [[ "$ruc_exists" == "1" ]]; then
        # Actualizar registro existente
        sql="UPDATE ruc_info SET 
            razon_social = $(escape_sql "$razon_social"),
            tipo_contribuyente = $(escape_sql "$tipo_contribuyente"),
            nombre_comercial = $(escape_sql "$nombre_comercial"),
            fecha_inscripcion = $fecha_inscripcion_sql,
            fecha_inicio_actividades = $fecha_inicio_actividades_sql,
            estado = $(escape_sql "$estado"),
            condicion = $(escape_sql "$condicion"),
            domicilio_fiscal = $(escape_sql "$domicilio_fiscal"),
            sistema_emision = $(escape_sql "$sistema_emision"),
            actividad_comercio_exterior = $(escape_sql "$actividad_comercio_exterior"),
            sistema_contabilidad = $(escape_sql "$sistema_contabilidad"),
            sistema_emision_electronica = $(escape_sql "$sistema_emision_electronica"),
            emisor_electronico_desde = $emisor_electronico_desde_sql,
            afiliado_ple = $(escape_sql "$afiliado_ple"),
            updated_at = CURRENT_TIMESTAMP
            WHERE ruc = '$ruc';"
    else
        # Insertar nuevo registro
        sql="INSERT INTO ruc_info (
            ruc, razon_social, tipo_contribuyente, nombre_comercial, 
            fecha_inscripcion, fecha_inicio_actividades, estado, condicion,
            domicilio_fiscal, sistema_emision, actividad_comercio_exterior,
            sistema_contabilidad, sistema_emision_electronica, emisor_electronico_desde,
            afiliado_ple
        ) VALUES (
            '$ruc', $(escape_sql "$razon_social"), $(escape_sql "$tipo_contribuyente"), 
            $(escape_sql "$nombre_comercial"), $fecha_inscripcion_sql, 
            $fecha_inicio_actividades_sql, $(escape_sql "$estado"), 
            $(escape_sql "$condicion"), $(escape_sql "$domicilio_fiscal"),
            $(escape_sql "$sistema_emision"), $(escape_sql "$actividad_comercio_exterior"),
            $(escape_sql "$sistema_contabilidad"), $(escape_sql "$sistema_emision_electronica"),
            $emisor_electronico_desde_sql, $(escape_sql "$afiliado_ple")
        );"
    fi
    
    if execute_sql "$sql" "Información básica para RUC $ruc" "true"; then
        info "✓ Información básica procesada para RUC: $ruc"
        process_arrays "$json_file" "$ruc"
        return 0
    else
        error "✗ Error procesando información básica para RUC: $ruc"
        return 1
    fi
}

# Función MEJORADA para procesar arrays
process_arrays() {
    local json_file="$1"
    local ruc="$2"
    
    # Limpiar arrays existentes para este RUC
    execute_sql "DELETE FROM ruc_actividades_economicas WHERE ruc = '$ruc';" "Limpiar actividades económicas" "false"
    execute_sql "DELETE FROM ruc_comprobantes_pago WHERE ruc = '$ruc';" "Limpiar comprobantes de pago" "false"
    execute_sql "DELETE FROM ruc_comprobantes_electronicos WHERE ruc = '$ruc';" "Limpiar comprobantes electrónicos" "false"
    execute_sql "DELETE FROM ruc_padrones WHERE ruc = '$ruc';" "Limpiar padrones" "false"
    
    # Procesar actividades económicas de forma segura
    local actividades=$(get_safe_array "$json_file" ".informacion_basica.actividades_economicas")
    if [[ -n "$actividades" ]]; then
        while IFS= read -r actividad; do
            if [[ -n "$actividad" ]]; then
                local sql="INSERT INTO ruc_actividades_economicas (ruc, actividad_economica) VALUES ('$ruc', $(escape_sql "$actividad"));"
                execute_sql "$sql" "Actividad económica" "false"
            fi
        done <<< "$actividades"
    fi
    
    # Procesar comprobantes de pago de forma segura
    local comprobantes_pago=$(get_safe_array "$json_file" ".informacion_basica.comprobantes_pago")
    if [[ -n "$comprobantes_pago" ]]; then
        while IFS= read -r comprobante; do
            if [[ -n "$comprobante" ]]; then
                local sql="INSERT INTO ruc_comprobantes_pago (ruc, comprobante_pago) VALUES ('$ruc', $(escape_sql "$comprobante"));"
                execute_sql "$sql" "Comprobante de pago" "false"
            fi
        done <<< "$comprobantes_pago"
    fi
    
    # Procesar comprobantes electrónicos de forma segura
    local comprobantes_electronicos=$(get_safe_array "$json_file" ".informacion_basica.comprobantes_electronicos")
    if [[ -n "$comprobantes_electronicos" ]]; then
        while IFS= read -r comprobante; do
            if [[ -n "$comprobante" ]]; then
                local sql="INSERT INTO ruc_comprobantes_electronicos (ruc, comprobante_electronico) VALUES ('$ruc', $(escape_sql "$comprobante"));"
                execute_sql "$sql" "Comprobante electrónico" "false"
            fi
        done <<< "$comprobantes_electronicos"
    fi
    
    # Procesar padrones de forma segura
    local padrones=$(get_safe_array "$json_file" ".informacion_basica.padrones")
    if [[ -n "$padrones" ]]; then
        while IFS= read -r padron; do
            if [[ -n "$padron" ]]; then
                local sql="INSERT INTO ruc_padrones (ruc, padron) VALUES ('$ruc', $(escape_sql "$padron"));"
                execute_sql "$sql" "Padrón" "false"
            fi
        done <<< "$padrones"
    fi
}

# Función NUEVA para procesar información histórica
process_informacion_historica() {
    local json_file="$1"
    local ruc="$2"
    
    # Verificar si hay información histórica
    local tiene_historica=$(jq -r '.informacion_historica // empty' "$json_file")
    if [[ -z "$tiene_historica" ]] || [[ "$tiene_historica" == "null" ]]; then
        return 0
    fi
    
    info "Procesando información histórica para RUC: $ruc"
    
    # Procesar información histórica básica
    local razon_social_hist=$(jq -r '.informacion_historica.razon_social // ""' "$json_file")
    local fecha_actualizada=$(jq -r '.informacion_historica.fecha_actualizada // ""' "$json_file")
    local fecha_consulta=$(jq -r '.informacion_historica.fecha_consulta // ""' "$json_file")
    
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    if [[ -n "$razon_social_hist" ]] || [[ -n "$fecha_actualizada" ]]; then
        local sql="INSERT INTO informacion_historica (ruc, razon_social, fecha_actualizada, fecha_consulta) 
                   VALUES ('$ruc', $(escape_sql "$razon_social_hist"), $(escape_sql "$fecha_actualizada"), $fecha_consulta_sql)
                   ON CONFLICT (ruc) DO UPDATE SET
                   razon_social = EXCLUDED.razon_social,
                   fecha_actualizada = EXCLUDED.fecha_actualizada,
                   fecha_consulta = EXCLUDED.fecha_consulta;"
        execute_sql "$sql" "Información histórica básica" "false"
    fi
    
    # Procesar condiciones históricas
    execute_sql "DELETE FROM condicion_historica WHERE ruc = '$ruc';" "Limpiar condiciones históricas" "false"
    local condiciones=$(jq -c '.informacion_historica.condiciones[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$condiciones" ]]; then
        while IFS= read -r condicion_obj; do
            if [[ -n "$condicion_obj" ]]; then
                local condicion=$(echo "$condicion_obj" | jq -r '.condicion // ""')
                local desde=$(echo "$condicion_obj" | jq -r '.desde // ""')
                local hasta=$(echo "$condicion_obj" | jq -r '.hasta // ""')
                
                local desde_sql=$(convert_date "$desde")
                local hasta_sql=$(convert_date "$hasta")
                
                local sql="INSERT INTO condicion_historica (ruc, condicion, desde, hasta) 
                           VALUES ('$ruc', $(escape_sql "$condicion"), $desde_sql, $hasta_sql);"
                execute_sql "$sql" "Condición histórica" "false"
            fi
        done <<< "$condiciones"
    fi
    
    # Procesar domicilios históricos
    execute_sql "DELETE FROM domicilio_fiscal_historico WHERE ruc = '$ruc';" "Limpiar domicilios históricos" "false"
    local domicilios=$(jq -c '.informacion_historica.domicilios[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$domicilios" ]]; then
        while IFS= read -r domicilio_obj; do
            if [[ -n "$domicilio_obj" ]]; then
                local direccion=$(echo "$domicilio_obj" | jq -r '.direccion // ""')
                local fecha_baja=$(echo "$domicilio_obj" | jq -r '.fecha_de_baja // ""')
                
                local fecha_baja_sql=$(convert_date "$fecha_baja")
                
                local sql="INSERT INTO domicilio_fiscal_historico (ruc, direccion, fecha_de_baja) 
                           VALUES ('$ruc', $(escape_sql "$direccion"), $fecha_baja_sql);"
                execute_sql "$sql" "Domicilio histórico" "false"
            fi
        done <<< "$domicilios"
    fi
    
    # Procesar razones sociales históricas
    local razones_sociales=$(jq -c '.informacion_historica.razones_sociales[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$razones_sociales" ]]; then
        execute_sql "DELETE FROM razon_social_historica WHERE ruc = '$ruc';" "Limpiar razones sociales históricas" "false"
        while IFS= read -r razon_obj; do
            if [[ -n "$razon_obj" ]]; then
                local nombre=$(echo "$razon_obj" | jq -r '.nombre // ""')
                local fecha_baja=$(echo "$razon_obj" | jq -r '.fecha_de_baja // ""')
                
                local fecha_baja_sql=$(convert_date "$fecha_baja")
                
                local sql="INSERT INTO razon_social_historica (ruc, nombre, fecha_de_baja) 
                           VALUES ('$ruc', $(escape_sql "$nombre"), $fecha_baja_sql);"
                execute_sql "$sql" "Razón social histórica" "false"
            fi
        done <<< "$razones_sociales"
    fi
}

# Función NUEVA para procesar cantidad de trabajadores
process_cantidad_trabajadores() {
    local json_file="$1"
    local ruc="$2"
    
    # Verificar si hay información de trabajadores
    local tiene_trabajadores=$(jq -r '.cantidad_trabajadores // empty' "$json_file")
    if [[ -z "$tiene_trabajadores" ]] || [[ "$tiene_trabajadores" == "null" ]]; then
        return 0
    fi
    
    info "Procesando cantidad trabajadores para RUC: $ruc"
    
    local fecha_consulta=$(jq -r '.cantidad_trabajadores.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    # Insertar registro principal
    local sql="INSERT INTO cantidad_trabajadores (ruc, fecha_consulta) 
               VALUES ('$ruc', $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET fecha_consulta = EXCLUDED.fecha_consulta;"
    execute_sql "$sql" "Cantidad trabajadores principal" "false"
    
    # Procesar períodos disponibles
    execute_sql "DELETE FROM cantidad_trabajadores_periodos WHERE ruc = '$ruc';" "Limpiar períodos trabajadores" "false"
    local periodos=$(get_safe_array "$json_file" ".cantidad_trabajadores.periodos_disponibles")
    if [[ -n "$periodos" ]]; then
        while IFS= read -r periodo; do
            if [[ -n "$periodo" ]]; then
                local sql="INSERT INTO cantidad_trabajadores_periodos (ruc, periodo) VALUES ('$ruc', $(escape_sql "$periodo"));"
                execute_sql "$sql" "Período trabajadores" "false"
            fi
        done <<< "$periodos"
    fi
    
    # Procesar detalle por período
    execute_sql "DELETE FROM detalle_trabajadores WHERE ruc = '$ruc';" "Limpiar detalle trabajadores" "false"
    local detalles=$(jq -c '.cantidad_trabajadores.detalle_por_periodo[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$detalles" ]]; then
        while IFS= read -r detalle_obj; do
            if [[ -n "$detalle_obj" ]]; then
                local periodo=$(echo "$detalle_obj" | jq -r '.periodo // ""')
                local cantidad_trabajadores=$(echo "$detalle_obj" | jq -r '.cantidad_trabajadores // 0')
                local cantidad_prestadores=$(echo "$detalle_obj" | jq -r '.cantidad_prestadores_servicio // 0')
                local cantidad_pensionistas=$(echo "$detalle_obj" | jq -r '.cantidad_pensionistas // 0')
                local total=$(echo "$detalle_obj" | jq -r '.total // 0')
                
                local sql="INSERT INTO detalle_trabajadores (ruc, periodo, cantidad_trabajadores, cantidad_prestadores_servicio, cantidad_pensionistas, total) 
                           VALUES ('$ruc', $(escape_sql "$periodo"), $cantidad_trabajadores, $cantidad_prestadores, $cantidad_pensionistas, $total);"
                execute_sql "$sql" "Detalle trabajadores" "false"
            fi
        done <<< "$detalles"
    fi
}

# Función MEJORADA para procesar representantes legales
process_representantes() {
    local json_file="$1"
    local ruc="$2"
    
    # Verificar si hay representantes
    local tiene_representantes=$(jq -r '.representantes_legales.representantes // empty' "$json_file")
    if [[ -z "$tiene_representantes" ]] || [[ "$tiene_representantes" == "null" ]]; then
        return 0
    fi
    
    info "Procesando representantes legales para RUC: $ruc"
    
    # Limpiar representantes existentes
    execute_sql "DELETE FROM representante_legal WHERE ruc = '$ruc';" "Limpiar representantes legales" "false"
    
    # Insertar registro principal
    local fecha_consulta=$(jq -r '.representantes_legales.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    local sql="INSERT INTO representantes_legales (ruc, fecha_consulta) 
               VALUES ('$ruc', $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET fecha_consulta = EXCLUDED.fecha_consulta;"
    execute_sql "$sql" "Representantes legales principal" "false"
    
    # Procesar cada representante
    local representantes=$(jq -c '.representantes_legales.representantes[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$representantes" ]]; then
        while IFS= read -r representante; do
            if [[ -n "$representante" ]]; then
                local tipo_documento=$(echo "$representante" | jq -r '.tipo_documento // ""')
                local numero_documento=$(echo "$representante" | jq -r '.numero_documento // ""')
                local nombre_completo=$(echo "$representante" | jq -r '.nombre_completo // ""')
                local cargo=$(echo "$representante" | jq -r '.cargo // ""')
                local fecha_desde=$(echo "$representante" | jq -r '.fecha_desde // ""')
                local fecha_hasta=$(echo "$representante" | jq -r '.fecha_hasta // ""')
                local vigente=$(echo "$representante" | jq -r '.vigente // false')
                
                local fecha_desde_sql=$(convert_date "$fecha_desde")
                local fecha_hasta_sql=$(convert_date "$fecha_hasta")
                
                local sql="INSERT INTO representante_legal (
                    ruc, tipo_documento, numero_documento, nombre_completo,
                    cargo, fecha_desde, fecha_hasta, vigente
                ) VALUES (
                    '$ruc', $(escape_sql "$tipo_documento"), $(escape_sql "$numero_documento"),
                    $(escape_sql "$nombre_completo"), $(escape_sql "$cargo"),
                    $fecha_desde_sql, $fecha_hasta_sql, $vigente
                );"
                
                execute_sql "$sql" "Representante legal" "false"
            fi
        done <<< "$representantes"
    fi
}

# Función MEJORADA para procesar deuda coactiva
process_deuda_coactiva() {
    local json_file="$1"
    local ruc="$2"
    
    local tiene_deuda=$(jq -r '.deuda_coactiva // empty' "$json_file")
    if [[ -z "$tiene_deuda" ]] || [[ "$tiene_deuda" == "null" ]]; then
        return 0
    fi
    
    info "Procesando deuda coactiva para RUC: $ruc"
    
    local total_deuda=$(jq -r '.deuda_coactiva.total_deuda // 0' "$json_file")
    local cantidad_documentos=$(jq -r '.deuda_coactiva.cantidad_documentos // 0' "$json_file")
    local fecha_consulta=$(jq -r '.deuda_coactiva.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO deuda_coactiva (ruc, total_deuda, cantidad_documentos, fecha_consulta)
               VALUES ('$ruc', $total_deuda, $cantidad_documentos, $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET
               total_deuda = EXCLUDED.total_deuda,
               cantidad_documentos = EXCLUDED.cantidad_documentos,
               fecha_consulta = EXCLUDED.fecha_consulta;"
    
    execute_sql "$sql" "Deuda coactiva para RUC $ruc" "false"
    
    # Procesar detalle de deudas si existe
    local deudas=$(jq -c '.deuda_coactiva.deudas[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$deudas" ]]; then
        execute_sql "DELETE FROM detalle_deuda WHERE ruc = '$ruc';" "Limpiar detalle deuda" "false"
        while IFS= read -r deuda_obj; do
            if [[ -n "$deuda_obj" ]]; then
                local monto=$(echo "$deuda_obj" | jq -r '.monto // 0')
                local periodo=$(echo "$deuda_obj" | jq -r '.periodo_tributario // ""')
                local fecha_inicio=$(echo "$deuda_obj" | jq -r '.fecha_inicio_cobranza // ""')
                local entidad=$(echo "$deuda_obj" | jq -r '.entidad // ""')
                
                local sql="INSERT INTO detalle_deuda (ruc, monto, periodo_tributario, fecha_inicio_cobranza, entidad) 
                           VALUES ('$ruc', $monto, $(escape_sql "$periodo"), $(escape_sql "$fecha_inicio"), $(escape_sql "$entidad"));"
                execute_sql "$sql" "Detalle deuda" "false"
            fi
        done <<< "$deudas"
    fi
}

# Función NUEVA para procesar omisiones tributarias
process_omisiones_tributarias() {
    local json_file="$1"
    local ruc="$2"
    
    local tiene_omisiones_obj=$(jq -r '.omisiones_tributarias // empty' "$json_file")
    if [[ -z "$tiene_omisiones_obj" ]] || [[ "$tiene_omisiones_obj" == "null" ]]; then
        return 0
    fi
    
    info "Procesando omisiones tributarias para RUC: $ruc"
    
    local tiene_omisiones=$(jq -r '.omisiones_tributarias.tiene_omisiones // false' "$json_file")
    local cantidad_omisiones=$(jq -r '.omisiones_tributarias.cantidad_omisiones // 0' "$json_file")
    local fecha_consulta=$(jq -r '.omisiones_tributarias.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO omisiones_tributarias (ruc, tiene_omisiones, cantidad_omisiones, fecha_consulta)
               VALUES ('$ruc', $tiene_omisiones, $cantidad_omisiones, $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET
               tiene_omisiones = EXCLUDED.tiene_omisiones,
               cantidad_omisiones = EXCLUDED.cantidad_omisiones,
               fecha_consulta = EXCLUDED.fecha_consulta;"
    
    execute_sql "$sql" "Omisiones tributarias para RUC $ruc" "false"
    
    # Procesar detalle de omisiones si existe
    local omisiones=$(jq -c '.omisiones_tributarias.omisiones[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$omisiones" ]]; then
        execute_sql "DELETE FROM omision WHERE ruc = '$ruc';" "Limpiar omisiones" "false"
        while IFS= read -r omision_obj; do
            if [[ -n "$omision_obj" ]]; then
                local periodo=$(echo "$omision_obj" | jq -r '.periodo // ""')
                local tributo=$(echo "$omision_obj" | jq -r '.tributo // ""')
                local tipo_declaracion=$(echo "$omision_obj" | jq -r '.tipo_declaracion // ""')
                local fecha_vencimiento=$(echo "$omision_obj" | jq -r '.fecha_vencimiento // ""')
                local estado=$(echo "$omision_obj" | jq -r '.estado // ""')
                
                local sql="INSERT INTO omision (ruc, periodo, tributo, tipo_declaracion, fecha_vencimiento, estado) 
                           VALUES ('$ruc', $(escape_sql "$periodo"), $(escape_sql "$tributo"), $(escape_sql "$tipo_declaracion"), $(escape_sql "$fecha_vencimiento"), $(escape_sql "$estado"));"
                execute_sql "$sql" "Omisión" "false"
            fi
        done <<< "$omisiones"
    fi
}

# Función NUEVA para procesar actas probatorias
process_actas_probatorias() {
    local json_file="$1"
    local ruc="$2"
    
    local tiene_actas_obj=$(jq -r '.actas_probatorias // empty' "$json_file")
    if [[ -z "$tiene_actas_obj" ]] || [[ "$tiene_actas_obj" == "null" ]]; then
        return 0
    fi
    
    info "Procesando actas probatorias para RUC: $ruc"
    
    local tiene_actas=$(jq -r '.actas_probatorias.tiene_actas // false' "$json_file")
    local cantidad_actas=$(jq -r '.actas_probatorias.cantidad_actas // 0' "$json_file")
    local fecha_consulta=$(jq -r '.actas_probatorias.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO actas_probatorias (ruc, tiene_actas, cantidad_actas, fecha_consulta)
               VALUES ('$ruc', $tiene_actas, $cantidad_actas, $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET
               tiene_actas = EXCLUDED.tiene_actas,
               cantidad_actas = EXCLUDED.cantidad_actas,
               fecha_consulta = EXCLUDED.fecha_consulta;"
    
    execute_sql "$sql" "Actas probatorias para RUC $ruc" "false"
    
    # Procesar detalle de actas si existe
    local actas=$(jq -c '.actas_probatorias.actas[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$actas" ]]; then
        execute_sql "DELETE FROM acta_probatoria WHERE ruc = '$ruc';" "Limpiar actas" "false"
        while IFS= read -r acta_obj; do
            if [[ -n "$acta_obj" ]]; then
                local numero_acta=$(echo "$acta_obj" | jq -r '.numero_acta // ""')
                local fecha_acta=$(echo "$acta_obj" | jq -r '.fecha_acta // ""')
                local lugar_intervencion=$(echo "$acta_obj" | jq -r '.lugar_intervencion // ""')
                local articulo_numeral=$(echo "$acta_obj" | jq -r '.articulo_numeral // ""')
                local descripcion_infraccion=$(echo "$acta_obj" | jq -r '.descripcion_infraccion // ""')
                local numero_ri_roz=$(echo "$acta_obj" | jq -r '.numero_ri_roz // ""')
                local tipo_ri_roz=$(echo "$acta_obj" | jq -r '.tipo_ri_roz // ""')
                local acta_reconocimiento=$(echo "$acta_obj" | jq -r '.acta_reconocimiento // ""')
                
                local sql="INSERT INTO acta_probatoria (ruc, numero_acta, fecha_acta, lugar_intervencion, articulo_numeral, descripcion_infraccion, numero_ri_roz, tipo_ri_roz, acta_reconocimiento) 
                           VALUES ('$ruc', $(escape_sql "$numero_acta"), $(escape_sql "$fecha_acta"), $(escape_sql "$lugar_intervencion"), $(escape_sql "$articulo_numeral"), $(escape_sql "$descripcion_infraccion"), $(escape_sql "$numero_ri_roz"), $(escape_sql "$tipo_ri_roz"), $(escape_sql "$acta_reconocimiento"));"
                execute_sql "$sql" "Acta probatoria" "false"
            fi
        done <<< "$actas"
    fi
}

# Función NUEVA para procesar facturas físicas
process_facturas_fisicas() {
    local json_file="$1"
    local ruc="$2"
    
    local tiene_facturas_obj=$(jq -r '.facturas_fisicas // empty' "$json_file")
    if [[ -z "$tiene_facturas_obj" ]] || [[ "$tiene_facturas_obj" == "null" ]]; then
        return 0
    fi
    
    info "Procesando facturas físicas para RUC: $ruc"
    
    local tiene_autorizacion=$(jq -r '.facturas_fisicas.tiene_autorizacion // false' "$json_file")
    local fecha_consulta=$(jq -r '.facturas_fisicas.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO facturas_fisicas (ruc, tiene_autorizacion, fecha_consulta)
               VALUES ('$ruc', $tiene_autorizacion, $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET
               tiene_autorizacion = EXCLUDED.tiene_autorizacion,
               fecha_consulta = EXCLUDED.fecha_consulta;"
    
    execute_sql "$sql" "Facturas físicas para RUC $ruc" "false"
    
    # Procesar autorizaciones si existe
    local autorizaciones=$(jq -c '.facturas_fisicas.autorizaciones[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$autorizaciones" ]]; then
        execute_sql "DELETE FROM factura_autorizada WHERE ruc = '$ruc';" "Limpiar facturas autorizadas" "false"
        while IFS= read -r auth_obj; do
            if [[ -n "$auth_obj" ]]; then
                local numero_autorizacion=$(echo "$auth_obj" | jq -r '.numero_autorizacion // ""')
                local fecha_autorizacion=$(echo "$auth_obj" | jq -r '.fecha_autorizacion // ""')
                local tipo_comprobante=$(echo "$auth_obj" | jq -r '.tipo_comprobante // ""')
                local serie=$(echo "$auth_obj" | jq -r '.serie // ""')
                local numero_inicial=$(echo "$auth_obj" | jq -r '.numero_inicial // ""')
                local numero_final=$(echo "$auth_obj" | jq -r '.numero_final // ""')
                
                local sql="INSERT INTO factura_autorizada (ruc, numero_autorizacion, fecha_autorizacion, tipo_comprobante, serie, numero_inicial, numero_final) 
                           VALUES ('$ruc', $(escape_sql "$numero_autorizacion"), $(escape_sql "$fecha_autorizacion"), $(escape_sql "$tipo_comprobante"), $(escape_sql "$serie"), $(escape_sql "$numero_inicial"), $(escape_sql "$numero_final"));"
                execute_sql "$sql" "Factura autorizada" "false"
            fi
        done <<< "$autorizaciones"
    fi
    
    # Procesar canceladas o bajas si existe
    local canceladas=$(jq -c '.facturas_fisicas.canceladas_o_bajas[]? // empty' "$json_file" 2>/dev/null)
    if [[ -n "$canceladas" ]]; then
        execute_sql "DELETE FROM factura_baja_o_cancelada WHERE ruc = '$ruc';" "Limpiar facturas canceladas" "false"
        while IFS= read -r canc_obj; do
            if [[ -n "$canc_obj" ]]; then
                local numero_autorizacion=$(echo "$canc_obj" | jq -r '.numero_autorizacion // ""')
                local fecha_autorizacion=$(echo "$canc_obj" | jq -r '.fecha_autorizacion // ""')
                local tipo_comprobante=$(echo "$canc_obj" | jq -r '.tipo_comprobante // ""')
                local serie=$(echo "$canc_obj" | jq -r '.serie // ""')
                local numero_inicial=$(echo "$canc_obj" | jq -r '.numero_inicial // ""')
                local numero_final=$(echo "$canc_obj" | jq -r '.numero_final // ""')
                
                local sql="INSERT INTO factura_baja_o_cancelada (ruc, numero_autorizacion, fecha_autorizacion, tipo_comprobante, serie, numero_inicial, numero_final) 
                           VALUES ('$ruc', $(escape_sql "$numero_autorizacion"), $(escape_sql "$fecha_autorizacion"), $(escape_sql "$tipo_comprobante"), $(escape_sql "$serie"), $(escape_sql "$numero_inicial"), $(escape_sql "$numero_final"));"
                execute_sql "$sql" "Factura cancelada" "false"
            fi
        done <<< "$canceladas"
    fi
}

# Función NUEVA para procesar reactiva perú
process_reactiva_peru() {
    local json_file="$1"
    local ruc="$2"
    
    local tiene_reactiva=$(jq -r '.reactiva_peru // empty' "$json_file")
    if [[ -z "$tiene_reactiva" ]] || [[ "$tiene_reactiva" == "null" ]]; then
        return 0
    fi
    
    info "Procesando reactiva perú para RUC: $ruc"
    
    local razon_social=$(jq -r '.reactiva_peru.razon_social // ""' "$json_file")
    local tiene_deuda_coactiva=$(jq -r '.reactiva_peru.tiene_deuda_coactiva // false' "$json_file")
    local fecha_actualizacion=$(jq -r '.reactiva_peru.fecha_actualizacion // ""' "$json_file")
    local referencia_legal=$(jq -r '.reactiva_peru.referencia_legal // ""' "$json_file")
    local fecha_consulta=$(jq -r '.reactiva_peru.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO reactiva_peru (ruc, razon_social, tiene_deuda_coactiva, fecha_actualizacion, referencia_legal, fecha_consulta)
               VALUES ('$ruc', $(escape_sql "$razon_social"), $tiene_deuda_coactiva, $(escape_sql "$fecha_actualizacion"), $(escape_sql "$referencia_legal"), $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET
               razon_social = EXCLUDED.razon_social,
               tiene_deuda_coactiva = EXCLUDED.tiene_deuda_coactiva,
               fecha_actualizacion = EXCLUDED.fecha_actualizacion,
               referencia_legal = EXCLUDED.referencia_legal,
               fecha_consulta = EXCLUDED.fecha_consulta;"
    
    execute_sql "$sql" "Reactiva Perú para RUC $ruc" "false"
}

# Función NUEVA para procesar programa covid19
process_programa_covid19() {
    local json_file="$1"
    local ruc="$2"
    
    local tiene_programa=$(jq -r '.programa_covid19 // empty' "$json_file")
    if [[ -z "$tiene_programa" ]] || [[ "$tiene_programa" == "null" ]]; then
        return 0
    fi
    
    info "Procesando programa covid19 para RUC: $ruc"
    
    local razon_social=$(jq -r '.programa_covid19.razon_social // ""' "$json_file")
    local participa_programa=$(jq -r '.programa_covid19.participa_programa // false' "$json_file")
    local tiene_deuda_coactiva=$(jq -r '.programa_covid19.tiene_deuda_coactiva // false' "$json_file")
    local fecha_actualizacion=$(jq -r '.programa_covid19.fecha_actualizacion // ""' "$json_file")
    local base_legal=$(jq -r '.programa_covid19.base_legal // ""' "$json_file")
    local fecha_consulta=$(jq -r '.programa_covid19.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO programa_covid19 (ruc, razon_social, participa_programa, tiene_deuda_coactiva, fecha_actualizacion, base_legal, fecha_consulta)
               VALUES ('$ruc', $(escape_sql "$razon_social"), $participa_programa, $tiene_deuda_coactiva, $(escape_sql "$fecha_actualizacion"), $(escape_sql "$base_legal"), $fecha_consulta_sql)
               ON CONFLICT (ruc) DO UPDATE SET
               razon_social = EXCLUDED.razon_social,
               participa_programa = EXCLUDED.participa_programa,
               tiene_deuda_coactiva = EXCLUDED.tiene_deuda_coactiva,
               fecha_actualizacion = EXCLUDED.fecha_actualizacion,
               base_legal = EXCLUDED.base_legal,
               fecha_consulta = EXCLUDED.fecha_consulta;"
    
    execute_sql "$sql" "Programa COVID19 para RUC $ruc" "false"
}

# Función NUEVA para actualizar tabla de metadata
update_ruc_completo() {
    local json_file="$1"
    local ruc="$2"
    
    # Verificar qué secciones tienen información
    local tiene_historica=$(jq -r 'if (.informacion_historica.condiciones | length) > 0 or (.informacion_historica.domicilios | length) > 0 then true else false end' "$json_file" 2>/dev/null || echo "false")
    local tiene_deuda=$(jq -r 'if .deuda_coactiva.cantidad_documentos > 0 then true else false end' "$json_file" 2>/dev/null || echo "false")
    local tiene_omisiones=$(jq -r '.omisiones_tributarias.tiene_omisiones // false' "$json_file" 2>/dev/null || echo "false")
    local tiene_trabajadores=$(jq -r 'if (.cantidad_trabajadores.detalle_por_periodo | length) > 0 then true else false end' "$json_file" 2>/dev/null || echo "false")
    local tiene_actas=$(jq -r '.actas_probatorias.tiene_actas // false' "$json_file" 2>/dev/null || echo "false")
    local tiene_facturas=$(jq -r '.facturas_fisicas.tiene_autorizacion // false' "$json_file" 2>/dev/null || echo "false")
    local tiene_reactiva=$(jq -r 'if .reactiva_peru then true else false end' "$json_file" 2>/dev/null || echo "false")
    local tiene_covid=$(jq -r 'if .programa_covid19 then true else false end' "$json_file" 2>/dev/null || echo "false")
    local tiene_representantes=$(jq -r 'if (.representantes_legales.representantes | length) > 0 then true else false end' "$json_file" 2>/dev/null || echo "false")
    
    local version_api=$(jq -r '.version_api // "1.0.0"' "$json_file")
    local fecha_consulta=$(jq -r '.fecha_consulta // ""' "$json_file")
    local fecha_consulta_sql=$(convert_timestamp "$fecha_consulta")
    
    local sql="INSERT INTO ruc_completo (
        ruc, fecha_consulta, version_api, tiene_informacion_historica, 
        tiene_deuda_coactiva, tiene_omisiones_tributarias, tiene_cantidad_trabajadores,
        tiene_actas_probatorias, tiene_facturas_fisicas, tiene_reactiva_peru,
        tiene_programa_covid19, tiene_representantes_legales
    ) VALUES (
        '$ruc', $fecha_consulta_sql, $(escape_sql "$version_api"), $tiene_historica,
        $tiene_deuda, $tiene_omisiones, $tiene_trabajadores,
        $tiene_actas, $tiene_facturas, $tiene_reactiva,
        $tiene_covid, $tiene_representantes
    ) ON CONFLICT (ruc) DO UPDATE SET
        fecha_consulta = EXCLUDED.fecha_consulta,
        version_api = EXCLUDED.version_api,
        tiene_informacion_historica = EXCLUDED.tiene_informacion_historica,
        tiene_deuda_coactiva = EXCLUDED.tiene_deuda_coactiva,
        tiene_omisiones_tributarias = EXCLUDED.tiene_omisiones_tributarias,
        tiene_cantidad_trabajadores = EXCLUDED.tiene_cantidad_trabajadores,
        tiene_actas_probatorias = EXCLUDED.tiene_actas_probatorias,
        tiene_facturas_fisicas = EXCLUDED.tiene_facturas_fisicas,
        tiene_reactiva_peru = EXCLUDED.tiene_reactiva_peru,
        tiene_programa_covid19 = EXCLUDED.tiene_programa_covid19,
        tiene_representantes_legales = EXCLUDED.tiene_representantes_legales;"
    
    execute_sql "$sql" "Metadata completa para RUC $ruc" "false"
}

# Función principal de procesamiento MEJORADA
process_json_file() {
    local json_file="$1"
    
    if [[ ! -f "$json_file" ]]; then
        error "Archivo no encontrado: $json_file"
        ((TOTAL_ERRORS++))
        return 1
    fi
    
    # Verificar que el archivo JSON es válido
    if ! jq . "$json_file" > /dev/null 2>&1; then
        error "JSON inválido en archivo: $json_file"
        ((TOTAL_ERRORS++))
        return 1
    fi
    
    local ruc=$(jq -r '.informacion_basica.ruc // empty' "$json_file")
    if [[ -z "$ruc" ]]; then
        error "RUC no encontrado en archivo: $json_file"
        ((TOTAL_ERRORS++))
        return 1
    fi
    
    log "========================================="
    log "Procesando archivo: $(basename "$json_file")"
    log "RUC: $ruc"
    log "========================================="
    
    # Iniciar transacción para consistencia
    if ! execute_sql "BEGIN;" "Iniciar transacción" "false"; then
        error "No se pudo iniciar transacción para RUC $ruc"
        ((TOTAL_ERRORS++))
        return 1
    fi
    
    local success=true
    
    # Procesar todas las secciones
    if ! process_basic_info "$json_file"; then
        success=false
    fi
    
    if [[ "$success" == "true" ]]; then
        process_informacion_historica "$json_file" "$ruc"
        process_deuda_coactiva "$json_file" "$ruc"
        process_omisiones_tributarias "$json_file" "$ruc"
        process_cantidad_trabajadores "$json_file" "$ruc"
        process_actas_probatorias "$json_file" "$ruc"
        process_facturas_fisicas "$json_file" "$ruc"
        process_reactiva_peru "$json_file" "$ruc"
        process_programa_covid19 "$json_file" "$ruc"
        process_representantes "$json_file" "$ruc"
        update_ruc_completo "$json_file" "$ruc"
    fi
    
    if [[ "$success" == "true" ]]; then
        if execute_sql "COMMIT;" "Confirmar transacción" "false"; then
            log "✓ Archivo procesado exitosamente: $(basename "$json_file")"
            ((TOTAL_PROCESSED++))
            return 0
        else
            error "Error confirmando transacción para RUC $ruc"
            execute_sql "ROLLBACK;" "Revertir transacción" "false"
            ((TOTAL_ERRORS++))
            return 1
        fi
    else
        error "✗ Error procesando archivo: $(basename "$json_file")"
        execute_sql "ROLLBACK;" "Revertir transacción" "false"
        ((TOTAL_ERRORS++))
        return 1
    fi
}

# Función para verificar y crear índices si no existen
ensure_indexes() {
    log "Verificando y creando índices..."
    
    local indexes=(
        "CREATE INDEX IF NOT EXISTS idx_ruc_info_ruc ON ruc_info(ruc);"
        "CREATE INDEX IF NOT EXISTS idx_ruc_info_razon_social ON ruc_info(razon_social);"
        "CREATE INDEX IF NOT EXISTS idx_ruc_info_estado ON ruc_info(estado);"
        "CREATE INDEX IF NOT EXISTS idx_deuda_coactiva_ruc ON deuda_coactiva(ruc);"
        "CREATE INDEX IF NOT EXISTS idx_representante_legal_ruc ON representante_legal(ruc);"
        "CREATE INDEX IF NOT EXISTS idx_representante_legal_documento ON representante_legal(numero_documento);"
    )
    
    for index_sql in "${indexes[@]}"; do
        execute_sql "$index_sql" "Crear índice" "false"
    done
    
    log "✓ Índices verificados"
}

# Función principal
main() {
    log "========================================="
    log "SCRIPT DE IMPORTACIÓN MEJORADO - v2.0"
    log "========================================="
    
    # Verificar dependencias
    check_dependencies
    
    # Verificar conexión a la base de datos
    check_db_connection
    
    # Asegurar índices
    ensure_indexes
    
    # Verificar que el directorio existe
    if [[ ! -d "$JSON_DIR" ]]; then
        error "Directorio no encontrado: $JSON_DIR"
        exit 1
    fi
    
    # Encontrar archivos JSON
    local json_files=($(find "$JSON_DIR" -name "*.json" -type f | sort))
    local total_files=${#json_files[@]}
    
    if [[ $total_files -eq 0 ]]; then
        warning "No se encontraron archivos JSON en $JSON_DIR"
        exit 0
    fi
    
    log "Encontrados $total_files archivos JSON para procesar"
    log "========================================="
    
    # Procesar cada archivo
    local start_time=$(date +%s)
    
    for json_file in "${json_files[@]}"; do
        process_json_file "$json_file"
        
        # Mostrar progreso cada 10 archivos
        if (( (TOTAL_PROCESSED + TOTAL_ERRORS) % 10 == 0 )); then
            local current=$(( TOTAL_PROCESSED + TOTAL_ERRORS ))
            local percentage=$(( current * 100 / total_files ))
            info "Progreso: $current/$total_files ($percentage%) - Procesados: $TOTAL_PROCESSED, Errores: $TOTAL_ERRORS"
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$(( end_time - start_time ))
    
    # Resumen final
    log "========================================="
    log "RESUMEN FINAL DE IMPORTACIÓN"
    log "========================================="
    log "Total de archivos encontrados: $total_files"
    log "Procesados exitosamente: $TOTAL_PROCESSED"
    log "Con errores: $TOTAL_ERRORS"
    log "Omitidos: $TOTAL_SKIPPED"
    log "Tiempo total: ${duration}s"
    log "========================================="
    
    if [[ $TOTAL_ERRORS -gt 0 ]]; then
        warning "Se completó la importación con $TOTAL_ERRORS errores"
        exit 1
    else
        log "🎉 Importación completada exitosamente"
        exit 0
    fi
}

# Ejecutar función principal
main "$@"