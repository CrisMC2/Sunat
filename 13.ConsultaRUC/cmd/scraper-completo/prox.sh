#!/bin/bash

# Script para scrapear y verificar proxies de free-proxy-list.net
# Autor: Script generado para scraping de proxies
# Fecha: $(date)

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Archivos
TEMP_HTML="temp_proxy_list.html"
RAW_PROXIES="raw_proxies.txt"
VERIFIED_PROXIES="proxies.txt"
LOG_FILE="proxy_checker.log"

# URLs
PROXY_URL="https://free-proxy-list.net/"
TEST_URL="http://httpbin.org/ip"

# Configuración
MAX_CONCURRENT=20
TIMEOUT=10

echo -e "${BLUE}=== Proxy Scraper & Verificador ===${NC}"
echo -e "${YELLOW}Descargando lista de proxies...${NC}"

# Función para logging
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" >> "$LOG_FILE"
}

# Limpiar archivos temporales anteriores
cleanup() {
    rm -f "$TEMP_HTML" "$RAW_PROXIES"
}

# Descargar la página
download_page() {
    if curl -s -A "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36" \
        -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8" \
        -H "Accept-Language: en-US,en;q=0.5" \
        -H "Accept-Encoding: gzip, deflate" \
        -H "Connection: keep-alive" \
        --compressed \
        "$PROXY_URL" -o "$TEMP_HTML"; then
        echo -e "${GREEN}✓ Página descargada correctamente${NC}"
        log "Página descargada exitosamente"
        return 0
    else
        echo -e "${RED}✗ Error al descargar la página${NC}"
        log "Error al descargar la página"
        return 1
    fi
}

# Extraer proxies del HTML
extract_proxies() {
    echo -e "${YELLOW}Extrayendo proxies del HTML...${NC}"
    
    # Usar grep y sed para extraer IPs y puertos de las filas de la tabla
    # Buscar patrones como <td>IP</td><td>PORT</td>
    grep -oP '<tr><td>\K[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}</td><td>[0-9]+' "$TEMP_HTML" | \
    sed 's|</td><td>|:|g' > "$RAW_PROXIES"
    
    # Método alternativo usando awk para mayor precisión
    awk '
    /<tr><td>[0-9]/ {
        # Extraer IP
        match($0, /<td>([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})<\/td>/, ip)
        # Buscar el puerto en la siguiente celda
        match($0, /<td>[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}<\/td><td>([0-9]+)<\/td>/, port)
        if (ip[1] && port[1]) {
            print ip[1] ":" port[1]
        }
    }
    ' "$TEMP_HTML" > "$RAW_PROXIES"
    
    # Si el método anterior no funciona, usar un enfoque más directo
    if [ ! -s "$RAW_PROXIES" ]; then
        echo -e "${YELLOW}Usando método alternativo de extracción...${NC}"
        
        # Extraer todas las filas de tabla y procesar
        grep -A1 -B1 '<td>[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}</td>' "$TEMP_HTML" | \
        grep -oP '<td>[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}</td><td>[0-9]+</td>' | \
        sed -E 's/<td>([0-9.]+)<\/td><td>([0-9]+)<\/td>/\1:\2/' > "$RAW_PROXIES"
    fi
    
    local count=$(wc -l < "$RAW_PROXIES" 2>/dev/null || echo "0")
    echo -e "${GREEN}✓ Extraídos $count proxies${NC}"
    log "Extraídos $count proxies"
    
    if [ "$count" -eq 0 ]; then
        echo -e "${RED}✗ No se pudieron extraer proxies. Verificando formato HTML...${NC}"
        echo -e "${YELLOW}Mostrando primeras líneas del HTML:${NC}"
        head -20 "$TEMP_HTML"
        return 1
    fi
    
    return 0
}

