package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

type ResultadoProcesamiento struct {
	RUC              string    `json:"ruc"`
	TipoContribuyente string   `json:"tipo_contribuyente"`
	RazonSocial      string    `json:"razon_social"`
	Estado           string    `json:"estado"`
	Condicion        string    `json:"condicion"`
	Exitoso          bool      `json:"exitoso"`
	Error            string    `json:"error,omitempty"`
	TiempoProceso    string    `json:"tiempo_proceso"`
	FechaProceso     time.Time `json:"fecha_proceso"`
}

func main() {
	// Leer archivo de RUCs
	rucsFile := "rucs_test.txt"
	if len(os.Args) > 1 {
		rucsFile = os.Args[1]
	}

	rucs, err := leerRUCs(rucsFile)
	if err != nil {
		log.Fatal("Error leyendo archivo de RUCs:", err)
	}

	fmt.Printf("=== PRUEBA MASIVA DE SCRAPER SUNAT ===\n")
	fmt.Printf("Total de RUCs a procesar: %d\n", len(rucs))
	fmt.Printf("- Personas Jurídicas (20xxx): %d\n", contarPorPrefijo(rucs, "20"))
	fmt.Printf("- Personas Naturales (10xxx): %d\n\n", contarPorPrefijo(rucs, "10"))

	// Crear scraper
	sunatScraper, err := scraper.NewSUNATScraper()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer sunatScraper.Close()

	// Procesar RUCs
	resultados := []ResultadoProcesamiento{}
	exitosos := 0
	fallidos := 0

	for i, ruc := range rucs {
		fmt.Printf("[%d/%d] Procesando RUC: %s... ", i+1, len(rucs), ruc)
		
		inicio := time.Now()
		resultado := ResultadoProcesamiento{
			RUC:          ruc,
			FechaProceso: time.Now(),
		}

		// Intentar scraping
		info, err := sunatScraper.ScrapeRUC(ruc)
		
		if err != nil {
			resultado.Exitoso = false
			resultado.Error = err.Error()
			fallidos++
			fmt.Printf("❌ ERROR: %s\n", err.Error())
		} else {
			resultado.Exitoso = true
			resultado.TipoContribuyente = info.TipoContribuyente
			resultado.RazonSocial = info.RazonSocial
			resultado.Estado = info.Estado
			resultado.Condicion = info.Condicion
			exitosos++
			
			// Guardar JSON individual
			jsonData, _ := json.MarshalIndent(info, "", "  ")
			fileName := fmt.Sprintf("resultados/ruc_%s.json", ruc)
			os.MkdirAll("resultados", 0755)
			os.WriteFile(fileName, jsonData, 0644)
			
			fmt.Printf("✅ OK - %s (%s)\n", info.RazonSocial, info.Estado)
		}
		
		resultado.TiempoProceso = time.Since(inicio).String()
		resultados = append(resultados, resultado)
		
		// Pausa entre consultas para no saturar
		if i < len(rucs)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	// Generar reporte
	fmt.Printf("\n=== RESUMEN DE PROCESAMIENTO ===\n")
	fmt.Printf("Total procesados: %d\n", len(rucs))
	fmt.Printf("Exitosos: %d (%.1f%%)\n", exitosos, float64(exitosos)/float64(len(rucs))*100)
	fmt.Printf("Fallidos: %d (%.1f%%)\n", fallidos, float64(fallidos)/float64(len(rucs))*100)

	// Análisis por tipo
	fmt.Printf("\n=== ANÁLISIS POR TIPO ===\n")
	analizarPorTipo(resultados)

	// Guardar reporte completo
	reporte := map[string]interface{}{
		"fecha_proceso":    time.Now(),
		"total_procesados": len(rucs),
		"exitosos":         exitosos,
		"fallidos":         fallidos,
		"resultados":       resultados,
	}

	reporteJSON, _ := json.MarshalIndent(reporte, "", "  ")
	reporteFile := fmt.Sprintf("reporte_masivo_%s.json", time.Now().Format("20060102_150405"))
	os.WriteFile(reporteFile, reporteJSON, 0644)
	
	fmt.Printf("\n✅ Reporte guardado en: %s\n", reporteFile)

	// Generar CSV para análisis
	generarCSV(resultados)
}

func leerRUCs(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rucs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ruc := strings.TrimSpace(scanner.Text())
		if ruc != "" {
			rucs = append(rucs, ruc)
		}
	}

	return rucs, scanner.Err()
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

func analizarPorTipo(resultados []ResultadoProcesamiento) {
	// Análisis de personas jurídicas
	juridicas := filtrarPorPrefijo(resultados, "20")
	fmt.Printf("\nPersonas Jurídicas (20xxx):\n")
	fmt.Printf("- Total: %d\n", len(juridicas))
	fmt.Printf("- Exitosos: %d\n", contarExitosos(juridicas))
	fmt.Printf("- Estados:\n")
	contarPorCampo(juridicas, "Estado")
	
	// Análisis de personas naturales
	naturales := filtrarPorPrefijo(resultados, "10")
	fmt.Printf("\nPersonas Naturales con Negocio (10xxx):\n")
	fmt.Printf("- Total: %d\n", len(naturales))
	fmt.Printf("- Exitosos: %d\n", contarExitosos(naturales))
	fmt.Printf("- Estados:\n")
	contarPorCampo(naturales, "Estado")
}

func filtrarPorPrefijo(resultados []ResultadoProcesamiento, prefijo string) []ResultadoProcesamiento {
	var filtrados []ResultadoProcesamiento
	for _, r := range resultados {
		if strings.HasPrefix(r.RUC, prefijo) {
			filtrados = append(filtrados, r)
		}
	}
	return filtrados
}

func contarExitosos(resultados []ResultadoProcesamiento) int {
	count := 0
	for _, r := range resultados {
		if r.Exitoso {
			count++
		}
	}
	return count
}

func contarPorCampo(resultados []ResultadoProcesamiento, campo string) {
	conteo := make(map[string]int)
	for _, r := range resultados {
		if r.Exitoso {
			var valor string
			switch campo {
			case "Estado":
				valor = r.Estado
			case "Condicion":
				valor = r.Condicion
			}
			if valor != "" {
				conteo[valor]++
			}
		}
	}
	
	for k, v := range conteo {
		fmt.Printf("  - %s: %d\n", k, v)
	}
}

func generarCSV(resultados []ResultadoProcesamiento) {
	csvFile := fmt.Sprintf("resultados_masivos_%s.csv", time.Now().Format("20060102_150405"))
	file, err := os.Create(csvFile)
	if err != nil {
		log.Printf("Error creando CSV: %v", err)
		return
	}
	defer file.Close()

	// Encabezados
	file.WriteString("RUC,Tipo,Razon Social,Estado,Condicion,Exitoso,Error,Tiempo Proceso\n")
	
	// Datos
	for _, r := range resultados {
		line := fmt.Sprintf("%s,%s,\"%s\",%s,%s,%t,%s,%s\n",
			r.RUC,
			map[bool]string{true: "Juridica", false: "Natural"}[strings.HasPrefix(r.RUC, "20")],
			strings.ReplaceAll(r.RazonSocial, "\"", "'"),
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