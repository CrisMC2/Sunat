package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	// RUCs de prueba: mezclados tipo 10 y 20
	rucs := []string{
		"20606454466", // Jurídica
		"10719706288", // Natural
		"20393261162", // Jurídica
		"10775397131", // Natural
	}

	fmt.Println("=== SCRAPER OPTIMIZADO - PRUEBA ===")
	fmt.Println("Usando navegación con botón 'Volver'")
	fmt.Println("Sin recargar página entre consultas")
	fmt.Printf("RUCs a procesar: %d\n", len(rucs))
	fmt.Println("")

	// Crear scraper optimizado
	scraperOpt, err := scraper.NewScraperOptimizado()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer scraperOpt.Close()

	// Procesar cada RUC
	for i, ruc := range rucs {
		inicio := time.Now()
		
		fmt.Printf("[%d/%d] ", i+1, len(rucs))
		
		// Obtener información completa
		rucCompleto, err := scraperOpt.ScrapeRUCCompleto(ruc)
		
		if err != nil {
			fmt.Printf("❌ Error procesando RUC %s: %v\n", ruc, err)
			continue
		}

		// Guardar JSON
		os.MkdirAll("resultados_optimizados", 0755)
		jsonData, _ := json.MarshalIndent(rucCompleto, "", "  ")
		fileName := fmt.Sprintf("resultados_optimizados/ruc_%s.json", ruc)
		os.WriteFile(fileName, jsonData, 0644)

		// Mostrar resumen
		tiempo := time.Since(inicio)
		fmt.Printf("   ⏱️  Tiempo: %.2f segundos\n", tiempo.Seconds())
		fmt.Printf("   💾 Guardado en: %s\n", fileName)
		
		// Mostrar información clave
		info := rucCompleto.InformacionBasica
		fmt.Printf("   📊 %s - %s (%s)\n", info.RazonSocial, info.Estado, info.Condicion)
		
		// Resumen de consultas adicionales
		consultasRealizadas := 2 // Básica + al menos una más
		if rucCompleto.DeudaCoactiva != nil {
			consultasRealizadas++
		}
		if rucCompleto.RepresentantesLegales != nil {
			consultasRealizadas++
		}
		if rucCompleto.CantidadTrabajadores != nil {
			consultasRealizadas++
		}
		
		fmt.Printf("   ✅ Consultas realizadas: %d\n\n", consultasRealizadas)
	}

	fmt.Println("✅ Prueba completada")
}