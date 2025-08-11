package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/consulta-ruc-scraper/pkg/models"
)

type ScraperVisual struct {
	browser *rod.Browser
	page    *rod.Page
	baseURL string
}

func NewScraperVisual() (*ScraperVisual, error) {
	// Lanzar Chrome con interfaz visible
	l := launcher.New().
		Headless(false).          // Mostrar navegador
		Devtools(false).          // Sin herramientas de desarrollo
		Set("window-size", "1400,900").
		Set("disable-blink-features", "AutomationControlled")

	url, err := l.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().
		ControlURL(url).
		MustConnect().
		SlowMotion(300 * time.Millisecond) // Hacer las acciones más lentas para poder ver

	return &ScraperVisual{
		browser: browser,
		baseURL: "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp",
	}, nil
}

func (s *ScraperVisual) Close() {
	if s.page != nil {
		s.page.Close()
	}
	if s.browser != nil {
		time.Sleep(2 * time.Second) // Esperar antes de cerrar para ver resultado
		s.browser.Close()
	}
}

func (s *ScraperVisual) ScrapeRUC(ruc string) (*models.RUCInfo, error) {
	fmt.Printf("\n🌐 Abriendo navegador Chrome...\n")
	fmt.Printf("👀 Podrás ver todo el proceso de scraping\n\n")

	// Navegar a la página
	fmt.Printf("📍 Navegando a SUNAT...\n")
	s.page = s.browser.MustPage(s.baseURL)
	s.page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	// Buscar el campo de RUC
	fmt.Printf("🔍 Buscando campo de RUC...\n")
	inputRUC := s.page.MustElement("#txtRuc")
	
	// Limpiar y escribir RUC
	fmt.Printf("✍️  Escribiendo RUC: %s\n", ruc)
	inputRUC.MustSelectAllText()
	inputRUC.MustInput(ruc)
	time.Sleep(1 * time.Second)

	// Hacer clic en buscar
	fmt.Printf("🖱️  Haciendo clic en buscar...\n")
	btnBuscar := s.page.MustElementX("//button[contains(text(), 'Buscar')]")
	btnBuscar.MustClick()
	
	// Esperar resultados
	fmt.Printf("⏳ Esperando resultados...\n")
	time.Sleep(3 * time.Second)

	// Extraer información
	fmt.Printf("📊 Extrayendo información...\n")
	info := &models.RUCInfo{
		RUC: ruc,
	}

	// Extraer cada campo
	campos := map[string]*string{
		"Número de RUC:": &info.RUC,
		"Tipo Contribuyente:": &info.TipoContribuyente,
		"Nombre Comercial:": &info.NombreComercial,
		"Fecha de Inscripción:": &info.FechaInscripcion,
		"Fecha de Inicio de Actividades:": &info.FechaInicioActividades,
		"Estado del Contribuyente:": &info.Estado,
		"Condición del Contribuyente:": &info.Condicion,
		"Domicilio Fiscal:": &info.DomicilioFiscal,
		"Sistema Emisión de Comprobante:": &info.SistemaEmision,
		"Actividad Comercio Exterior:": &info.ActividadComercioExterior,
		"Sistema Contabilidiad:": &info.SistemaContabilidad,
		"Emisor electrónico desde:": &info.EmisorElectronicoDesde,
		"Afiliado PLE desde:": &info.AfiliadoPLE,
	}

	// Buscar razón social
	if elem, err := s.page.Element("h4.list-group-item-heading"); err == nil {
		info.RazonSocial = strings.TrimSpace(elem.MustText())
		fmt.Printf("   ✓ Razón Social: %s\n", info.RazonSocial)
	}

	// Extraer otros campos
	rows := s.page.MustElements(".list-group-item")
	for _, row := range rows {
		text := row.MustText()
		for label, field := range campos {
			if strings.Contains(text, label) && field != nil {
				value := strings.TrimSpace(strings.Replace(text, label, "", 1))
				*field = value
				fmt.Printf("   ✓ %s %s\n", label, value)
			}
		}
	}

	fmt.Printf("\n✅ Información extraída exitosamente\n")
	fmt.Printf("⏸️  Manteniendo navegador abierto por 5 segundos...\n")
	time.Sleep(5 * time.Second)

	return info, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <RUC>")
		fmt.Println("Ejemplo: go run main.go 20606642131")
		os.Exit(1)
	}

	ruc := os.Args[1]

	fmt.Printf("\n╔════════════════════════════════════════════════╗\n")
	fmt.Printf("║   SCRAPER VISUAL - RUC: %s          ║\n", ruc)
	fmt.Printf("╚════════════════════════════════════════════════╝\n")

	// Crear scraper visual
	scraper, err := NewScraperVisual()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer scraper.Close()

	// Procesar RUC
	info, err := scraper.ScrapeRUC(ruc)
	if err != nil {
		log.Fatal("Error procesando RUC:", err)
	}

	// Guardar resultado
	outputDir := "data_completa"
	os.MkdirAll(outputDir, 0755)
	
	filename := filepath.Join(outputDir, fmt.Sprintf("ruc_%s_visual.json", ruc))
	jsonData, _ := json.MarshalIndent(info, "", "  ")
	os.WriteFile(filename, jsonData, 0644)

	fmt.Printf("\n💾 Guardado en: %s\n", filename)
	fmt.Printf("🏁 Proceso completado\n")
}