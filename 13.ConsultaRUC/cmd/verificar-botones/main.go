package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <RUC>")
		os.Exit(1)
	}

	ruc := os.Args[1]

	fmt.Printf("\n╔════════════════════════════════════════════════╗\n")
	fmt.Printf("║    VERIFICADOR DE BOTONES DISPONIBLES          ║\n")
	fmt.Printf("╚════════════════════════════════════════════════╝\n")
	fmt.Printf("\n📌 RUC: %s\n", ruc)

	// Lanzar navegador visible
	l := launcher.New().
		Headless(false).
		Set("window-size", "1400,900")

	url, err := l.Launch()
	if err != nil {
		log.Fatal(err)
	}

	browser := rod.New().
		ControlURL(url).
		MustConnect()
	defer browser.Close()

	// Navegar a SUNAT
	page := browser.MustPage("https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp")
	page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	// Buscar RUC
	fmt.Printf("\n🔍 Buscando RUC...\n")
	inputRUC := page.MustElement("#txtRuc")
	inputRUC.MustSelectAllText()
	inputRUC.MustInput(ruc)
	
	btnBuscar := page.MustElementX("//button[contains(text(), 'Buscar')]")
	btnBuscar.MustClick()
	time.Sleep(3 * time.Second)

	// Obtener información básica
	fmt.Printf("\n📋 INFORMACIÓN BÁSICA:\n")
	if elem, err := page.Element("h4.list-group-item-heading"); err == nil {
		razonSocial := strings.TrimSpace(elem.MustText())
		fmt.Printf("   Razón Social: %s\n", razonSocial)
	}

	// Lista de todos los posibles botones
	botones := []struct {
		clase       string
		nombre      string
		descripcion string
	}{
		{"btnInfHis", "Información Histórica", "Historial de cambios del contribuyente"},
		{"btnInfDeuCoa", "Deuda Coactiva", "Deudas en cobranza coactiva"},
		{"btnInfOmi", "Omisiones Tributarias", "Omisiones y multas tributarias"},
		{"btnInfTra", "Cantidad de Trabajadores", "Número de trabajadores declarados"},
		{"btnInfProb", "Actas Probatorias", "Actas de fiscalización"},
		{"btnInfFacFis", "Comprobantes Físicos", "Autorización de facturas físicas"},
		{"btnReacPeru", "Reactiva Perú", "Participación en programa Reactiva Perú"},
		{"btnCovid19", "Programa COVID-19", "Beneficios por COVID-19"},
		{"btnInfRep", "Representantes Legales", "Representantes de la empresa"},
		{"btnInfLocAnex", "Establecimientos Anexos", "Locales y sucursales"},
		{"btnEEFF", "Estados Financieros", "Estados financieros presentados"},
		{"btnPadAgen", "Agentes de Retención", "Padrón de agentes de retención"},
	}

	// Verificar qué botones están disponibles
	fmt.Printf("\n🔘 BOTONES DISPONIBLES:\n")
	botonesEncontrados := 0
	botonesNoEncontrados := []string{}

	for _, boton := range botones {
		selector := fmt.Sprintf("//button[contains(@class, '%s')]", boton.clase)
		
		// Intentar encontrar el botón
		if elem, err := page.Timeout(2 * time.Second).ElementX(selector); err == nil && elem != nil {
			botonesEncontrados++
			fmt.Printf("   ✅ %s - %s\n", boton.nombre, boton.descripcion)
			
			// Verificar si está habilitado
			disabled, _ := elem.Property("disabled")
			if disabled.Bool() {
				fmt.Printf("      ⚠️  (Deshabilitado)\n")
			}
		} else {
			botonesNoEncontrados = append(botonesNoEncontrados, boton.nombre)
		}
	}

	// Mostrar botones no encontrados
	if len(botonesNoEncontrados) > 0 {
		fmt.Printf("\n❌ BOTONES NO DISPONIBLES:\n")
		for _, nombre := range botonesNoEncontrados {
			fmt.Printf("   • %s\n", nombre)
		}
	}

	// Resumen
	fmt.Printf("\n📊 RESUMEN:\n")
	fmt.Printf("   Total de botones disponibles: %d\n", botonesEncontrados)
	fmt.Printf("   Total de botones no disponibles: %d\n", len(botonesNoEncontrados))
	
	tipo := "Persona Natural"
	if strings.HasPrefix(ruc, "20") {
		tipo = "Persona Jurídica"
	}
	fmt.Printf("   Tipo de contribuyente: %s\n", tipo)

	fmt.Printf("\n⏸️  Manteniendo navegador abierto por 10 segundos...\n")
	time.Sleep(10 * time.Second)
}