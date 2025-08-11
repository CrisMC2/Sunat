#!/bin/bash

# Test del scraper extendido con un solo RUC
echo "=== TEST SCRAPER EXTENDIDO - UN RUC ==="
echo ""
echo "Este test intentará obtener TODA la información disponible"
echo "incluyendo hacer clic en todos los botones adicionales"
echo ""

export PATH="/usr/local/go/bin:$PATH"

# Crear programa de prueba simple
cat > test_extendido.go << 'EOF'
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

	fmt.Printf("Probando scraper extendido con RUC: %s\n\n", ruc)

	// Crear scraper extendido
	scraperExt, err := scraper.NewScraperExtendido()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer scraperExt.Close()

	// Primero probar información básica
	fmt.Println("1. Probando información básica...")
	info, err := scraperExt.ScrapeRUC(ruc)
	if err != nil {
		log.Fatal("Error en información básica:", err)
	}
	fmt.Printf("✓ Razón Social: %s\n", info.RazonSocial)
	fmt.Printf("✓ Estado: %s\n", info.Estado)

	// Ahora probar consultas individuales
	fmt.Println("\n2. Probando consultas adicionales individuales...")
	
	// Deuda Coactiva
	fmt.Print("   - Deuda Coactiva: ")
	if deuda, err := scraperExt.ScrapeDeudaCoactiva(ruc); err == nil {
		fmt.Printf("✓ (Documentos: %d)\n", deuda.CantidadDocumentos)
	} else {
		fmt.Printf("✗ Error: %v\n", err)
	}

	// Representantes Legales
	fmt.Print("   - Representantes Legales: ")
	if reps, err := scraperExt.ScrapeRepresentantesLegales(ruc); err == nil {
		fmt.Printf("✓ (Encontrados: %d)\n", len(reps.Representantes))
	} else {
		fmt.Printf("✗ Error: %v\n", err)
	}

	fmt.Println("\n✅ Test completado")
}
EOF

# Ejecutar test
go run test_extendido.go

# Limpiar
rm test_extendido.go