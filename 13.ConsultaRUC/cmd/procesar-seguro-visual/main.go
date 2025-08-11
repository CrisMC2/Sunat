package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("\nUso: go run main.go <RUC> [visual]")
		fmt.Println("Ejemplos:")
		fmt.Println("  go run main.go 20606642131        # Sin visualización")
		fmt.Println("  go run main.go 20606642131 visual # Con visualización")
		os.Exit(1)
	}

	ruc := os.Args[1]
	visual := false
	
	if len(os.Args) > 2 && os.Args[2] == "visual" {
		visual = true
	}

	fmt.Printf("\n╔════════════════════════════════════════════════╗\n")
	fmt.Printf("║         SCRAPER SEGURO - SUNAT                 ║\n")
	fmt.Printf("╚════════════════════════════════════════════════╝\n")
	fmt.Printf("\n📌 RUC: %s\n", ruc)
	
	if visual {
		fmt.Println("👀 Modo: VISUAL (verás el navegador)")
	} else {
		fmt.Println("🚀 Modo: RÁPIDO (sin visualización)")
	}

	// Crear scraper
	scraper, err := scraper.NewScraperSeguro(visual)
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer scraper.Close()

	// Procesar RUC
	inicio := time.Now()
	resultado, err := scraper.ScrapeRUCCompleto(ruc)
	if err != nil {
		log.Fatal("Error procesando RUC:", err)
	}
	duracion := time.Since(inicio)

	// Guardar resultado
	outputDir := "data_completa"
	os.MkdirAll(outputDir, 0755)
	
	filename := filepath.Join(outputDir, fmt.Sprintf("ruc_%s_seguro.json", ruc))
	jsonData, _ := json.MarshalIndent(resultado, "", "  ")
	os.WriteFile(filename, jsonData, 0644)

	// Resumen
	fmt.Printf("\n✅ PROCESAMIENTO COMPLETADO\n")
	fmt.Printf("📋 Razón Social: %s\n", resultado.InformacionBasica.RazonSocial)
	fmt.Printf("📋 Estado: %s - %s\n", resultado.InformacionBasica.Estado, resultado.InformacionBasica.Condicion)
	fmt.Printf("⏱️  Tiempo total: %v\n", duracion.Round(time.Second))
	fmt.Printf("💾 Archivo guardado: %s\n", filename)
	fmt.Printf("📊 Tamaño: %d bytes\n", len(jsonData))

	// Mostrar qué información se obtuvo
	fmt.Printf("\n📑 INFORMACIÓN OBTENIDA:\n")
	if resultado.InformacionHistorica != nil {
		fmt.Printf("   ✓ Información Histórica\n")
	}
	if resultado.DeudaCoactiva != nil {
		fmt.Printf("   ✓ Deuda Coactiva\n")
	}
	if resultado.OmisionesTributarias != nil {
		fmt.Printf("   ✓ Omisiones Tributarias\n")
	}
	if resultado.CantidadTrabajadores != nil {
		fmt.Printf("   ✓ Cantidad de Trabajadores\n")
	}
	if resultado.ActasProbatorias != nil {
		fmt.Printf("   ✓ Actas Probatorias\n")
	}
	if resultado.RepresentantesLegales != nil {
		fmt.Printf("   ✓ Representantes Legales\n")
	}
	if resultado.EstablecimientosAnexos != nil {
		fmt.Printf("   ✓ Establecimientos Anexos\n")
	}
	if resultado.ReactivaPeru != nil {
		fmt.Printf("   ✓ Reactiva Perú\n")
	}
	if resultado.ProgramaCovid19 != nil {
		fmt.Printf("   ✓ Programa COVID-19\n")
	}
}