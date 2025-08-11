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

# Configurar PGPASSWORD para evitar prompts de contraseña
export PGPASSWORD="$DB_PASSWORD"

# Función para ejecutar consultas SQL
execute_sql() {
    local query="$1"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$query" -q
}

# Función para convertir fecha del formato DD/MM/YYYY a YYYY-MM-DD
convert_date() {
    local date_str="$1"
    if [[ "$date_str" == "" || "$date_str" == "-" || "$date_str" == "null" ]]; then
        echo "NULL"
    else
        # Convertir DD/MM/YYYY a YYYY-MM-DD
        echo "'$(echo "$date_str" | awk -F'/' '{print $3"-"$2"-"$1}')'"
    fi
}

# Función para limpiar y escapar strings
clean_string() {
    local str="$1"
    if [[ "$str" == "" || "$str" == "null" || "$str" == "-" ]]; then
        echo "NULL"
    else
        # Escapar comillas simples duplicándolas
        echo "'$(echo "$str" | sed "s/'/''/g")'"
    fi
}

# Función para convertir booleanos
convert_boolean() {
    local bool_val="$1"
    case "$bool_val" in
        "true"|"SÍ"|"SI") echo "true" ;;
        "false"|"NO") echo "false" ;;
        *) echo "NULL" ;;
    esac
}

