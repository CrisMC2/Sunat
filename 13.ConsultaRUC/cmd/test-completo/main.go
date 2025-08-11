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

type ResumenRUC struct {
	RUC                   string    `json:"ruc"`
	TipoRUC               string    `json:"tipo_ruc"` // JURIDICA o NATURAL
	RazonSocial           string    `json:"razon_social"`
	Estado                string    `json:"estado"`
	Condicion             string    `json:"condicion"`
	ConsultasDisponibles  []string  `json:"consultas_disponibles"`
	ConsultasRealizadas   []string  `json:"consultas_realizadas"`
	TieneDeudaCoactiva    bool      `json:"tiene_deuda_coactiva"`
	TieneRepresentantes   bool      `json:"tiene_representantes"`
	CantidadAnexos        int       `json:"cantidad_anexos"`
	Exitoso               bool      `json:"exitoso"`
	Error                 string    `json:"error,omitempty"`
	TiempoProceso         float64   `json:"tiempo_proceso_segundos"`
}

func main() {
	// RUCs de prueba - solo algunos para test completo
	rucs := []string{
		// Personas Jurídicas
		"20606454466", "20393261162", "20600656288",
		// Personas Naturales
		"10719706288", "10775397131", "10420242986",
	}

	fmt.Println("=== TEST COMPLETO DE SCRAPER SUNAT ===")
	fmt.Printf("Consultando TODA la información disponible para cada RUC\n")
	fmt.Printf("Total de RUCs: %d (3 jurídicas, 3 naturales)\n\n", len(rucs))

	// Crear scraper extendido
	scraperExt, err := scraper.NewScraperExtendido()
	if err != nil {
		log.Fatal("Error creando scraper extendido:", err)
	}
	defer scraperExt.Close()

	resultados := []ResumenRUC{}
	
	for i, ruc := range rucs {
		fmt.Printf("\n[%d/%d] Procesando RUC %s\n", i+1, len(rucs), ruc)
		fmt.Println(strings.Repeat("-", 50))
		
		inicio := time.Now()
		resumen := ResumenRUC{
			RUC: ruc,
		}
		
		// Determinar tipo de RUC
		if strings.HasPrefix(ruc, "20") {
			resumen.TipoRUC = "JURIDICA"
			resumen.ConsultasDisponibles = []string{
				"Información Histórica",
				"Deuda Coactiva", 
				"Omisiones Tributarias",
				"Cantidad de Trabajadores",
				"Actas Probatorias",
				"Facturas Físicas",
				"Reactiva Perú",
				"Programa COVID-19",
				"Representantes Legales",
				"Establecimientos Anexos",
			}
		} else if strings.HasPrefix(ruc, "10") {
			resumen.TipoRUC = "NATURAL"
			resumen.ConsultasDisponibles = []string{
				"Información Histórica",
				"Deuda Coactiva",
				"Omisiones Tributarias",
				"Cantidad de Trabajadores",
				"Actas Probatorias",
				"Facturas Físicas",
			}
		}
		
		// Obtener información completa
		fmt.Println("📋 Obteniendo información básica...")
		rucCompleto, err := scraperExt.ScrapeRUCCompleto(ruc)
		
		if err != nil {
			resumen.Exitoso = false
			resumen.Error = err.Error()
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			resumen.Exitoso = true
			resumen.RazonSocial = rucCompleto.InformacionBasica.RazonSocial
			resumen.Estado = rucCompleto.InformacionBasica.Estado
			resumen.Condicion = rucCompleto.InformacionBasica.Condicion
			
			// Verificar qué consultas se realizaron exitosamente
			resumen.ConsultasRealizadas = []string{"Información Básica"}
			
			if rucCompleto.InformacionHistorica != nil {
				resumen.ConsultasRealizadas = append(resumen.ConsultasRealizadas, "Información Histórica")
				fmt.Println("✅ Información Histórica obtenida")
			}
			
			if rucCompleto.DeudaCoactiva != nil {
				resumen.ConsultasRealizadas = append(resumen.ConsultasRealizadas, "Deuda Coactiva")
				resumen.TieneDeudaCoactiva = rucCompleto.DeudaCoactiva.CantidadDocumentos > 0
				fmt.Printf("✅ Deuda Coactiva: %s\n", 
					map[bool]string{true: "SÍ", false: "NO"}[resumen.TieneDeudaCoactiva])
			}
			
			if rucCompleto.RepresentantesLegales != nil {
				resumen.ConsultasRealizadas = append(resumen.ConsultasRealizadas, "Representantes Legales")
				resumen.TieneRepresentantes = len(rucCompleto.RepresentantesLegales.Representantes) > 0
				fmt.Printf("✅ Representantes Legales: %d encontrados\n", 
					len(rucCompleto.RepresentantesLegales.Representantes))
			}
			
			if rucCompleto.EstablecimientosAnexos != nil {
				resumen.ConsultasRealizadas = append(resumen.ConsultasRealizadas, "Establecimientos Anexos")
				resumen.CantidadAnexos = rucCompleto.EstablecimientosAnexos.CantidadAnexos
				fmt.Printf("✅ Establecimientos Anexos: %d\n", resumen.CantidadAnexos)
			}
			
			if rucCompleto.CantidadTrabajadores != nil {
				resumen.ConsultasRealizadas = append(resumen.ConsultasRealizadas, "Cantidad de Trabajadores")
				fmt.Println("✅ Cantidad de Trabajadores obtenida")
			}
			
			if rucCompleto.OmisionesTributarias != nil {
				resumen.ConsultasRealizadas = append(resumen.ConsultasRealizadas, "Omisiones Tributarias")
				fmt.Printf("✅ Omisiones Tributarias: %s\n",
					map[bool]string{true: "SÍ", false: "NO"}[rucCompleto.OmisionesTributarias.TieneOmisiones])
			}
			
			// Guardar JSON completo
			os.MkdirAll("resultados_completos", 0755)
			jsonData, _ := json.MarshalIndent(rucCompleto, "", "  ")
			fileName := fmt.Sprintf("resultados_completos/ruc_completo_%s.json", ruc)
			os.WriteFile(fileName, jsonData, 0644)
			fmt.Printf("💾 Datos completos guardados en: %s\n", fileName)
		}
		
		resumen.TiempoProceso = time.Since(inicio).Seconds()
		resultados = append(resultados, resumen)
		
		// Mostrar resumen
		fmt.Printf("\n📊 Resumen para RUC %s:\n", ruc)
		fmt.Printf("   - Tipo: %s\n", resumen.TipoRUC)
		fmt.Printf("   - Consultas disponibles: %d\n", len(resumen.ConsultasDisponibles))
		fmt.Printf("   - Consultas realizadas: %d\n", len(resumen.ConsultasRealizadas))
		fmt.Printf("   - Tiempo: %.2f segundos\n", resumen.TiempoProceso)
		
		// Pausa entre RUCs
		if i < len(rucs)-1 {
			fmt.Println("\nEsperando antes del siguiente RUC...")
			time.Sleep(3 * time.Second)
		}
	}
	
	// Generar reporte final
	generarReporteFinal(resultados)
}

