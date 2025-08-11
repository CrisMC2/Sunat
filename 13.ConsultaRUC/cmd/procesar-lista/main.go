package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

type ResumenRUC struct {
	RUC                string    `json:"ruc"`
	TipoRUC            string    `json:"tipo_ruc"`
	RazonSocial        string    `json:"razon_social"`
	TipoContribuyente  string    `json:"tipo_contribuyente"`
	Estado             string    `json:"estado"`
	Condicion          string    `json:"condicion"`
	DomicilioFiscal    string    `json:"domicilio_fiscal"`
	FechaInscripcion   string    `json:"fecha_inscripcion"`
	
	// Resumen de consultas adicionales
	TieneDeudaCoactiva       bool   `json:"tiene_deuda_coactiva"`
	TieneOmisiones           bool   `json:"tiene_omisiones"`
	TieneActasProbatorias    bool   `json:"tiene_actas_probatorias"`
	CantidadRepresentantes   int    `json:"cantidad_representantes"`
	CantidadAnexos           int    `json:"cantidad_anexos"`
	ParticipaReactivaPeru    bool   `json:"participa_reactiva_peru"`
	ParticipaCovid19         bool   `json:"participa_covid19"`
	
	// Metadata
	Exitoso               bool      `json:"exitoso"`
	Error                 string    `json:"error,omitempty"`
	TiempoProceso         float64   `json:"tiempo_proceso_segundos"`
	FechaProceso          time.Time `json:"fecha_proceso"`
}

func main() {
	// Leer RUCs del archivo
	rucs, err := leerRUCsDeArchivo("rucs_test.txt")
	if err != nil {
		log.Fatal("Error leyendo archivo de RUCs:", err)
	}

	fmt.Println("=== PROCESAMIENTO MASIVO DE RUCs ===")
	fmt.Printf("Total de RUCs: %d\n", len(rucs))
	fmt.Printf("- Personas Jurídicas (20xxx): %d\n", contarPorPrefijo(rucs, "20"))
	fmt.Printf("- Personas Naturales (10xxx): %d\n\n", contarPorPrefijo(rucs, "10"))

	// Crear directorios
	os.MkdirAll("resultados_completos", 0755)
	os.MkdirAll("reportes", 0755)

	// Procesar por lotes para mejor estabilidad
	resultados := []ResumenRUC{}
	batchSize := 5
	
	for i := 0; i < len(rucs); i += batchSize {
		end := i + batchSize
		if end > len(rucs) {
			end = len(rucs)
		}
		
		batch := rucs[i:end]
		fmt.Printf("\n--- Procesando lote %d-%d ---\n", i+1, end)
		
		// Crear nuevo scraper para cada lote
		scraperOpt, err := scraper.NewScraperOptimizado()
		if err != nil {
			log.Printf("Error creando scraper: %v", err)
			continue
		}
		
		// Procesar RUCs del lote
		for j, ruc := range batch {
			numero := i + j + 1
			resultado := procesarRUC(scraperOpt, ruc, numero, len(rucs))
			resultados = append(resultados, resultado)
			
			// Pequeña pausa entre RUCs
			if j < len(batch)-1 {
				time.Sleep(2 * time.Second)
			}
		}
		
		// Cerrar scraper
		scraperOpt.Close()
		
		// Pausa entre lotes
		if end < len(rucs) {
			fmt.Println("\nPausa entre lotes (5 segundos)...")
			time.Sleep(5 * time.Second)
		}
	}

	// Generar reportes
	generarReportes(resultados)
}

func procesarRUC(scraper *scraper.ScraperOptimizado, ruc string, numero, total int) ResumenRUC {
	fmt.Printf("\n[%d/%d] ", numero, total)
	
	inicio := time.Now()
	resultado := ResumenRUC{
		RUC:          ruc,
		FechaProceso: time.Now(),
		Exitoso:      false,
	}
	
	// Determinar tipo
	if strings.HasPrefix(ruc, "20") {
		resultado.TipoRUC = "JURIDICA"
	} else {
		resultado.TipoRUC = "NATURAL"
	}
	
	// Intentar obtener información completa
	rucCompleto, err := scraper.ScrapeRUCCompleto(ruc)
	
	if err != nil {
		resultado.Error = err.Error()
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		resultado.Exitoso = true
		
		// Extraer información básica
		info := rucCompleto.InformacionBasica
		resultado.RazonSocial = info.RazonSocial
		resultado.TipoContribuyente = info.TipoContribuyente
		resultado.Estado = info.Estado
		resultado.Condicion = info.Condicion
		resultado.DomicilioFiscal = info.DomicilioFiscal
		resultado.FechaInscripcion = info.FechaInscripcion
		
		// Extraer resumen de consultas adicionales
		if rucCompleto.DeudaCoactiva != nil {
			resultado.TieneDeudaCoactiva = rucCompleto.DeudaCoactiva.CantidadDocumentos > 0
		}
		
		if rucCompleto.OmisionesTributarias != nil {
			resultado.TieneOmisiones = rucCompleto.OmisionesTributarias.TieneOmisiones
		}
		
		if rucCompleto.ActasProbatorias != nil {
			resultado.TieneActasProbatorias = rucCompleto.ActasProbatorias.TieneActas
		}
		
		if rucCompleto.RepresentantesLegales != nil {
			resultado.CantidadRepresentantes = len(rucCompleto.RepresentantesLegales.Representantes)
		}
		
		if rucCompleto.EstablecimientosAnexos != nil {
			resultado.CantidadAnexos = rucCompleto.EstablecimientosAnexos.CantidadAnexos
		}
		
		if rucCompleto.ReactivaPeru != nil {
			resultado.ParticipaReactivaPeru = rucCompleto.ReactivaPeru.ParticipaProgramma
		}
		
		if rucCompleto.ProgramaCovid19 != nil {
			resultado.ParticipaCovid19 = rucCompleto.ProgramaCovid19.ParticipaProgramma
		}
		
		// Guardar JSON completo
		jsonData, _ := json.MarshalIndent(rucCompleto, "", "  ")
		fileName := fmt.Sprintf("resultados_completos/ruc_%s.json", ruc)
		os.WriteFile(fileName, jsonData, 0644)
		
		fmt.Printf("✅ %s\n", resultado.RazonSocial)
	}
	
	resultado.TiempoProceso = time.Since(inicio).Seconds()
	return resultado
}