# Verificar un proxy individual
check_proxy() {
    local proxy="$1"
    local ip=$(echo "$proxy" | cut -d: -f1)
    local port=$(echo "$proxy" | cut -d: -f2)
    
    # Verificar formato básico
    if [[ ! "$ip" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] || [[ ! "$port" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    
    # Probar conectividad básica
    if timeout "$TIMEOUT" nc -z "$ip" "$port" 2>/dev/null; then
        # Probar como proxy HTTP
        local response=$(timeout "$TIMEOUT" curl -s --proxy "$proxy" --max-time "$TIMEOUT" "$TEST_URL" 2>/dev/null)
        if [[ "$response" == *"origin"* ]] && [[ "$response" == *"$ip"* || "$response" != *"$(curl -s --max-time 5 "$TEST_URL" 2>/dev/null | grep -o '"origin": "[^"]*"' | cut -d'"' -f4)"* ]]; then
            return 0
        fi
        
        # Probar conectividad simple si falla el test HTTP
        if timeout 3 bash -c "echo >/dev/tcp/$ip/$port" 2>/dev/null; then
            return 0
        fi
    fi
    
    return 1
}

# Verificar proxies en paralelo
verify_proxies() {
    echo -e "${YELLOW}Verificando proxies (esto puede tomar varios minutos)...${NC}"
    
    # Limpiar archivo de salida
    > "$VERIFIED_PROXIES"
    
    local total=$(wc -l < "$RAW_PROXIES")
    local current=0
    local working=0
    
    # Crear función para verificación en background
    verify_batch() {
        local proxy="$1"
        if check_proxy "$proxy"; then
            echo "$proxy" >> "$VERIFIED_PROXIES"
            echo -e "${GREEN}✓ $proxy${NC}"
            log "Proxy funcionando: $proxy"
            return 0
        else
            echo -e "${RED}✗ $proxy${NC}"
            log "Proxy no funciona: $proxy"
            return 1
        fi
    }
    
    export -f check_proxy verify_batch log
    export TIMEOUT TEST_URL VERIFIED_PROXIES LOG_FILE RED GREEN NC
    
    # Procesar en lotes para controlar concurrencia
    echo -e "${BLUE}Procesando $total proxies con máximo $MAX_CONCURRENT conexiones simultáneas...${NC}"
    
    # Usar xargs para paralelización controlada
    cat "$RAW_PROXIES" | head -100 | xargs -I {} -P "$MAX_CONCURRENT" bash -c 'verify_batch "$@"' _ {}
    
    working=$(wc -l < "$VERIFIED_PROXIES" 2>/dev/null || echo "0")
    echo -e "${BLUE}=== Resultados ===${NC}"
    echo -e "${GREEN}Proxies funcionando: $working${NC}"
    echo -e "${YELLOW}Total verificados: $total${NC}"
    echo -e "${BLUE}Archivo de salida: $VERIFIED_PROXIES${NC}"
    
    log "Verificación completada: $working/$total proxies funcionando"
}

# Función principal
main() {
    echo -e "${BLUE}Iniciando proceso de scraping y verificación...${NC}"
    log "Iniciando script de proxy scraper"
    
    # Limpiar archivos anteriores
    cleanup
    
    # Descargar página
    if ! download_page; then
        echo -e "${RED}Error: No se pudo descargar la página${NC}"
        exit 1
    fi
    
    # Extraer proxies
    if ! extract_proxies; then
        echo -e "${RED}Error: No se pudieron extraer proxies${NC}"
        cleanup
        exit 1
    fi
    
    # Mostrar muestra de proxies extraídos
    echo -e "${BLUE}Muestra de proxies extraídos:${NC}"
    head -5 "$RAW_PROXIES"
    echo "..."
    
    # Preguntar si continuar con verificación
    read -p "¿Continuar con la verificación? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        verify_proxies
    else
        echo -e "${YELLOW}Proxies extraídos guardados en $RAW_PROXIES${NC}"
        echo -e "${YELLOW}Puedes verificarlos más tarde ejecutando la función verify_proxies${NC}"
    fi
    
    # Limpiar archivos temporales
    cleanup
    
    echo -e "${GREEN}¡Proceso completado!${NC}"
    if [ -f "$VERIFIED_PROXIES" ] && [ -s "$VERIFIED_PROXIES" ]; then
        echo -e "${GREEN}Proxies verificados guardados en: $VERIFIED_PROXIES${NC}"
        echo -e "${BLUE}Primeros 5 proxies funcionando:${NC}"
        head -5 "$VERIFIED_PROXIES"
    fi
}

# Verificar dependencias
check_dependencies() {
    local deps=("curl" "grep" "sed" "awk" "nc" "timeout")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            echo -e "${RED}Error: $dep no está instalado${NC}"
            echo "Instala las dependencias necesarias:"
            echo "Ubuntu/Debian: sudo apt install curl grep sed gawk netcat-openbsd coreutils"
            echo "CentOS/RHEL: sudo yum install curl grep sed gawk nmap-ncat coreutils"
            exit 1
        fi
    done
}

# Verificar dependencias antes de ejecutar
check_dependencies

# Ejecutar función principal
main "$@"