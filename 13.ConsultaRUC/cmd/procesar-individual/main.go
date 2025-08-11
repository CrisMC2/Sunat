package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <RUC>")
		fmt.Println("Ejemplo: go run main.go 20606642131")
		os.Exit(1)
	}

	ruc := os.Args[1]
	outputDir := "data_completa"

	fmt.Printf("\n╔════════════════════════════════════════════════╗\n")
	fmt.Printf("║   PROCESAMIENTO INDIVIDUAL RUC: %s   ║\n", ruc)
	fmt.Printf("╚════════════════════════════════════════════════╝\n")

	// Crear directorio si no existe
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Error creando directorio:", err)
	}

	// Determinar tipo
	tipo := "Persona Natural"
	if strings.HasPrefix(ruc, "20") {
		tipo = "Persona Jurídica"
	}
	fmt.Printf("\nTipo: %s\n", tipo)

	// Crear scraper
	s, err := scraper.NewScraperOptimizado()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer s.Close()

	// Procesar
	fmt.Println("\nProcesando...")
	inicio := time.Now()
	
	resultado, err := s.ScrapeRUCCompleto(ruc)
	if err != nil {
		log.Fatal("Error procesando RUC:", err)
	}

	// Guardar
	filename := filepath.Join(outputDir, fmt.Sprintf("ruc_%s_completo.json", ruc))
	jsonData, _ := json.MarshalIndent(resultado, "", "  ")
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Fatal("Error guardando archivo:", err)
	}

	// Resumen
	fmt.Printf("\n✅ PROCESADO EXITOSAMENTE\n")
	fmt.Printf("📋 Razón Social: %s\n", resultado.InformacionBasica.RazonSocial)
	fmt.Printf("📋 Estado: %s - %s\n", resultado.InformacionBasica.Estado, resultado.InformacionBasica.Condicion)
	fmt.Printf("⏱️  Tiempo: %v\n", time.Since(inicio).Round(time.Second))
	fmt.Printf("💾 Guardado en: %s\n", filename)
	fmt.Printf("📦 Tamaño: %d bytes\n", len(jsonData))
}