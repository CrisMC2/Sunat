package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func procesarRUCConReintentos(ruc string, outputDir string, maxReintentos int) error {
	var ultimoError error
	
	for intento := 1; intento <= maxReintentos; intento++ {
		fmt.Printf("\n   Intento %d/%d...\n", intento, maxReintentos)
		
		// Crear nuevo scraper para cada intento
		s, err := scraper.NewScraperOptimizado()
		if err != nil {
			ultimoError = err
			fmt.Printf("   ❌ Error creando scraper: %v\n", err)
			if intento < maxReintentos {
				fmt.Printf("   ⏳ Reintentando en 10 segundos...\n")
				time.Sleep(10 * time.Second)
			}
			continue
		}
		
		// Procesar con manejo de panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					ultimoError = fmt.Errorf("panic recuperado: %v", r)
					fmt.Printf("   ⚠️  Error durante procesamiento: %v\n", r)
				}
				s.Close()
			}()
			
			// Procesar RUC
			resultado, err := s.ScrapeRUCCompleto(ruc)
			if err != nil {
				ultimoError = err
				return
			}
			
			// Guardar resultado
			filename := filepath.Join(outputDir, fmt.Sprintf("ruc_%s_completo.json", ruc))
			jsonData, _ := json.MarshalIndent(resultado, "", "  ")
			if err := os.WriteFile(filename, jsonData, 0644); err != nil {
				ultimoError = err
				return
			}
			
			// Éxito
			fmt.Printf("\n   ✅ PROCESADO EXITOSAMENTE\n")
			fmt.Printf("   📋 Razón Social: %s\n", resultado.InformacionBasica.RazonSocial)
			fmt.Printf("   📋 Estado: %s - %s\n", resultado.InformacionBasica.Estado, resultado.InformacionBasica.Condicion)
			fmt.Printf("   💾 Guardado en: %s\n", filename)
			
			// Mostrar resumen de información obtenida
			mostrarResumenInfo(resultado)
			
			ultimoError = nil
		}()
		
		if ultimoError == nil {
			return nil // Éxito
		}
		
		if intento < maxReintentos {
			fmt.Printf("   ⏳ Reintentando en 15 segundos...\n")
			time.Sleep(15 * time.Second)
		}
	}
	
	return ultimoError
}

func mostrarResumenInfo(resultado *models.RUCCompleto) {
	fmt.Printf("\n   📊 Información Adicional Obtenida:\n")
	
	if resultado.InformacionHistorica != nil {
		fmt.Printf("      ✓ Información histórica\n")
	}
	if resultado.DeudaCoactiva != nil {
		if resultado.DeudaCoactiva.TotalDeuda > 0 {
			fmt.Printf("      ✓ Deuda coactiva: S/ %.2f\n", resultado.DeudaCoactiva.TotalDeuda)
		} else {
			fmt.Printf("      ✓ Deuda coactiva: SIN DEUDA\n")
		}
	}
	if resultado.OmisionesTributarias != nil {
		if resultado.OmisionesTributarias.TieneOmisiones {
			fmt.Printf("      ✓ Omisiones tributarias: %d\n", resultado.OmisionesTributarias.CantidadOmisiones)
		} else {
			fmt.Printf("      ✓ Omisiones tributarias: SIN OMISIONES\n")
		}
	}
	if resultado.CantidadTrabajadores != nil {
		fmt.Printf("      ✓ Cantidad de trabajadores consultada\n")
	}
	if resultado.ActasProbatorias != nil {
		if resultado.ActasProbatorias.TieneActas {
			fmt.Printf("      ✓ Actas probatorias: %d\n", resultado.ActasProbatorias.CantidadActas)
		} else {
			fmt.Printf("      ✓ Actas probatorias: SIN ACTAS\n")
		}
	}
	if resultado.FacturasFisicas != nil {
		fmt.Printf("      ✓ Facturas físicas consultadas\n")
	}
	
	// Solo para personas jurídicas
	if strings.HasPrefix(resultado.InformacionBasica.RUC, "20") {
		if resultado.ReactivaPeru != nil {
			if resultado.ReactivaPeru.ParticipaProgramma {
				fmt.Printf("      ✓ Reactiva Perú: PARTICIPA\n")
			} else {
				fmt.Printf("      ✓ Reactiva Perú: NO PARTICIPA\n")
			}
		}
		if resultado.ProgramaCovid19 != nil {
			if resultado.ProgramaCovid19.ParticipaProgramma {
				fmt.Printf("      ✓ Programa COVID-19: PARTICIPA\n")
			} else {
				fmt.Printf("      ✓ Programa COVID-19: NO PARTICIPA\n")
			}
		}
		if resultado.RepresentantesLegales != nil && len(resultado.RepresentantesLegales.Representantes) > 0 {
			fmt.Printf("      ✓ Representantes legales: %d\n", len(resultado.RepresentantesLegales.Representantes))
		}
		if resultado.EstablecimientosAnexos != nil {
			fmt.Printf("      ✓ Establecimientos anexos: %d\n", resultado.EstablecimientosAnexos.CantidadAnexos)
		}
	}
}

