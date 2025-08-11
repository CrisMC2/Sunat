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

func procesarRUCIndividual(ruc string, outputDir string) error {
	fmt.Printf("\n[PROCESANDO] RUC: %s\n", ruc)
	fmt.Println("=" + strings.Repeat("=", 50))
	
	// Crear nuevo scraper para cada RUC
	s, err := scraper.NewScraperOptimizado()
	if err != nil {
		return fmt.Errorf("error creando scraper: %v", err)
	}
	defer func() {
		// Asegurar cierre limpio
		if r := recover(); r != nil {
			fmt.Printf("⚠️  Recuperando de error: %v\n", r)
		}
		s.Close()
	}()

	// Procesar RUC
	resultado, err := s.ScrapeRUCCompleto(ruc)
	if err != nil {
		return fmt.Errorf("error procesando RUC: %v", err)
	}

	// Guardar resultado
	filename := filepath.Join(outputDir, fmt.Sprintf("ruc_%s_completo.json", ruc))
	jsonData, _ := json.MarshalIndent(resultado, "", "  ")
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("error guardando archivo: %v", err)
	}

	// Mostrar resumen
	fmt.Printf("\n✅ PROCESADO EXITOSAMENTE\n")
	fmt.Printf("📋 Razón Social: %s\n", resultado.InformacionBasica.RazonSocial)
	fmt.Printf("📋 Estado: %s - %s\n", resultado.InformacionBasica.Estado, resultado.InformacionBasica.Condicion)
	fmt.Printf("💾 Guardado en: %s\n", filename)
	
	// Mostrar información adicional obtenida
	fmt.Printf("\n📊 Información Adicional Obtenida:\n")
	if resultado.InformacionHistorica != nil {
		fmt.Printf("   ✓ Información histórica\n")
	}
	if resultado.DeudaCoactiva != nil {
		if resultado.DeudaCoactiva.TotalDeuda > 0 {
			fmt.Printf("   ✓ Deuda coactiva: S/ %.2f\n", resultado.DeudaCoactiva.TotalDeuda)
		} else {
			fmt.Printf("   ✓ Deuda coactiva: SIN DEUDA\n")
		}
	}
	if resultado.OmisionesTributarias != nil {
		if resultado.OmisionesTributarias.TieneOmisiones {
			fmt.Printf("   ✓ Omisiones tributarias: %d\n", resultado.OmisionesTributarias.CantidadOmisiones)
		} else {
			fmt.Printf("   ✓ Omisiones tributarias: SIN OMISIONES\n")
		}
	}
	if resultado.CantidadTrabajadores != nil {
		fmt.Printf("   ✓ Cantidad de trabajadores\n")
	}
	if resultado.ActasProbatorias != nil {
		if resultado.ActasProbatorias.TieneActas {
			fmt.Printf("   ✓ Actas probatorias: %d\n", resultado.ActasProbatorias.CantidadActas)
		} else {
			fmt.Printf("   ✓ Actas probatorias: SIN ACTAS\n")
		}
	}
	if resultado.RepresentantesLegales != nil && len(resultado.RepresentantesLegales.Representantes) > 0 {
		fmt.Printf("   ✓ Representantes legales: %d\n", len(resultado.RepresentantesLegales.Representantes))
	}
	if resultado.EstablecimientosAnexos != nil {
		fmt.Printf("   ✓ Establecimientos anexos: %d\n", resultado.EstablecimientosAnexos.CantidadAnexos)
	}

	return nil
}

func main() {
	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Println("║     PROCESAMIENTO COMPLETO DE RUCs - SUNAT     ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	
	// Crear directorio de salida
	outputDir := "data_completa"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Error creando directorio de salida:", err)
	}
	
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

	fmt.Printf("\n📊 RESUMEN DE PROCESAMIENTO\n")
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
	fmt.Printf("- Personas Naturales (10xxx): %d\n", naturales)
	fmt.Printf("- Directorio de salida: %s/\n", outputDir)

	// Procesar RUCs uno por uno
	exitosos := 0
	errores := 0
	tiempoInicio := time.Now()
	
	for i, ruc := range rucs {
		fmt.Printf("\n\n[%d/%d] ", i+1, len(rucs))
		
		tipo := "Persona Natural"
		if strings.HasPrefix(ruc, "20") {
			tipo = "Persona Jurídica"
		}
		fmt.Printf("Tipo: %s\n", tipo)
		
		// Procesar RUC
		err := procesarRUCIndividual(ruc, outputDir)
		if err != nil {
			fmt.Printf("\n❌ ERROR: %v\n", err)
			errores++
		} else {
			exitosos++
		}
		
		// Pausa entre RUCs
		if i < len(rucs)-1 {
			fmt.Printf("\n⏳ Esperando 5 segundos antes del siguiente RUC...\n")
			time.Sleep(5 * time.Second)
		}
	}

	// Resumen final
	tiempoTotal := time.Since(tiempoInicio)
	fmt.Println("\n\n╔════════════════════════════════════════════════╗")
	fmt.Println("║              RESUMEN FINAL                      ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Printf("✅ Exitosos: %d RUCs\n", exitosos)
	fmt.Printf("❌ Errores: %d RUCs\n", errores)
	fmt.Printf("⏱️  Tiempo total: %v\n", tiempoTotal.Round(time.Second))
	fmt.Printf("📁 Archivos guardados en: %s/\n", outputDir)
	
	// Generar resumen consolidado
	if exitosos > 0 {
		timestamp := time.Now().Format("20060102_150405")
		resumenFile := filepath.Join(outputDir, fmt.Sprintf("resumen_%s.json", timestamp))
		
		resumen := map[string]interface{}{
			"fecha_procesamiento": time.Now(),
			"total_rucs": len(rucs),
			"exitosos": exitosos,
			"errores": errores,
			"tiempo_total": tiempoTotal.String(),
			"rucs_procesados": rucs,
		}
		
		jsonData, _ := json.MarshalIndent(resumen, "", "  ")
		os.WriteFile(resumenFile, jsonData, 0644)
		fmt.Printf("\n📋 Resumen guardado en: %s\n", resumenFile)
	}
}