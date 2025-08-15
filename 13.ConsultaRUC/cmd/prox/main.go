package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Proxy struct {
	IP           string    `json:"ip"`
	Port         string    `json:"port"`
	Country      string    `json:"country"`
	Anonymity    string    `json:"anonymity"`
	HTTPS        string    `json:"https"`
	Status       string    `json:"status"`
	Error        string    `json:"error,omitempty"`
	TestTime     time.Time `json:"test_time"`
	ResponseTime int64     `json:"response_time_ms"`
}

type TestResult struct {
	Proxy    Proxy
	Working  bool
	Error    string
	Duration int64
}

func main() {
	fmt.Println("🚀 Iniciando scraper de proxies...")
	startTime := time.Now()

	// Scraper los proxies
	proxies, err := scrapeProxies("https://free-proxy-list.net")
	if err != nil {
		log.Fatalf("Error al hacer scraping: %v", err)
	}

	fmt.Printf("📊 Se encontraron %d proxies\n", len(proxies))

	// Verificar proxies concurrentemente con registro detallado
	fmt.Println("🔍 Verificando proxies...")
	allResults, workingProxies := verifyProxiesParallelWithLog(proxies, 30) // Reducido a 30 workers

	fmt.Printf("\n📈 RESUMEN DE VERIFICACIÓN:\n")
	fmt.Printf("✅ Proxies funcionando: %d\n", len(workingProxies))
	fmt.Printf("❌ Proxies fallidos: %d\n", len(allResults)-len(workingProxies))
	fmt.Printf("⏱️  Tiempo total: %v\n", time.Since(startTime))

	// Guardar proxies funcionando
	if err := saveProxiesToFile(workingProxies, "proxies.txt"); err != nil {
		log.Fatalf("Error al guardar proxies: %v", err)
	}

	// Guardar registro completo
	if err := saveCompleteLog(allResults, "proxy_verification_log.txt"); err != nil {
		log.Fatalf("Error al guardar log: %v", err)
	}

	// Guardar reporte JSON
	if err := saveJSONReport(allResults, "proxy_report.json"); err != nil {
		log.Fatalf("Error al guardar reporte JSON: %v", err)
	}

	fmt.Println("\n💾 Archivos generados:")
	fmt.Println("   📄 proxies.txt - Solo proxies funcionando")
	fmt.Println("   📋 proxy_verification_log.txt - Log detallado de todas las pruebas")
	fmt.Println("   📊 proxy_report.json - Reporte completo en JSON")
}

func scrapeProxies(baseURL string) ([]Proxy, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error al realizar petición GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("código de estado: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error al parsear HTML: %v", err)
	}

	var proxies []Proxy

	// Buscar la tabla de proxies
	doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
		cells := s.Find("td")
		if cells.Length() >= 7 {
			ip := strings.TrimSpace(cells.Eq(0).Text())
			port := strings.TrimSpace(cells.Eq(1).Text())
			country := strings.TrimSpace(cells.Eq(3).Text())
			anonymity := strings.TrimSpace(cells.Eq(4).Text())
			https := strings.TrimSpace(cells.Eq(6).Text())

			// Validar que IP y puerto no estén vacíos
			if ip != "" && port != "" && isValidIP(ip) {
				proxy := Proxy{
					IP:        ip,
					Port:      port,
					Country:   country,
					Anonymity: anonymity,
					HTTPS:     https,
					Status:    "pending",
					TestTime:  time.Now(),
				}
				proxies = append(proxies, proxy)
			}
		}
	})

	return proxies, nil
}

