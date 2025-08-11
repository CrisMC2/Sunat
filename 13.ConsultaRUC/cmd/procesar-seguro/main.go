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

func procesarRUCSeguro(ruc string) (*models.RUCCompleto, error) {
	// Crear nuevo scraper para cada RUC
	s, err := scraper.NewScraperOptimizado()
	if err != nil {
		return nil, err
	}
	defer func() {
		// Asegurar cierre limpio
		if r := recover(); r != nil {
			fmt.Printf("   ⚠️  Recuperando de error: %v\n", r)
		}
		s.Close()
	}()

	return s.ScrapeRUCCompleto(ruc)
}

func main() {
	fmt.Println("\n=== PROCESAMIENTO SEGURO DE RUCs ===")
	
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

	fmt.Printf("Total de RUCs a procesar: %d\n", len(rucs))
	
	// Contar por tipo
	juridicas := 0
	naturales := 0
	for _, ruc := range rucs {
		if strings.HasPrefix(ruc, "20") {
			juridicas++
		} else {
			naturales++
		}
	}
	
	fmt.Printf("- Personas Jurídicas (20xxx): %d\n", juridicas)
	fmt.Printf("- Personas Naturales (10xxx): %d\n\n", naturales)

	// Procesar RUCs
	exitosos := 0
	errores := 0
	resultados := []models.RUCCompleto{}
	
	for i, ruc := range rucs {
		fmt.Printf("[%d/%d] Procesando RUC: %s\n", i+1, len(rucs), ruc)
		
		tipo := "Persona Natural"
		if strings.HasPrefix(ruc, "20") {
			tipo = "Persona Jurídica"
		}
		fmt.Printf("   Tipo: %s\n", tipo)
		
		// Procesar con manejo de errores
		resultado, err := procesarRUCSeguro(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n\n", err)
			errores++
			// Continuar con el siguiente RUC
			continue
		}
		
		if resultado != nil {
			// Guardar resultado individual
			jsonData, _ := json.MarshalIndent(resultado, "", "  ")
			filename := fmt.Sprintf("ruc_completo_%s.json", ruc)
			os.WriteFile(filename, jsonData, 0644)
			
			fmt.Printf("   ✓ %s - %s\n", resultado.InformacionBasica.RazonSocial, resultado.InformacionBasica.Estado)
			fmt.Printf("   ✓ Guardado en: %s\n", filename)
			
			// Mostrar resumen de información obtenida
			if resultado.InformacionHistorica != nil {
				fmt.Printf("   📋 Información histórica: ✓\n")
			}
			if resultado.DeudaCoactiva != nil {
				if resultado.DeudaCoactiva.TotalDeuda > 0 {
					fmt.Printf("   💰 Deuda coactiva: S/ %.2f\n", resultado.DeudaCoactiva.TotalDeuda)
				} else {
					fmt.Printf("   💰 Deuda coactiva: SIN DEUDA\n")
				}
			}
			if resultado.RepresentantesLegales != nil && len(resultado.RepresentantesLegales.Representantes) > 0 {
				fmt.Printf("   👥 Representantes legales: %d\n", len(resultado.RepresentantesLegales.Representantes))
			}
			
			resultados = append(resultados, *resultado)
			exitosos++
		}
		
		fmt.Println()
		
		// Pausa entre RUCs para evitar sobrecarga
		if i < len(rucs)-1 {
			fmt.Println("   ⏳ Esperando 3 segundos antes del siguiente RUC...")
			time.Sleep(3 * time.Second)
		}
	}

	// Resumen final
	fmt.Println("\n=== RESUMEN DE PROCESAMIENTO ===")
	fmt.Printf("✓ Exitosos: %d RUCs\n", exitosos)
	fmt.Printf("✗ Errores: %d RUCs\n", errores)
	fmt.Printf("Total: %d RUCs\n", len(rucs))
	
	// Guardar resumen consolidado
	if len(resultados) > 0 {
		timestamp := time.Now().Format("20060102_150405")
		resumenFile := fmt.Sprintf("resumen_completo_%s.json", timestamp)
		jsonData, _ := json.MarshalIndent(resultados, "", "  ")
		os.WriteFile(resumenFile, jsonData, 0644)
		fmt.Printf("\n✓ Resumen consolidado guardado en: %s\n", resumenFile)
	}
}