# Función para procesar un archivo JSON
process_json_file() {
    local file="$1"
    echo "Procesando: $file"
    
    # Extraer datos básicos del JSON usando jq
    local ruc=$(jq -r '.informacion_basica.ruc // empty' "$file")
    
    if [[ -z "$ruc" ]]; then
        echo "Error: No se pudo extraer RUC de $file"
        return 1
    fi
    
    echo "Procesando RUC: $ruc"
    
    # Extraer información básica
    local razon_social=$(jq -r '.informacion_basica.razon_social // ""' "$file")
    local tipo_contribuyente=$(jq -r '.informacion_basica.tipo_contribuyente // ""' "$file")
    local nombre_comercial=$(jq -r '.informacion_basica.nombre_comercial // ""' "$file")
    local fecha_inscripcion=$(jq -r '.informacion_basica.fecha_inscripcion // ""' "$file")
    local fecha_inicio_actividades=$(jq -r '.informacion_basica.fecha_inicio_actividades // ""' "$file")
    local estado=$(jq -r '.informacion_basica.estado // ""' "$file")
    local condicion=$(jq -r '.informacion_basica.condicion // ""' "$file")
    local domicilio_fiscal=$(jq -r '.informacion_basica.domicilio_fiscal // ""' "$file")
    local sistema_emision=$(jq -r '.informacion_basica.sistema_emision // ""' "$file")
    local actividad_comercio_exterior=$(jq -r '.informacion_basica.actividad_comercio_exterior // ""' "$file")
    local sistema_contabilidad=$(jq -r '.informacion_basica.sistema_contabilidad // ""' "$file")
    local emisor_electronico_desde=$(jq -r '.informacion_basica.emisor_electronico_desde // ""' "$file")
    local afiliado_ple=$(jq -r '.informacion_basica.afiliado_ple // ""' "$file")
    local fecha_consulta=$(jq -r '.fecha_consulta // ""' "$file")
    local version_api=$(jq -r '.version_api // ""' "$file")
    
    # Insertar información básica
    local insert_basic="INSERT INTO ruc_informacion_basica (
        ruc, razon_social, tipo_contribuyente, nombre_comercial, fecha_inscripcion,
        fecha_inicio_actividades, estado, condicion, domicilio_fiscal, sistema_emision,
        actividad_comercio_exterior, sistema_contabilidad, emisor_electronico_desde, afiliado_ple
    ) VALUES (
        '$ruc',
        $(clean_string "$razon_social"),
        $(clean_string "$tipo_contribuyente"),
        $(clean_string "$nombre_comercial"),
        $(convert_date "$fecha_inscripcion"),
        $(convert_date "$fecha_inicio_actividades"),
        $(clean_string "$estado"),
        $(clean_string "$condicion"),
        $(clean_string "$domicilio_fiscal"),
        $(clean_string "$sistema_emision"),
        $(clean_string "$actividad_comercio_exterior"),
        $(clean_string "$sistema_contabilidad"),
        $(convert_date "$emisor_electronico_desde"),
        $(clean_string "$afiliado_ple")
    ) ON CONFLICT (ruc) DO UPDATE SET
        razon_social = EXCLUDED.razon_social,
        tipo_contribuyente = EXCLUDED.tipo_contribuyente,
        nombre_comercial = EXCLUDED.nombre_comercial,
        fecha_inscripcion = EXCLUDED.fecha_inscripcion,
        fecha_inicio_actividades = EXCLUDED.fecha_inicio_actividades,
        estado = EXCLUDED.estado,
        condicion = EXCLUDED.condicion,
        domicilio_fiscal = EXCLUDED.domicilio_fiscal,
        sistema_emision = EXCLUDED.sistema_emision,
        actividad_comercio_exterior = EXCLUDED.actividad_comercio_exterior,
        sistema_contabilidad = EXCLUDED.sistema_contabilidad,
        emisor_electronico_desde = EXCLUDED.emisor_electronico_desde,
        afiliado_ple = EXCLUDED.afiliado_ple,
        updated_at = CURRENT_TIMESTAMP;"
    
    execute_sql "$insert_basic"
    
    # Obtener el ID del RUC insertado
    local ruc_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_informacion_basica WHERE ruc = '$ruc';" | xargs)
    
    if [[ -z "$ruc_id" ]]; then
        echo "Error: No se pudo obtener ID para RUC $ruc"
        return 1
    fi
    
    # Limpiar datos anteriores relacionados
    execute_sql "DELETE FROM ruc_actividades_economicas WHERE ruc_id = $ruc_id;"
    execute_sql "DELETE FROM ruc_comprobantes_pago WHERE ruc_id = $ruc_id;"
    execute_sql "DELETE FROM ruc_sistemas_emision_electronica WHERE ruc_id = $ruc_id;"
    execute_sql "DELETE FROM ruc_comprobantes_electronicos WHERE ruc_id = $ruc_id;"
    execute_sql "DELETE FROM ruc_padrones WHERE ruc_id = $ruc_id;"
    
    # Insertar actividades económicas
    jq -r '.informacion_basica.actividades_economicas[]? // empty' "$file" | while read -r actividad; do
        if [[ -n "$actividad" && "$actividad" != "null" ]]; then
            execute_sql "INSERT INTO ruc_actividades_economicas (ruc_id, actividad_economica) VALUES ($ruc_id, $(clean_string "$actividad"));"
        fi
    done
    
    # Insertar comprobantes de pago
    jq -r '.informacion_basica.comprobantes_pago[]? // empty' "$file" | while read -r comprobante; do
        if [[ -n "$comprobante" && "$comprobante" != "null" ]]; then
            execute_sql "INSERT INTO ruc_comprobantes_pago (ruc_id, comprobante_pago) VALUES ($ruc_id, $(clean_string "$comprobante"));"
        fi
    done
    
    # Insertar sistemas de emisión electrónica
    jq -r '.informacion_basica.sistema_emision_electronica[]? // empty' "$file" | while read -r sistema; do
        if [[ -n "$sistema" && "$sistema" != "null" ]]; then
            execute_sql "INSERT INTO ruc_sistemas_emision_electronica (ruc_id, sistema_emision) VALUES ($ruc_id, $(clean_string "$sistema"));"
        fi
    done
    
    # Insertar comprobantes electrónicos
    jq -r '.informacion_basica.comprobantes_electronicos[]? // empty' "$file" | while read -r comprobante; do
        if [[ -n "$comprobante" && "$comprobante" != "null" ]]; then
            execute_sql "INSERT INTO ruc_comprobantes_electronicos (ruc_id, comprobante_electronico) VALUES ($ruc_id, $(clean_string "$comprobante"));"
        fi
    done
    
    # Insertar padrones
    jq -r '.informacion_basica.padrones[]? // empty' "$file" | while read -r padron; do
        if [[ -n "$padron" && "$padron" != "null" ]]; then
            execute_sql "INSERT INTO ruc_padrones (ruc_id, padron) VALUES ($ruc_id, $(clean_string "$padron"));"
        fi
    done
    
    # Insertar consulta
    if [[ -n "$fecha_consulta" && "$fecha_consulta" != "null" ]]; then
        execute_sql "INSERT INTO ruc_consultas (ruc_id, fecha_consulta, version_api) VALUES ($ruc_id, '$fecha_consulta', $(clean_string "$version_api"));"
    fi
    
    # Procesar información histórica si existe
    if jq -e '.informacion_historica' "$file" > /dev/null 2>&1; then
        execute_sql "DELETE FROM ruc_informacion_historica WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_informacion_historica (ruc_id) VALUES ($ruc_id);"
        local hist_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_informacion_historica WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Razones sociales históricas
        jq -c '.informacion_historica.razones_sociales[]? // empty' "$file" | while read -r razon; do
            local nombre=$(echo "$razon" | jq -r '.nombre // ""')
            local fecha_baja=$(echo "$razon" | jq -r '.fecha_de_baja // ""')
            execute_sql "INSERT INTO ruc_razones_sociales_historicas (informacion_historica_id, nombre, fecha_de_baja) VALUES ($hist_id, $(clean_string "$nombre"), $(convert_date "$fecha_baja"));"
        done
        
        # Condiciones históricas
        jq -c '.informacion_historica.condiciones[]? // empty' "$file" | while read -r condicion_hist; do
            local condicion=$(echo "$condicion_hist" | jq -r '.condicion // ""')
            local desde=$(echo "$condicion_hist" | jq -r '.desde // ""')
            local hasta=$(echo "$condicion_hist" | jq -r '.hasta // ""')
            execute_sql "INSERT INTO ruc_condiciones_historicas (informacion_historica_id, condicion, desde, hasta) VALUES ($hist_id, $(clean_string "$condicion"), $(convert_date "$desde"), $(convert_date "$hasta"));"
        done
        
        # Domicilios fiscales históricos
        jq -c '.informacion_historica.domicilios[]? // empty' "$file" | while read -r domicilio; do
            local direccion=$(echo "$domicilio" | jq -r '.direccion // ""')
            local fecha_baja=$(echo "$domicilio" | jq -r '.fecha_de_baja // ""')
            execute_sql "INSERT INTO ruc_domicilios_fiscales_historicos (informacion_historica_id, direccion, fecha_de_baja) VALUES ($hist_id, $(clean_string "$direccion"), $(convert_date "$fecha_baja"));"
        done
    fi
    
    # Procesar deuda coactiva si existe
    if jq -e '.deuda_coactiva' "$file" > /dev/null 2>&1; then
        local total_deuda=$(jq -r '.deuda_coactiva.total_deuda // 0' "$file")
        local cantidad_documentos=$(jq -r '.deuda_coactiva.cantidad_documentos // 0' "$file")
        
        execute_sql "DELETE FROM ruc_deuda_coactiva WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_deuda_coactiva (ruc_id, total_deuda, cantidad_documentos) VALUES ($ruc_id, $total_deuda, $cantidad_documentos);"
        local deuda_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_deuda_coactiva WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Detalle de deudas
        jq -c '.deuda_coactiva.deudas[]? // empty' "$file" | while read -r deuda; do
            local monto=$(echo "$deuda" | jq -r '.monto // 0')
            local periodo=$(echo "$deuda" | jq -r '.periodo_tributario // ""')
            local fecha_inicio=$(echo "$deuda" | jq -r '.fecha_inicio_cobranza // ""')
            local entidad=$(echo "$deuda" | jq -r '.entidad // ""')
            execute_sql "INSERT INTO ruc_detalle_deudas (deuda_coactiva_id, monto, periodo_tributario, fecha_inicio_cobranza, entidad) VALUES ($deuda_id, $monto, $(clean_string "$periodo"), $(convert_date "$fecha_inicio"), $(clean_string "$entidad"));"
        done
    fi
    
    # Procesar omisiones tributarias si existe
    if jq -e '.omisiones_tributarias' "$file" > /dev/null 2>&1; then
        local tiene_omisiones=$(jq -r '.omisiones_tributarias.tiene_omisiones // false' "$file")
        local cantidad_omisiones=$(jq -r '.omisiones_tributarias.cantidad_omisiones // 0' "$file")
        
        execute_sql "DELETE FROM ruc_omisiones_tributarias WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_omisiones_tributarias (ruc_id, tiene_omisiones, cantidad_omisiones) VALUES ($ruc_id, $(convert_boolean "$tiene_omisiones"), $cantidad_omisiones);"
        local omisiones_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_omisiones_tributarias WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Detalle de omisiones
        jq -c '.omisiones_tributarias.omisiones[]? // empty' "$file" | while read -r omision; do
            local periodo=$(echo "$omision" | jq -r '.periodo // ""')
            local tributo=$(echo "$omision" | jq -r '.tributo // ""')
            local tipo_declaracion=$(echo "$omision" | jq -r '.tipo_declaracion // ""')
            local fecha_vencimiento=$(echo "$omision" | jq -r '.fecha_vencimiento // ""')
            local estado=$(echo "$omision" | jq -r '.estado // ""')
            execute_sql "INSERT INTO ruc_omisiones (omisiones_tributarias_id, periodo, tributo, tipo_declaracion, fecha_vencimiento, estado) VALUES ($omisiones_id, $(clean_string "$periodo"), $(clean_string "$tributo"), $(clean_string "$tipo_declaracion"), $(convert_date "$fecha_vencimiento"), $(clean_string "$estado"));"
        done
    fi
    
    # Procesar cantidad de trabajadores si existe
    if jq -e '.cantidad_trabajadores' "$file" > /dev/null 2>&1; then
        execute_sql "DELETE FROM ruc_cantidad_trabajadores WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_cantidad_trabajadores (ruc_id) VALUES ($ruc_id);"
        local trabajadores_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_cantidad_trabajadores WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Periodos disponibles
        jq -r '.cantidad_trabajadores.periodos_disponibles[]? // empty' "$file" | while read -r periodo; do
            if [[ -n "$periodo" && "$periodo" != "null" ]]; then
                execute_sql "INSERT INTO ruc_periodos_disponibles_trabajadores (cantidad_trabajadores_id, periodo) VALUES ($trabajadores_id, $(clean_string "$periodo"));"
            fi
        done
        
        # Detalle por periodo
        jq -c '.cantidad_trabajadores.detalle_por_periodo[]? // empty' "$file" | while read -r detalle; do
            local periodo=$(echo "$detalle" | jq -r '.periodo // ""')
            local cant_trabajadores=$(echo "$detalle" | jq -r '.cantidad_trabajadores // 0')
            local cant_prestadores=$(echo "$detalle" | jq -r '.cantidad_prestadores_servicio // 0')
            local cant_pensionistas=$(echo "$detalle" | jq -r '.cantidad_pensionistas // 0')
            local total=$(echo "$detalle" | jq -r '.total // 0')
            execute_sql "INSERT INTO ruc_detalle_trabajadores (cantidad_trabajadores_id, periodo, cantidad_trabajadores, cantidad_prestadores_servicio, cantidad_pensionistas, total) VALUES ($trabajadores_id, $(clean_string "$periodo"), $cant_trabajadores, $cant_prestadores, $cant_pensionistas, $total);"
        done
    fi
    
    # Procesar actas probatorias si existe
    if jq -e '.actas_probatorias' "$file" > /dev/null 2>&1; then
        local tiene_actas=$(jq -r '.actas_probatorias.tiene_actas // false' "$file")
        local cantidad_actas=$(jq -r '.actas_probatorias.cantidad_actas // 0' "$file")
        
        execute_sql "DELETE FROM ruc_actas_probatorias WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_actas_probatorias (ruc_id, tiene_actas, cantidad_actas) VALUES ($ruc_id, $(convert_boolean "$tiene_actas"), $cantidad_actas);"
        local actas_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_actas_probatorias WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Detalle de actas
        jq -c '.actas_probatorias.actas[]? // empty' "$file" | while read -r acta; do
            local numero_acta=$(echo "$acta" | jq -r '.numero_acta // ""')
            local fecha_acta=$(echo "$acta" | jq -r '.fecha_acta // ""')
            local lugar_intervencion=$(echo "$acta" | jq -r '.lugar_intervencion // ""')
            local articulo_numeral=$(echo "$acta" | jq -r '.articulo_numeral // ""')
            local descripcion_infraccion=$(echo "$acta" | jq -r '.descripcion_infraccion // ""')
            local numero_ri_roz=$(echo "$acta" | jq -r '.numero_ri_roz // ""')
            local tipo_ri_roz=$(echo "$acta" | jq -r '.tipo_ri_roz // ""')
            local acta_reconocimiento=$(echo "$acta" | jq -r '.acta_reconocimiento // ""')
            execute_sql "INSERT INTO ruc_actas (actas_probatorias_id, numero_acta, fecha_acta, lugar_intervencion, articulo_numeral, descripcion_infraccion, numero_ri_roz, tipo_ri_roz, acta_reconocimiento) VALUES ($actas_id, $(clean_string "$numero_acta"), $(convert_date "$fecha_acta"), $(clean_string "$lugar_intervencion"), $(clean_string "$articulo_numeral"), $(clean_string "$descripcion_infraccion"), $(clean_string "$numero_ri_roz"), $(clean_string "$tipo_ri_roz"), $(clean_string "$acta_reconocimiento"));"
        done
    fi
    
    # Procesar facturas físicas si existe
    if jq -e '.facturas_fisicas' "$file" > /dev/null 2>&1; then
        local tiene_autorizacion=$(jq -r '.facturas_fisicas.tiene_autorizacion // false' "$file")
        
        execute_sql "DELETE FROM ruc_facturas_fisicas WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_facturas_fisicas (ruc_id, tiene_autorizacion) VALUES ($ruc_id, $(convert_boolean "$tiene_autorizacion"));"
        local facturas_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_facturas_fisicas WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Facturas autorizadas
        jq -c '.facturas_fisicas.autorizaciones[]? // empty' "$file" | while read -r factura; do
            local numero_autorizacion=$(echo "$factura" | jq -r '.numero_autorizacion // ""')
            local fecha_autorizacion=$(echo "$factura" | jq -r '.fecha_autorizacion // ""')
            local tipo_comprobante=$(echo "$factura" | jq -r '.tipo_comprobante // ""')
            local serie=$(echo "$factura" | jq -r '.serie // ""')
            local numero_inicial=$(echo "$factura" | jq -r '.numero_inicial // ""')
            local numero_final=$(echo "$factura" | jq -r '.numero_final // ""')
            execute_sql "INSERT INTO ruc_facturas_autorizadas (facturas_fisicas_id, numero_autorizacion, fecha_autorizacion, tipo_comprobante, serie, numero_inicial, numero_final) VALUES ($facturas_id, $(clean_string "$numero_autorizacion"), $(convert_date "$fecha_autorizacion"), $(clean_string "$tipo_comprobante"), $(clean_string "$serie"), $(clean_string "$numero_inicial"), $(clean_string "$numero_final"));"
        done
        
        # Facturas canceladas o de baja
        jq -c '.facturas_fisicas.canceladas_o_bajas[]? // empty' "$file" | while read -r factura; do
            local numero_autorizacion=$(echo "$factura" | jq -r '.numero_autorizacion // ""')
            local fecha_autorizacion=$(echo "$factura" | jq -r '.fecha_autorizacion // ""')
            local tipo_comprobante=$(echo "$factura" | jq -r '.tipo_comprobante // ""')
            local serie=$(echo "$factura" | jq -r '.serie // ""')
            local numero_inicial=$(echo "$factura" | jq -r '.numero_inicial // ""')
            local numero_final=$(echo "$factura" | jq -r '.numero_final // ""')
            execute_sql "INSERT INTO ruc_facturas_canceladas_bajas (facturas_fisicas_id, numero_autorizacion, fecha_autorizacion, tipo_comprobante, serie, numero_inicial, numero_final) VALUES ($facturas_id, $(clean_string "$numero_autorizacion"), $(convert_date "$fecha_autorizacion"), $(clean_string "$tipo_comprobante"), $(clean_string "$serie"), $(clean_string "$numero_inicial"), $(clean_string "$numero_final"));"
        done
    fi
    
    # Procesar Reactiva Perú si existe
    if jq -e '.reactiva_peru' "$file" > /dev/null 2>&1; then
        local razon_social_reactiva=$(jq -r '.reactiva_peru.razon_social // ""' "$file")
        local tiene_deuda_coactiva=$(jq -r '.reactiva_peru.tiene_deuda_coactiva // false' "$file")
        local fecha_actualizacion=$(jq -r '.reactiva_peru.fecha_actualizacion // ""' "$file")
        local referencia_legal=$(jq -r '.reactiva_peru.referencia_legal // ""' "$file")
        
        execute_sql "DELETE FROM ruc_reactiva_peru WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_reactiva_peru (ruc_id, razon_social, tiene_deuda_coactiva, fecha_actualizacion, referencia_legal) VALUES ($ruc_id, $(clean_string "$razon_social_reactiva"), $(convert_boolean "$tiene_deuda_coactiva"), $(convert_date "$fecha_actualizacion"), $(clean_string "$referencia_legal"));"
    fi
    
    # Procesar Programa COVID-19 si existe
    if jq -e '.programa_covid19' "$file" > /dev/null 2>&1; then
        local razon_social_covid=$(jq -r '.programa_covid19.razon_social // ""' "$file")
        local participa_programa=$(jq -r '.programa_covid19.participa_programa // false' "$file")
        local tiene_deuda_coactiva_covid=$(jq -r '.programa_covid19.tiene_deuda_coactiva // false' "$file")
        local fecha_actualizacion_covid=$(jq -r '.programa_covid19.fecha_actualizacion // ""' "$file")
        local base_legal=$(jq -r '.programa_covid19.base_legal // ""' "$file")
        
        execute_sql "DELETE FROM ruc_programa_covid19 WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_programa_covid19 (ruc_id, razon_social, participa_programa, tiene_deuda_coactiva, fecha_actualizacion, base_legal) VALUES ($ruc_id, $(clean_string "$razon_social_covid"), $(convert_boolean "$participa_programa"), $(convert_boolean "$tiene_deuda_coactiva_covid"), $(convert_date "$fecha_actualizacion_covid"), $(clean_string "$base_legal"));"
    fi
    
    # Procesar representantes legales si existe
    if jq -e '.representantes_legales' "$file" > /dev/null 2>&1; then
        execute_sql "DELETE FROM ruc_representantes_legales WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_representantes_legales (ruc_id) VALUES ($ruc_id);"
        local representantes_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_representantes_legales WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Detalle de representantes
        jq -c '.representantes_legales.representantes[]? // empty' "$file" | while read -r representante; do
            local tipo_documento=$(echo "$representante" | jq -r '.tipo_documento // ""')
            local numero_documento=$(echo "$representante" | jq -r '.numero_documento // ""')
            local nombre_completo=$(echo "$representante" | jq -r '.nombre_completo // ""')
            local cargo=$(echo "$representante" | jq -r '.cargo // ""')
            local fecha_desde=$(echo "$representante" | jq -r '.fecha_desde // ""')
            local fecha_hasta=$(echo "$representante" | jq -r '.fecha_hasta // ""')
            local vigente=$(echo "$representante" | jq -r '.vigente // false')
            execute_sql "INSERT INTO ruc_representantes (representantes_legales_id, tipo_documento, numero_documento, nombre_completo, cargo, fecha_desde, fecha_hasta, vigente) VALUES ($representantes_id, $(clean_string "$tipo_documento"), $(clean_string "$numero_documento"), $(clean_string "$nombre_completo"), $(clean_string "$cargo"), $(convert_date "$fecha_desde"), $(convert_date "$fecha_hasta"), $(convert_boolean "$vigente"));"
        done
    fi
    
    # Procesar establecimientos anexos si existe
    if jq -e '.establecimientos_anexos' "$file" > /dev/null 2>&1; then
        local cantidad_anexos=$(jq -r '.establecimientos_anexos.cantidad_anexos // 0' "$file")
        
        execute_sql "DELETE FROM ruc_establecimientos_anexos WHERE ruc_id = $ruc_id;"
        execute_sql "INSERT INTO ruc_establecimientos_anexos (ruc_id, cantidad_anexos) VALUES ($ruc_id, $cantidad_anexos);"
        local establecimientos_id=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT id FROM ruc_establecimientos_anexos WHERE ruc_id = $ruc_id ORDER BY id DESC LIMIT 1;" | xargs)
        
        # Detalle de establecimientos
        jq -c '.establecimientos_anexos.establecimientos[]? // empty' "$file" | while read -r establecimiento; do
            local codigo=$(echo "$establecimiento" | jq -r '.codigo // ""')
            local tipo_establecimiento=$(echo "$establecimiento" | jq -r '.tipo_establecimiento // ""')
            local direccion=$(echo "$establecimiento" | jq -r '.direccion // ""')
            local actividad_economica=$(echo "$establecimiento" | jq -r '.actividad_economica // ""')
            execute_sql "INSERT INTO ruc_establecimientos (establecimientos_anexos_id, codigo, tipo_establecimiento, direccion, actividad_economica) VALUES ($establecimientos_id, $(clean_string "$codigo"), $(clean_string "$tipo_establecimiento"), $(clean_string "$direccion"), $(clean_string "$actividad_economica"));"
        done
    fi
    
    echo "✓ Completado procesamiento de RUC: $ruc"
}