func main() {
	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Println("║   PROCESAMIENTO DE 5 RUCs ORIGINALES - SUNAT   ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	
	// Crear directorio de salida
	outputDir := "data_completa"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatal("Error creando directorio de salida:", err)
	}
	
	// RUCs específicos a procesar
	rucs := []string{
		"20606642131", // FUCE & CIA E.I.R.L.
		"20613467999", // GRUPO MARTINEZ & CONTADORES S.R.L.
		"10719706288", // MEDINA MALDONADO BLANCA NELIDA
		"10775397131", // PEREZ DIAZ LIDIA
		"10420242986", // MALCA PEREZ ELIAS
	}
	
	fmt.Printf("\n📊 RESUMEN DE PROCESAMIENTO\n")
	fmt.Printf("Total de RUCs a procesar: %d\n", len(rucs))
	fmt.Printf("- Personas Jurídicas (20xxx): 2\n")
	fmt.Printf("- Personas Naturales (10xxx): 3\n")
	fmt.Printf("- Directorio de salida: %s/\n", outputDir)
	fmt.Printf("- Reintentos por RUC: 3\n")

	// Procesar RUCs
	exitosos := 0
	errores := 0
	tiempoInicio := time.Now()
	
	for i, ruc := range rucs {
		fmt.Printf("\n" + strings.Repeat("=", 60))
		fmt.Printf("\n[%d/%d] RUC: %s", i+1, len(rucs), ruc)
		
		tipo := "Persona Natural"
		if strings.HasPrefix(ruc, "20") {
			tipo = "Persona Jurídica"
		}
		fmt.Printf(" (%s)\n", tipo)
		fmt.Println(strings.Repeat("=", 60))
		
		// Procesar con reintentos
		err := procesarRUCConReintentos(ruc, outputDir, 3)
		if err != nil {
			fmt.Printf("\n   ❌ ERROR FINAL: %v\n", err)
			errores++
		} else {
			exitosos++
		}
		
		// Pausa entre RUCs
		if i < len(rucs)-1 {
			fmt.Printf("\n⏳ Esperando 10 segundos antes del siguiente RUC...\n")
			time.Sleep(10 * time.Second)
		}
	}

	// Resumen final
	tiempoTotal := time.Since(tiempoInicio)
	fmt.Printf("\n\n" + strings.Repeat("=", 60))
	fmt.Println("\n╔════════════════════════════════════════════════╗")
	fmt.Println("║                 RESUMEN FINAL                   ║")
	fmt.Println("╚════════════════════════════════════════════════╝")
	fmt.Printf("✅ Exitosos: %d/%d RUCs\n", exitosos, len(rucs))
	fmt.Printf("❌ Errores: %d/%d RUCs\n", errores, len(rucs))
	fmt.Printf("⏱️  Tiempo total: %v\n", tiempoTotal.Round(time.Second))
	fmt.Printf("📁 Archivos guardados en: %s/\n\n", outputDir)
	
	// Listar archivos generados
	fmt.Println("📋 Archivos generados:")
	files, _ := os.ReadDir(outputDir)
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "ruc_") && strings.HasSuffix(file.Name(), "_completo.json") {
			fmt.Printf("   ✓ %s\n", file.Name())
		}
	}
	
	// Generar resumen consolidado
	timestamp := time.Now().Format("20060102_150405")
	resumenFile := filepath.Join(outputDir, fmt.Sprintf("resumen_5_rucs_%s.json", timestamp))
	
	resumen := map[string]interface{}{
		"fecha_procesamiento": time.Now(),
		"rucs_procesados": rucs,
		"total_rucs": len(rucs),
		"exitosos": exitosos,
		"errores": errores,
		"tiempo_total": tiempoTotal.String(),
		"reintentos_por_ruc": 3,
	}
	
	jsonData, _ := json.MarshalIndent(resumen, "", "  ")
	os.WriteFile(resumenFile, jsonData, 0644)
	fmt.Printf("\n📊 Resumen guardado en: %s\n", resumenFile)
}