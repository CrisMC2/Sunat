package main

import (
	"fmt"
	"log"
	"os"

	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	ruc := "20606454466"
	if len(os.Args) > 1 {
		ruc = os.Args[1]
	}

	fmt.Printf("=== TEST DE BOTONES ADICIONALES ===\n")
	fmt.Printf("RUC: %s\n\n", ruc)

	// Crear scraper extendido
	scraperExt, err := scraper.NewScraperExtendido()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer scraperExt.Close()

	// 1. Información básica (funciona)
	fmt.Println("1. Información Básica:")
	info, err := scraperExt.ScrapeRUC(ruc)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   ✅ %s - %s\n", info.RazonSocial, info.Estado)
	}

	// 2. Probar cada botón individualmente
	fmt.Println("\n2. Probando botones adicionales:")

	// Información Histórica
	fmt.Print("   - Información Histórica: ")
	if hist, err := scraperExt.ScrapeInformacionHistorica(ruc); err == nil {
		fmt.Printf("✅ (Cambios domicilio: %d)\n", len(hist.CambiosDomicilio))
	} else {
		fmt.Printf("❌ %v\n", err)
	}

	// Deuda Coactiva
	fmt.Print("   - Deuda Coactiva: ")
	if deuda, err := scraperExt.ScrapeDeudaCoactiva(ruc); err == nil {
		if deuda.CantidadDocumentos == 0 {
			fmt.Println("✅ Sin deudas")
		} else {
			fmt.Printf("✅ %d documentos, Total: S/ %.2f\n", 
				deuda.CantidadDocumentos, deuda.TotalDeuda)
		}
	} else {
		fmt.Printf("❌ %v\n", err)
	}

	// Representantes Legales (solo para personas jurídicas)
	fmt.Print("   - Representantes Legales: ")
	if reps, err := scraperExt.ScrapeRepresentantesLegales(ruc); err == nil {
		fmt.Printf("✅ %d representantes\n", len(reps.Representantes))
		for _, rep := range reps.Representantes {
			fmt.Printf("      • %s %s %s - %s\n", 
				rep.ApellidoPaterno, rep.ApellidoMaterno, rep.Nombres, rep.Cargo)
		}
	} else {
		fmt.Printf("❌ %v\n", err)
	}

	// Cantidad de Trabajadores
	fmt.Print("   - Cantidad de Trabajadores: ")
	if trab, err := scraperExt.ScrapeCantidadTrabajadores(ruc); err == nil {
		fmt.Printf("✅ Periodos disponibles: %d\n", len(trab.PeriodosDisponibles))
	} else {
		fmt.Printf("❌ %v\n", err)
	}

	// Establecimientos Anexos
	fmt.Print("   - Establecimientos Anexos: ")
	if estab, err := scraperExt.ScrapeEstablecimientosAnexos(ruc); err == nil {
		fmt.Printf("✅ %d anexos\n", estab.CantidadAnexos)
	} else {
		fmt.Printf("❌ %v\n", err)
	}

	fmt.Println("\n✅ Test completado")
}