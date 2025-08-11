package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

type ResultadoRUC struct {
	RUC               string    `json:"ruc"`
	TipoContribuyente string    `json:"tipo_contribuyente,omitempty"`
	RazonSocial       string    `json:"razon_social,omitempty"`
	Estado            string    `json:"estado,omitempty"`
	Condicion         string    `json:"condicion,omitempty"`
	Exitoso           bool      `json:"exitoso"`
	Error             string    `json:"error,omitempty"`
	TiempoProceso     float64   `json:"tiempo_proceso_segundos"`
}

func main() {
	// RUCs de prueba
	rucs := []string{
		// Personas Jurídicas (20xxx)
		"20606454466", "20393261162", "20393758008", "20393317991",
		"20600656288", "20602770568", "20605437070", "20604817103",
		"20606695439", "20554129286", "20606642131", "20613467999",
		// Personas Naturales con Negocio (10xxx)
		"10719706288", "10775397131", "10420242986", "10467806527",
		"10477649845", "10758787651", "10763669322", "10776888813",
		"10712337601", "10738074292", "10758153911", "10470593291",
		"10729620730", "10725461327", "10724702517",
	}

	fmt.Println("=== TEST DE SCRAPER SUNAT - VERSIÓN ROBUSTA ===")
	fmt.Printf("Total de RUCs a procesar: %d\n", len(rucs))
	fmt.Printf("- Personas Jurídicas (20xxx): %d\n", contarPorPrefijo(rucs, "20"))
	fmt.Printf("- Personas Naturales (10xxx): %d\n\n", contarPorPrefijo(rucs, "10"))

	resultados := []ResultadoRUC{}
	exitosos := 0
	fallidos := 0

	// Procesar por lotes para evitar problemas de memoria
	batchSize := 5
	for i := 0; i < len(rucs); i += batchSize {
		end := i + batchSize
		if end > len(rucs) {
			end = len(rucs)
		}
		
		batch := rucs[i:end]
		fmt.Printf("\n--- Procesando lote %d-%d ---\n", i+1, end)
		
		// Crear nuevo scraper para cada lote
		sunatScraper, err := scraper.NewSUNATScraper()
		if err != nil {
			log.Printf("Error creando scraper para lote: %v", err)
			continue
		}
		
		// Procesar RUCs del lote
		for j, ruc := range batch {
			fmt.Printf("[%d/%d] RUC %s: ", i+j+1, len(rucs), ruc)
			
			resultado := procesarRUC(sunatScraper, ruc)
			resultados = append(resultados, resultado)
			
			if resultado.Exitoso {
				exitosos++
				fmt.Printf("✅ %s\n", resultado.RazonSocial)
			} else {
				fallidos++
				fmt.Printf("❌ Error: %s\n", resultado.Error)
			}
			
			// Pausa entre consultas
			if j < len(batch)-1 {
				time.Sleep(1 * time.Second)
			}
		}
		
		// Cerrar scraper del lote
		sunatScraper.Close()
		
		// Pausa entre lotes
		if end < len(rucs) {
			fmt.Println("Pausa entre lotes...")
			time.Sleep(3 * time.Second)
		}
	}

	// Generar reporte
	generarReporte(resultados, exitosos, fallidos)
}

func procesarRUC(scraper *scraper.SUNATScraper, ruc string) ResultadoRUC {
	inicio := time.Now()
	resultado := ResultadoRUC{
		RUC: ruc,
	}
	
	// Usar recover para capturar panics
	defer func() {
		if r := recover(); r != nil {
			resultado.Exitoso = false
			resultado.Error = fmt.Sprintf("panic: %v", r)
		}
		resultado.TiempoProceso = time.Since(inicio).Seconds()
	}()
	
	// Intentar scraping
	info, err := scraper.ScrapeRUC(ruc)
	if err != nil {
		resultado.Exitoso = false
		resultado.Error = err.Error()
		return resultado
	}
	
	// Extraer información
	resultado.Exitoso = true
	resultado.RazonSocial = info.RazonSocial
	resultado.TipoContribuyente = info.TipoContribuyente
	resultado.Estado = info.Estado
	resultado.Condicion = info.Condicion
	
	// Guardar JSON individual
	os.MkdirAll("resultados", 0755)
	jsonData, _ := json.MarshalIndent(info, "", "  ")
	fileName := fmt.Sprintf("resultados/ruc_%s.json", ruc)
	os.WriteFile(fileName, jsonData, 0644)
	
	return resultado
}

func contarPorPrefijo(rucs []string, prefijo string) int {
	count := 0
	for _, ruc := range rucs {
		if strings.HasPrefix(ruc, prefijo) {
			count++
		}
	}
	return count
}

