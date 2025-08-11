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

		// Opción 1: Obtener solo información básica
		fmt.Println("\n1. Información Básica:")
		infoBasica, err := scraperExt.ScrapeRUC(ruc)
		if err != nil {
			log.Printf("Error obteniendo información básica de RUC %s: %v\n", ruc, err)
			continue
		}

		basicJSON, _ := json.MarshalIndent(infoBasica, "", "  ")
		fmt.Println(string(basicJSON))

		// Opción 2: Obtener información específica
		fmt.Println("\n2. Información Adicional Específica:")

		// Representantes Legales
		fmt.Println("\n   - Representantes Legales:")
		if reps, err := scraperExt.ScrapeRepresentantesLegales(ruc); err == nil {
			for _, rep := range reps.Representantes {
				fmt.Printf("     * %s %s %s - %s (%s)\n",
					rep.ApellidoPaterno, rep.ApellidoMaterno, rep.Nombres,
					rep.Cargo,
					map[bool]string{true: "Vigente", false: "No vigente"}[rep.Vigente])
			}
		} else {
			fmt.Printf("     Error: %v\n", err)
		}

		// Deuda Coactiva
		fmt.Println("\n   - Deuda Coactiva:")
		if deuda, err := scraperExt.ScrapeDeudaCoactiva(ruc); err == nil {
			if deuda.CantidadDocumentos == 0 {
				fmt.Println("     No registra deuda coactiva")
			} else {
				fmt.Printf("     Total deuda: S/ %.2f\n", deuda.TotalDeuda)
				fmt.Printf("     Cantidad de documentos: %d\n", deuda.CantidadDocumentos)
			}
		} else {
			fmt.Printf("     Error: %v\n", err)
		}

		// Establecimientos Anexos
		fmt.Println("\n   - Establecimientos Anexos:")
		if estab, err := scraperExt.ScrapeEstablecimientosAnexos(ruc); err == nil {
			if estab.CantidadAnexos == 0 {
				fmt.Println("     No registra establecimientos anexos")
			} else {
				fmt.Printf("     Cantidad de anexos: %d\n", estab.CantidadAnexos)
				for _, e := range estab.Establecimientos {
					fmt.Printf("     * %s - %s (%s)\n",
						e.CodigoEstablecimiento,
						e.Direccion,
						e.Estado)
				}
			}
		} else {
			fmt.Printf("     Error: %v\n", err)
		}

		// Opción 3: Obtener TODA la información disponible
		fmt.Println("\n3. Información Completa (todas las consultas):")
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
