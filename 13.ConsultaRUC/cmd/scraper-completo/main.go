package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	rucs := []string{"20606316977"}

	if len(os.Args) > 1 {
		rucs = os.Args[1:]
	}

	// Crear scraper extendido
	scraperExt, err := scraper.NewScraperExtendido()
	if err != nil {
		log.Fatal("Error creating extended scraper:", err)
	}
	defer scraperExt.Close()

	for _, ruc := range rucs {
		fmt.Printf("\n=== Consultando RUC: %s ===\n", ruc)
		// Opción 1: Obtener TODA la información disponible
		fmt.Println("\n1. Información Completa (todas las consultas):")
		fmt.Println("   Obteniendo toda la información disponible...")

		rucCompleto, err := scraperExt.ScrapeRUCCompleto(ruc)
		if err != nil {
			log.Printf("Error obteniendo información completa de RUC %s: %v\n", ruc, err)
			continue
		}

		// Guardar JSON completo
		fileName := fmt.Sprintf("ruc_completo_%s.json", ruc)
		jsonData, err := json.MarshalIndent(rucCompleto, "", "  ")
		if err != nil {
			log.Printf("Error marshaling data for RUC %s: %v\n", ruc, err)
			continue
		}

		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			log.Printf("Error saving data for RUC %s: %v\n", ruc, err)
			continue
		}

		fmt.Printf("\n   ✅ Información completa guardada en: %s\n", fileName)

		// Mostrar resumen
		fmt.Println("\n   Resumen:")
		fmt.Printf("   - Razón Social: %s\n", rucCompleto.InformacionBasica.RazonSocial)
		fmt.Printf("   - Estado: %s\n", rucCompleto.InformacionBasica.Estado)
		fmt.Printf("   - Condición: %s\n", rucCompleto.InformacionBasica.Condicion)

		if rucCompleto.DeudaCoactiva != nil {
			fmt.Printf("   - Deuda Coactiva: %s\n",
				map[bool]string{true: "SÍ", false: "NO"}[rucCompleto.DeudaCoactiva.CantidadDocumentos > 0])
		}

		if rucCompleto.RepresentantesLegales != nil {
			vigentes := 0
			for _, r := range rucCompleto.RepresentantesLegales.Representantes {
				if r.Vigente {
					vigentes++
				}
			}
			fmt.Printf("   - Representantes Legales Vigentes: %d\n", vigentes)
		}

		if rucCompleto.EstablecimientosAnexos != nil {
			fmt.Printf("   - Establecimientos Anexos: %d\n", rucCompleto.EstablecimientosAnexos.CantidadAnexos)
		}

		fmt.Println("\n" + strings.Repeat("=", 50))
	}
}