func generarReporte(resultados []ResultadoRUC, exitosos, fallidos int) {
	fmt.Println("\n=== RESUMEN FINAL ===")
	fmt.Printf("Total procesados: %d\n", len(resultados))
	fmt.Printf("Exitosos: %d (%.1f%%)\n", exitosos, float64(exitosos)/float64(len(resultados))*100)
	fmt.Printf("Fallidos: %d (%.1f%%)\n", fallidos, float64(fallidos)/float64(len(resultados))*100)
	
	// Análisis por tipo
	fmt.Println("\n=== ANÁLISIS POR TIPO ===")
	
	// Personas Jurídicas
	juridicas := filtrarPorPrefijo(resultados, "20")
	juridicasOK := contarExitosos(juridicas)
	fmt.Printf("\nPersonas Jurídicas (20xxx):\n")
	fmt.Printf("- Total: %d\n", len(juridicas))
	fmt.Printf("- Exitosos: %d (%.1f%%)\n", juridicasOK, float64(juridicasOK)/float64(len(juridicas))*100)
	
	// Estados más comunes
	estados := make(map[string]int)
	for _, r := range juridicas {
		if r.Exitoso && r.Estado != "" {
			estados[r.Estado]++
		}
	}
	fmt.Println("- Estados:")
	for estado, count := range estados {
		fmt.Printf("  * %s: %d\n", estado, count)
	}
	
	// Personas Naturales
	naturales := filtrarPorPrefijo(resultados, "10")
	naturalesOK := contarExitosos(naturales)
	fmt.Printf("\nPersonas Naturales con Negocio (10xxx):\n")
	fmt.Printf("- Total: %d\n", len(naturales))
	fmt.Printf("- Exitosos: %d (%.1f%%)\n", naturalesOK, float64(naturalesOK)/float64(len(naturales))*100)
	
	// Estados más comunes
	estadosNat := make(map[string]int)
	for _, r := range naturales {
		if r.Exitoso && r.Estado != "" {
			estadosNat[r.Estado]++
		}
	}
	fmt.Println("- Estados:")
	for estado, count := range estadosNat {
		fmt.Printf("  * %s: %d\n", estado, count)
	}
	
	// Guardar reporte JSON
	reporte := map[string]interface{}{
		"fecha":            time.Now().Format("2006-01-02 15:04:05"),
		"total_procesados": len(resultados),
		"exitosos":         exitosos,
		"fallidos":         fallidos,
		"tasa_exito":       fmt.Sprintf("%.1f%%", float64(exitosos)/float64(len(resultados))*100),
		"tiempo_promedio":  calcularTiempoPromedio(resultados),
		"resultados":       resultados,
	}
	
	reporteJSON, _ := json.MarshalIndent(reporte, "", "  ")
	reporteFile := fmt.Sprintf("reporte_%s.json", time.Now().Format("20060102_150405"))
	os.WriteFile(reporteFile, reporteJSON, 0644)
	fmt.Printf("\n✅ Reporte guardado en: %s\n", reporteFile)
	
	// Generar CSV
	generarCSV(resultados)
}

func filtrarPorPrefijo(resultados []ResultadoRUC, prefijo string) []ResultadoRUC {
	var filtrados []ResultadoRUC
	for _, r := range resultados {
		if strings.HasPrefix(r.RUC, prefijo) {
			filtrados = append(filtrados, r)
		}
	}
	return filtrados
}

func contarExitosos(resultados []ResultadoRUC) int {
	count := 0
	for _, r := range resultados {
		if r.Exitoso {
			count++
		}
	}
	return count
}

func calcularTiempoPromedio(resultados []ResultadoRUC) float64 {
	var total float64
	for _, r := range resultados {
		total += r.TiempoProceso
	}
	return total / float64(len(resultados))
}

func generarCSV(resultados []ResultadoRUC) {
	csvFile := fmt.Sprintf("resultados_%s.csv", time.Now().Format("20060102_150405"))
	file, err := os.Create(csvFile)
	if err != nil {
		return
	}
	defer file.Close()
	
	// Encabezados
	file.WriteString("RUC,Tipo,Razon Social,Tipo Contribuyente,Estado,Condicion,Exitoso,Error,Tiempo (seg)\n")
	
	// Datos
	for _, r := range resultados {
		tipo := "Natural"
		if strings.HasPrefix(r.RUC, "20") {
			tipo = "Juridica"
		}
		
		line := fmt.Sprintf("%s,%s,\"%s\",\"%s\",%s,%s,%t,%s,%.2f\n",
			r.RUC,
			tipo,
			strings.ReplaceAll(r.RazonSocial, "\"", "'"),
			strings.ReplaceAll(r.TipoContribuyente, "\"", "'"),
			r.Estado,
			r.Condicion,
			r.Exitoso,
			r.Error,
			r.TiempoProceso,
		)
		file.WriteString(line)
	}
	
	fmt.Printf("✅ CSV guardado en: %s\n", csvFile)
}