func isValidIP(ip string) bool {
	// Expresión regular simple para validar IP
	ipRegex := regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$`)
	return ipRegex.MatchString(ip) && ip != "0.0.0.0"
}

func verifyProxiesParallelWithLog(proxies []Proxy, maxWorkers int) ([]Proxy, []Proxy) {
	jobs := make(chan Proxy, len(proxies))
	results := make(chan TestResult, len(proxies))
	var wg sync.WaitGroup

	// Iniciar workers
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for proxy := range jobs {
				result := verifyProxyAdvanced(proxy)
				results <- result

				// Log en tiempo real
				status := "❌"
				if result.Working {
					status = "✅"
				}
				fmt.Printf("%s [Worker %d] %s:%s (%s) - %dms - %s\n",
					status, workerID, proxy.IP, proxy.Port, proxy.Country,
					result.Duration, result.Error)
			}
		}(w)
	}

	// Enviar trabajos
	go func() {
		for _, proxy := range proxies {
			jobs <- proxy
		}
		close(jobs)
	}()

	// Esperar a que terminen todos los workers
	go func() {
		wg.Wait()
		close(results)
	}()

	// Recopilar resultados
	var allResults []Proxy
	var workingProxies []Proxy

	for result := range results {
		// Actualizar proxy con resultado
		result.Proxy.Status = "failed"
		result.Proxy.Error = result.Error
		result.Proxy.ResponseTime = result.Duration
		result.Proxy.TestTime = time.Now()

		if result.Working {
			result.Proxy.Status = "working"
			result.Proxy.Error = ""
			workingProxies = append(workingProxies, result.Proxy)
		}

		allResults = append(allResults, result.Proxy)
	}

	return allResults, workingProxies
}

func verifyProxyAdvanced(proxy Proxy) TestResult {
	startTime := time.Now()
	proxyURL := fmt.Sprintf("http://%s:%s", proxy.IP, proxy.Port)

	// Crear un cliente HTTP con el proxy configurado
	proxyURLParsed, err := url.Parse(proxyURL)
	if err != nil {
		return TestResult{
			Proxy:    proxy,
			Working:  false,
			Error:    fmt.Sprintf("Invalid proxy URL: %v", err),
			Duration: time.Since(startTime).Milliseconds(),
		}
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURLParsed),
		},
		Timeout: 15 * time.Second, // Timeout aumentado
	}

	// Múltiples URLs de prueba con diferentes propósitos
	testURLs := []struct {
		URL    string
		Name   string
		Expect string
	}{
		{"http://httpbin.org/ip", "HttpBin IP", "origin"},
		{"http://icanhazip.com", "ICanHazIP", ""},
		{"http://ipinfo.io/ip", "IPInfo", ""},
		{"http://checkip.amazonaws.com", "AWS CheckIP", ""},
		{"http://httpbin.org/get", "HttpBin GET", "args"},
	}

	var lastError string
	successCount := 0

	for _, test := range testURLs {
		resp, err := client.Get(test.URL)
		if err != nil {
			lastError = fmt.Sprintf("%s failed: %v", test.Name, err)
			continue
		}

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err == nil && len(body) > 0 {
				// Verificación adicional del contenido si se especifica
				if test.Expect == "" || strings.Contains(string(body), test.Expect) {
					successCount++
					if successCount >= 2 { // Al menos 2 pruebas exitosas
						return TestResult{
							Proxy:    proxy,
							Working:  true,
							Error:    "OK",
							Duration: time.Since(startTime).Milliseconds(),
						}
					}
				}
			}
		} else {
			lastError = fmt.Sprintf("%s returned status %d", test.Name, resp.StatusCode)
		}
		resp.Body.Close()
	}

	if lastError == "" {
		lastError = "All tests failed - no valid response"
	}

	return TestResult{
		Proxy:    proxy,
		Working:  false,
		Error:    lastError,
		Duration: time.Since(startTime).Milliseconds(),
	}
}

func saveProxiesToFile(proxies []Proxy, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error al crear archivo: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Escribir encabezado
	header := fmt.Sprintf("# Lista de Proxies Verificados - Generada: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	header += "# ⚠️  NOTA: Verificación no es 100% infalible. Proxies pueden fallar posteriormente.\n"
	header += "# Total de proxies funcionando: " + fmt.Sprintf("%d\n", len(proxies))
	header += strings.Repeat("-", 70) + "\n\n"

	if _, err := writer.WriteString(header); err != nil {
		return err
	}

	// Formato simple para uso directo
	writer.WriteString("# FORMATO SIMPLE (IP:Puerto)\n")
	for _, proxy := range proxies {
		line := fmt.Sprintf("%s:%s\n", proxy.IP, proxy.Port)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	writer.WriteString("\n# FORMATO DETALLADO\n")
	for _, proxy := range proxies {
		line := fmt.Sprintf("%s:%s | %s | %s | HTTPS: %s | Respuesta: %dms\n",
			proxy.IP, proxy.Port, proxy.Country, proxy.Anonymity, proxy.HTTPS, proxy.ResponseTime)
		if _, err := writer.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

func saveCompleteLog(allResults []Proxy, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error al crear log: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Encabezado del log
	header := fmt.Sprintf("REGISTRO COMPLETO DE VERIFICACIÓN DE PROXIES\n")
	header += fmt.Sprintf("Generado: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	header += strings.Repeat("=", 80) + "\n\n"

	writer.WriteString(header)

	// Estadísticas
	working := 0
	failed := 0
	for _, proxy := range allResults {
		if proxy.Status == "working" {
			working++
		} else {
			failed++
		}
	}

	stats := fmt.Sprintf("ESTADÍSTICAS:\n")
	stats += fmt.Sprintf("✅ Proxies funcionando: %d (%.1f%%)\n", working, float64(working)/float64(len(allResults))*100)
	stats += fmt.Sprintf("❌ Proxies fallidos: %d (%.1f%%)\n", failed, float64(failed)/float64(len(allResults))*100)
	stats += fmt.Sprintf("📊 Total procesados: %d\n\n", len(allResults))

	writer.WriteString(stats)

	// Registro detallado
	writer.WriteString("REGISTRO DETALLADO POR ORDEN DE VERIFICACIÓN:\n")
	writer.WriteString(strings.Repeat("-", 80) + "\n")

	for i, proxy := range allResults {
		status := "❌ FALLÓ"
		if proxy.Status == "working" {
			status = "✅ FUNCIONA"
		}

		logEntry := fmt.Sprintf("[%03d] %s %s:%s\n", i+1, status, proxy.IP, proxy.Port)
		logEntry += fmt.Sprintf("      País: %s | Anonimato: %s | HTTPS: %s\n", proxy.Country, proxy.Anonymity, proxy.HTTPS)
		logEntry += fmt.Sprintf("      Tiempo respuesta: %dms | Verificado: %s\n", proxy.ResponseTime, proxy.TestTime.Format("15:04:05"))

		if proxy.Error != "" && proxy.Error != "OK" {
			logEntry += fmt.Sprintf("      Error: %s\n", proxy.Error)
		}

		logEntry += "\n"

		if _, err := writer.WriteString(logEntry); err != nil {
			return err
		}
	}

	return nil
}

func saveJSONReport(allResults []Proxy, filename string) error {
	report := struct {
		GeneratedAt    time.Time `json:"generated_at"`
		TotalProxies   int       `json:"total_proxies"`
		WorkingProxies int       `json:"working_proxies"`
		FailedProxies  int       `json:"failed_proxies"`
		SuccessRate    float64   `json:"success_rate"`
		Results        []Proxy   `json:"results"`
	}{
		GeneratedAt:  time.Now(),
		TotalProxies: len(allResults),
		Results:      allResults,
	}

	// Calcular estadísticas
	for _, proxy := range allResults {
		if proxy.Status == "working" {
			report.WorkingProxies++
		} else {
			report.FailedProxies++
		}
	}

	if report.TotalProxies > 0 {
		report.SuccessRate = float64(report.WorkingProxies) / float64(report.TotalProxies) * 100
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error al crear reporte JSON: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	return encoder.Encode(report)
}
