package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	fmt.Println("\n=== PROCESAMIENTO SIMPLIFICADO DE RUCs ===")

	// Leer archivo de RUCs
	data, err := os.ReadFile("rucs_test.txt")
	if err != nil {
		log.Fatal("Error leyendo archivo:", err)
	}

	// Limpiar y filtrar RUCs
	rucs := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		ruc := strings.TrimSpace(line)
		if ruc != "" && (strings.HasPrefix(ruc, "10") || strings.HasPrefix(ruc, "20")) {
			rucs = append(rucs, ruc)
		}
	}

	fmt.Printf("Total de RUCs a procesar: %d\n\n", len(rucs))

	// Crear scraper optimizado
	s, err := scraper.NewScraperOptimizado()
	if err != nil {
		log.Fatalf("Error al inicializar el scraper: %v", err)
	}
	defer s.Close()

	resultados := []models.RUCCompleto{}

	for i, ruc := range rucs {
		fmt.Printf("[%d/%d] Procesando RUC: %s\n", i+1, len(rucs), ruc)

		// Obtener información completa
		resultado, err := s.ScrapeRUCCompleto(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		// Guardar resultado individual
		jsonData, _ := json.MarshalIndent(resultado, "", "  ")
		filename := fmt.Sprintf("ruc_completo_%s.json", ruc)
		os.WriteFile(filename, jsonData, 0644)

		fmt.Printf("   ✓ %s - %s\n", resultado.InformacionBasica.RUC, resultado.InformacionBasica.RazonSocial)
		fmt.Printf("   ✓ Guardado en: %s\n\n", filename)

		resultados = append(resultados, *resultado)

		// Pausa entre consultas
		time.Sleep(2 * time.Second)
	}

	// Guardar resumen final
	resumenFile := fmt.Sprintf("resumen_rucs_%s.json", time.Now().Format("20060102_150405"))
	jsonData, _ := json.MarshalIndent(resultados, "", "  ")
	os.WriteFile(resumenFile, jsonData, 0644)

	fmt.Printf("\n=== PROCESAMIENTO COMPLETADO ===\n")
	fmt.Printf("✓ Procesados: %d RUCs\n", len(resultados))
	fmt.Printf("✓ Resumen guardado en: %s\n", resumenFile)
}