func generarReporteFinal(resultados []ResumenRUC) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("REPORTE FINAL - TEST COMPLETO")
	fmt.Println(strings.Repeat("=", 60))
	
	exitosos := 0
	totalConsultasRealizadas := 0
	totalConsultasDisponibles := 0
	
	for _, r := range resultados {
		if r.Exitoso {
			exitosos++
		}
		totalConsultasRealizadas += len(r.ConsultasRealizadas)
		totalConsultasDisponibles += len(r.ConsultasDisponibles)
	}
	
	fmt.Printf("\nRUCs procesados: %d\n", len(resultados))
	fmt.Printf("Exitosos: %d (%.1f%%)\n", exitosos, float64(exitosos)/float64(len(resultados))*100)
	fmt.Printf("\nConsultas totales disponibles: %d\n", totalConsultasDisponibles)
	fmt.Printf("Consultas realizadas: %d (%.1f%%)\n", 
		totalConsultasRealizadas, 
		float64(totalConsultasRealizadas)/float64(totalConsultasDisponibles)*100)
	
	// Análisis por tipo
	fmt.Println("\n--- Por Tipo de RUC ---")
	
	// Jurídicas
	juridicas := filtrarPorTipo(resultados, "JURIDICA")
	if len(juridicas) > 0 {
		fmt.Printf("\nPersonas Jurídicas (20xxx):\n")
		for _, j := range juridicas {
			fmt.Printf("  • %s - %s\n", j.RUC, j.RazonSocial)
			fmt.Printf("    Consultas: %d/%d\n", 
				len(j.ConsultasRealizadas), len(j.ConsultasDisponibles))
		}
	}
	
	// Naturales
	naturales := filtrarPorTipo(resultados, "NATURAL")
	if len(naturales) > 0 {
		fmt.Printf("\nPersonas Naturales (10xxx):\n")
		for _, n := range naturales {
			fmt.Printf("  • %s - %s\n", n.RUC, n.RazonSocial)
			fmt.Printf("    Consultas: %d/%d\n", 
				len(n.ConsultasRealizadas), len(n.ConsultasDisponibles))
		}
	}
	
	// Guardar reporte
	reporte := map[string]interface{}{
		"fecha":                    time.Now().Format("2006-01-02 15:04:05"),
		"total_rucs":               len(resultados),
		"exitosos":                 exitosos,
		"consultas_disponibles":    totalConsultasDisponibles,
		"consultas_realizadas":     totalConsultasRealizadas,
		"tasa_completitud":         fmt.Sprintf("%.1f%%", float64(totalConsultasRealizadas)/float64(totalConsultasDisponibles)*100),
		"resultados":               resultados,
	}
	
	reporteJSON, _ := json.MarshalIndent(reporte, "", "  ")
	reporteFile := fmt.Sprintf("reporte_completo_%s.json", time.Now().Format("20060102_150405"))
	os.WriteFile(reporteFile, reporteJSON, 0644)
	fmt.Printf("\n✅ Reporte completo guardado en: %s\n", reporteFile)
}

func filtrarPorTipo(resultados []ResumenRUC, tipo string) []ResumenRUC {
	var filtrados []ResumenRUC
	for _, r := range resultados {
		if r.TipoRUC == tipo {
			filtrados = append(filtrados, r)
		}
	}
	return filtrados
}