func generarReportes(resultados []ResumenRUC) {
	fmt.Println("\n=== GENERANDO REPORTES ===")
	
	// 1. Reporte JSON
	jsonData, _ := json.MarshalIndent(resultados, "", "  ")
	jsonFile := fmt.Sprintf("reportes/resumen_%s.json", time.Now().Format("20060102_150405"))
	os.WriteFile(jsonFile, jsonData, 0644)
	fmt.Printf("✅ Reporte JSON: %s\n", jsonFile)
	
	// 2. Reporte CSV
	csvFile := fmt.Sprintf("reportes/resumen_%s.csv", time.Now().Format("20060102_150405"))
	file, err := os.Create(csvFile)
	if err == nil {
		defer file.Close()
		
		writer := csv.NewWriter(file)
		defer writer.Flush()
		
		// Encabezados
		headers := []string{
			"RUC", "Tipo", "Razon Social", "Tipo Contribuyente", 
			"Estado", "Condicion", "Domicilio Fiscal", "Fecha Inscripcion",
			"Deuda Coactiva", "Omisiones", "Actas Probatorias",
			"Cant. Representantes", "Cant. Anexos", 
			"Reactiva Peru", "COVID-19", "Tiempo (seg)",
		}
		writer.Write(headers)
		
		// Datos
		for _, r := range resultados {
			record := []string{
				r.RUC,
				r.TipoRUC,
				r.RazonSocial,
				r.TipoContribuyente,
				r.Estado,
				r.Condicion,
				r.DomicilioFiscal,
				r.FechaInscripcion,
				fmt.Sprintf("%v", r.TieneDeudaCoactiva),
				fmt.Sprintf("%v", r.TieneOmisiones),
				fmt.Sprintf("%v", r.TieneActasProbatorias),
				fmt.Sprintf("%d", r.CantidadRepresentantes),
				fmt.Sprintf("%d", r.CantidadAnexos),
				fmt.Sprintf("%v", r.ParticipaReactivaPeru),
				fmt.Sprintf("%v", r.ParticipaCovid19),
				fmt.Sprintf("%.2f", r.TiempoProceso),
			}
			writer.Write(record)
		}
		
		fmt.Printf("✅ Reporte CSV: %s\n", csvFile)
	}
	
	// 3. Estadísticas
	fmt.Println("\n=== ESTADÍSTICAS ===")
	exitosos := 0
	juridicasActivas := 0
	naturalesActivas := 0
	conDeuda := 0
	conOmisiones := 0
	
	for _, r := range resultados {
		if r.Exitoso {
			exitosos++
			if r.Estado == "ACTIVO" {
				if r.TipoRUC == "JURIDICA" {
					juridicasActivas++
				} else {
					naturalesActivas++
				}
			}
			if r.TieneDeudaCoactiva {
				conDeuda++
			}
			if r.TieneOmisiones {
				conOmisiones++
			}
		}
	}
	
	fmt.Printf("Total procesados: %d\n", len(resultados))
	fmt.Printf("Exitosos: %d (%.1f%%)\n", exitosos, float64(exitosos)/float64(len(resultados))*100)
	fmt.Printf("Jurídicas activas: %d\n", juridicasActivas)
	fmt.Printf("Naturales activas: %d\n", naturalesActivas)
	fmt.Printf("Con deuda coactiva: %d\n", conDeuda)
	fmt.Printf("Con omisiones: %d\n", conOmisiones)
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

func leerRUCsDeArchivo(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rucs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ruc := strings.TrimSpace(scanner.Text())
		if ruc != "" && (strings.HasPrefix(ruc, "10") || strings.HasPrefix(ruc, "20")) {
			rucs = append(rucs, ruc)
		}
	}

	return rucs, scanner.Err()
}