package main

import (
	"log"
	"os"
	"strings"

	"github.com/consulta-ruc-scraper/pkg/database"
	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/scraper"
)

// Función main modificada
func main() {
	rucs := []string{"20606316977"}
	if len(os.Args) > 1 {
		rucs = os.Args[1:]
	}

	// Configuración de la base de datos
	dbConnectionString := os.Getenv("DATABASE_URL")
	if dbConnectionString == "" {
		dbConnectionString = "postgres://postgres:admin123@localhost:5433/sunat?sslmode=disable"
	}

	// Crear conexión a la base de datos
	dbService, err := database.NewDatabaseService(dbConnectionString)
	if err != nil {
		log.Fatal("Error conectando a la base de datos:", err)
	}
	defer dbService.Close()

	// Crear scraper extendido
	scraperExt, err := scraper.NewScraperExtendido()
	if err != nil {
		log.Fatal("Error creating extended scraper:", err)
	}
	defer scraperExt.Close()

	for i, ruc := range rucs {
		log.Printf("[%d/%d] Procesando RUC: %s", i+1, len(rucs), ruc)

		// Obtener información completa del RUC (ahora pasa dbService)
		rucCompleto, err := scraperExt.ScrapeRUCCompleto(ruc, dbService)

		// Guardar en la base de datos incluso si hay errores parciales
		if rucCompleto != nil {
			// Insertar en la base de datos
			dbErr := dbService.InsertRUCCompleto(rucCompleto)
			if dbErr != nil {
				log.Printf("❌ Error guardando RUC %s en la base de datos: %v", ruc, dbErr)
				continue
			}

			// Determinar el tipo de resultado
			if err != nil {
				log.Printf("⚠️ RUC %s (%s) guardado con datos parciales - Error: %v",
					ruc, getValueOrDefault(rucCompleto.InformacionBasica.RazonSocial), err)
			} else {
				log.Printf("✅ RUC %s (%s) guardado exitosamente",
					ruc, getValueOrDefault(rucCompleto.InformacionBasica.RazonSocial))
			}

			showSummary(rucCompleto)
		} else {
			log.Printf("❌ Error crítico obteniendo información del RUC %s: %v", ruc, err)
			continue
		}
	}
	log.Println("Proceso completado.")
}

func showSummary(ruc *models.RUCCompleto) {
	log.Printf("   Estado: %s | Condición: %s",
		getValueOrDefault(ruc.InformacionBasica.Estado),
		getValueOrDefault(ruc.InformacionBasica.Condicion))

	// Información adicional disponible
	var info []string
	if ruc.DeudaCoactiva != nil && ruc.DeudaCoactiva.CantidadDocumentos > 0 {
		info = append(info, "Deuda Coactiva")
	}
	if ruc.RepresentantesLegales != nil {
		vigentes := countActiveRepresentatives(ruc.RepresentantesLegales.Representantes)
		if vigentes > 0 {
			info = append(info, "Representantes")
		}
	}
	if ruc.EstablecimientosAnexos != nil && ruc.EstablecimientosAnexos.CantidadAnexos > 0 {
		info = append(info, "Establecimientos")
	}
	if ruc.OmisionesTributarias != nil && ruc.OmisionesTributarias.TieneOmisiones {
		info = append(info, "Omisiones")
	}
	if ruc.ActasProbatorias != nil && ruc.ActasProbatorias.TieneActas {
		info = append(info, "Actas")
	}

	if len(info) > 0 {
		log.Printf("   Datos adicionales: %s", strings.Join(info, ", "))
	}
}

func getValueOrDefault(value string) string {
	if value == "" || value == "-" || value == "No hay información" {
		return "N/A"
	}
	return value
}

func countActiveRepresentatives(reps []models.RepresentanteLegal) int {
	count := 0
	for _, rep := range reps {
		if rep.Vigente {
			count++
		}
	}
	return count
}
