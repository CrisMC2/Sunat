package utils

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// GetRandomProxyFromFile obtiene un proxy aleatorio del archivo
func GetRandomProxyFromFile(filename string) (string, error) {
	proxies, err := ReadProxiesFromFile(filename)
	if err != nil {
		return "", err
	}

	if len(proxies) == 0 {
		return "", fmt.Errorf("no hay proxies disponibles")
	}

	// REEMPLAZAR por:
	randomProxy := proxies[rand.IntN(len(proxies))]

	// Asegurar que tenga el esquema http://
	if !strings.HasPrefix(randomProxy, "http://") && !strings.HasPrefix(randomProxy, "https://") {
		randomProxy = "http://" + randomProxy
	}

	return randomProxy, nil
}

// ReadProxiesFromFile lee proxies del archivo y filtra los válidos
func ReadProxiesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error abriendo archivo de proxies: %v", err)
	}
	defer file.Close()

	var validProxies []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Saltar líneas vacías y comentarios
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// ✅ VALIDACIÓN MÁS PERMISIVA
		if IsValidProxyFormat(line) {
			validProxies = append(validProxies, line)
		} else {
			fmt.Printf("⚠️ Línea %d: formato de proxy inválido: %s\n", lineNum, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error leyendo archivo: %v", err)
	}

	if len(validProxies) == 0 {
		return nil, fmt.Errorf("no hay proxies válidos en el archivo %s", filename)
	}

	return validProxies, nil
}

// IsValidRUC valida que el RUC tenga el formato correcto
func IsValidRUC(ruc string) bool {
	if len(ruc) != 11 {
		return false
	}
	// Verificar que solo contenga dígitos
	for _, char := range ruc {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// IsValidProxyFormat valida que el proxy tenga formato IP:PORT
func IsValidProxyFormat(proxy string) bool {
	// Remover espacios en blanco
	proxy = strings.TrimSpace(proxy)

	if proxy == "" {
		return false
	}

	// Si tiene http:// o https://, validar como URL completa
	if strings.HasPrefix(proxy, "http://") || strings.HasPrefix(proxy, "https://") {
		_, err := url.Parse(proxy)
		return err == nil
	}

	// Si no tiene esquema, validar como IP:PORT o DOMAIN:PORT
	parts := strings.Split(proxy, ":")
	if len(parts) != 2 {
		return false
	}

	host := parts[0]
	port := parts[1]

	// Validar que el puerto sea numérico
	portRegex := regexp.MustCompile(`^\d+$`)
	if !portRegex.MatchString(port) {
		return false
	}

	// Validar que el host no esté vacío
	if host == "" {
		return false
	}

	// Validar IP (opcional, también acepta dominios)
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return ipRegex.MatchString(host) || domainRegex.MatchString(host) || host == "localhost"
}

// GenerateSecureRandomInt genera un número aleatorio seguro
func GenerateSecureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, fmt.Errorf("max debe ser positivo")
	}
	n := rand.IntN(max)
	return n, nil
}

// GenerateRandomInt genera un número aleatorio simple (fallback)
func GenerateRandomInt(max int) int {
	if max <= 0 {
		return 0
	}
	// Usar time como seed más el PID para mejor distribución
	seed := time.Now().UnixNano() + int64(os.Getpid())
	return int(seed % int64(max))
}

// GenerateRandomFloat genera un float aleatorio entre 0 y 1
func GenerateRandomFloat() float64 {
	seed := time.Now().UnixNano()
	return float64(seed%1000) / 1000.0
}

// ValidateEnvironment verifica que las dependencias estén disponibles
func ValidateEnvironment() error {
	// Verificar que existe el archivo de proxies
	if _, err := os.Stat("proxies.txt"); os.IsNotExist(err) {
		return fmt.Errorf("archivo proxies.txt no encontrado")
	}

	return nil
}