# Función principal
main() {
    echo "============================================"
    echo "INICIANDO IMPORTACIÓN DE DATOS JSON A POSTGRESQL"
    echo "============================================"
    
    # Verificar que jq está instalado
    if ! command -v jq &> /dev/null; then
        echo "Error: jq no está instalado. Instálalo con: sudo apt-get install jq"
        exit 1
    fi
    
    # Verificar que psql está instalado
    if ! command -v psql &> /dev/null; then
        echo "Error: psql no está instalado. Instala postgresql-client"
        exit 1
    fi
    
    # Verificar conexión a la base de datos
    if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
        echo "Error: No se puede conectar a la base de datos"
        echo "Verifica las credenciales y que PostgreSQL esté ejecutándose"
        exit 1
    fi
    
    echo "✓ Conexión a base de datos establecida"
    
    # Verificar que el directorio JSON existe
    if [ ! -d "$JSON_DIR" ]; then
        echo "Error: El directorio $JSON_DIR no existe"
        exit 1
    fi
    
    # Contar archivos JSON
    local total_files=$(find "$JSON_DIR" -name "*.json" -type f | wc -l)
    
    if [ "$total_files" -eq 0 ]; then
        echo "Error: No se encontraron archivos JSON en $JSON_DIR"
        exit 1
    fi
    
    echo "✓ Encontrados $total_files archivos JSON para procesar"
    
    # Procesar cada archivo JSON
    local processed=0
    local errors=0
    
    find "$JSON_DIR" -name "*.json" -type f | while read -r json_file; do
        if process_json_file "$json_file"; then
            ((processed++))
        else
            ((errors++))
            echo "✗ Error procesando: $json_file"
        fi
    done
    
    echo "============================================"
    echo "IMPORTACIÓN COMPLETADA"
    echo "Archivos procesados: $processed"
    echo "Errores: $errors"
    echo "============================================"
    
    # Mostrar estadísticas finales
    echo "ESTADÍSTICAS FINALES:"
    local total_rucs=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM ruc_informacion_basica;" | xargs)
    echo "- Total RUCs en base de datos: $total_rucs"
    
    local rucs_con_deuda=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM ruc_deuda_coactiva WHERE total_deuda > 0;" | xargs)
    echo "- RUCs con deuda coactiva: $rucs_con_deuda"
    
    local rucs_con_trabajadores=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(DISTINCT ruc_id) FROM ruc_cantidad_trabajadores;" | xargs)
    echo "- RUCs con información de trabajadores: $rucs_con_trabajadores"
    
    local rucs_con_representantes=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(DISTINCT ruc_id) FROM ruc_representantes_legales;" | xargs)
    echo "- RUCs con representantes legales: $rucs_con_representantes"
    
    echo "============================================"
}

# Ejecutar función principal
main "$@"