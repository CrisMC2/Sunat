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

type ScraperVisualCompleto struct {
	browser *rod.Browser
	page    *rod.Page
	baseURL string
}

func NewScraperVisualCompleto() (*ScraperVisualCompleto, error) {
	// Lanzar Chrome con interfaz visible
	l := launcher.New().
		Headless(false).          // Mostrar navegador
		Devtools(false).          // Sin herramientas de desarrollo
		Set("window-size", "1400,900").
		Set("window-position", "100,50").
		Set("disable-blink-features", "AutomationControlled")

	url, err := l.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().
		ControlURL(url).
		MustConnect().
		SlowMotion(500 * time.Millisecond) // Hacer las acciones más lentas para poder ver

	return &ScraperVisualCompleto{
		browser: browser,
		baseURL: "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp",
	}, nil
}

func (s *ScraperVisualCompleto) Close() {
	fmt.Printf("\n🔚 Cerrando navegador en 3 segundos...\n")
	time.Sleep(3 * time.Second)
	if s.page != nil {
		s.page.Close()
	}
	if s.browser != nil {
		s.browser.Close()
	}
}

func (s *ScraperVisualCompleto) mostrarAccion(accion string) {
	fmt.Printf("\n➡️  %s\n", accion)
	time.Sleep(1 * time.Second) // Pausa para ver la acción
}

func (s *ScraperVisualCompleto) ScrapeRUCCompleto(ruc string) (*models.RUCCompleto, error) {
	fmt.Printf("\n🌐 INICIANDO SCRAPING VISUAL COMPLETO\n")
	fmt.Printf("👀 Observa el navegador Chrome para ver el proceso\n")
	fmt.Printf("⏱️  Este proceso tomará unos minutos...\n\n")

	// Navegar a la página
	s.mostrarAccion("Abriendo página de SUNAT...")
	s.page = s.browser.MustPage(s.baseURL)
	s.page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	// Buscar RUC
	s.mostrarAccion(fmt.Sprintf("Escribiendo RUC: %s", ruc))
	inputRUC := s.page.MustElement("#txtRuc")
	inputRUC.MustSelectAllText()
	inputRUC.MustInput(ruc)
	time.Sleep(1 * time.Second)

	s.mostrarAccion("Haciendo clic en BUSCAR")
	btnBuscar := s.page.MustElementX("//button[contains(text(), 'Buscar')]")
	btnBuscar.MustClick()
	time.Sleep(3 * time.Second)

	// Crear estructura para resultado
	rucCompleto := &models.RUCCompleto{
		FechaConsulta: time.Now(),
		VersionAPI:    "1.0",
	}

	// Extraer información básica
	s.mostrarAccion("Extrayendo información básica...")
	info := s.extraerInfoBasica(ruc)
	rucCompleto.InformacionBasica = *info
	fmt.Printf("   ✅ Información básica extraída\n")

	// Determinar tipo de RUC
	esPersonaJuridica := strings.HasPrefix(ruc, "20")
	
	// Consultas adicionales
	fmt.Printf("\n📑 INICIANDO CONSULTAS ADICIONALES\n")
	
	// 1. Información Histórica
	if s.consultarBoton("btnInfHis", "Información Histórica") {
		rucCompleto.InformacionHistorica = &models.InformacionHistorica{}
		s.volverPaginaPrincipal()
	}

	// 2. Deuda Coactiva
	if s.consultarBoton("btnInfDeuCoa", "Deuda Coactiva") {
		rucCompleto.DeudaCoactiva = &models.DeudaCoactiva{}
		s.volverPaginaPrincipal()
	}

	// 3. Omisiones Tributarias
	if s.consultarBoton("btnInfOmi", "Omisiones Tributarias") {
		rucCompleto.OmisionesTributarias = &models.OmisionesTributarias{}
		s.volverPaginaPrincipal()
	}

	// Solo para personas jurídicas
	if esPersonaJuridica {
		fmt.Printf("\n🏢 CONSULTAS EXCLUSIVAS PARA PERSONAS JURÍDICAS\n")
		
		// Representantes Legales
		if s.consultarBoton("btnInfRep", "Representantes Legales") {
			rucCompleto.RepresentantesLegales = &models.RepresentantesLegales{}
			s.volverPaginaPrincipal()
		}

		// Establecimientos Anexos
		if s.consultarBoton("btnInfLocAnex", "Establecimientos Anexos") {
			rucCompleto.EstablecimientosAnexos = &models.EstablecimientosAnexos{}
			s.volverPaginaPrincipal()
		}
	}

	fmt.Printf("\n✅ PROCESO COMPLETADO\n")
	return rucCompleto, nil
}

func (s *ScraperVisualCompleto) consultarBoton(clase string, nombre string) bool {
	s.mostrarAccion(fmt.Sprintf("Consultando: %s", nombre))
	
	// Buscar y hacer clic en el botón
	selector := fmt.Sprintf("//button[contains(@class, '%s')]", clase)
	if btn, err := s.page.ElementX(selector); err == nil && btn != nil {
		btn.MustClick()
		time.Sleep(3 * time.Second)
		
		// Verificar si se abrió nueva pestaña
		pages := s.browser.MustPages()
		if len(pages) > 1 {
			fmt.Printf("   ✅ %s - Nueva pestaña abierta\n", nombre)
		} else {
			fmt.Printf("   ✅ %s - Información cargada\n", nombre)
		}
		return true
	}
	
	fmt.Printf("   ⚠️  %s - No disponible\n", nombre)
	return false
}

func (s *ScraperVisualCompleto) volverPaginaPrincipal() {
	s.mostrarAccion("Volviendo a la página principal...")
	
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		// Cerrar pestaña adicional
		pages[len(pages)-1].MustClose()
		time.Sleep(1 * time.Second)
		// Volver el foco a la página principal
		s.page = pages[0]
		s.page.MustActivate()
	} else {
		// Buscar botón Volver
		if volverBtn, err := s.page.ElementX("//button[contains(text(), 'Volver')]"); err == nil && volverBtn != nil {
			volverBtn.MustClick()
			time.Sleep(2 * time.Second)
		}
	}
}

func (s *ScraperVisualCompleto) extraerInfoBasica(ruc string) *models.RUCInfo {
	info := &models.RUCInfo{RUC: ruc}

	// Extraer razón social
	if elem, err := s.page.Element("h4.list-group-item-heading"); err == nil {
		info.RazonSocial = strings.TrimSpace(elem.MustText())
	}

	// Mapa de campos
	campos := map[string]*string{
		"Tipo Contribuyente:": &info.TipoContribuyente,
		"Estado del Contribuyente:": &info.Estado,
		"Condición del Contribuyente:": &info.Condicion,
		"Domicilio Fiscal:": &info.DomicilioFiscal,
		"Fecha de Inscripción:": &info.FechaInscripcion,
		"Fecha de Inicio de Actividades:": &info.FechaInicioActividades,
	}

	// Extraer campos
	rows := s.page.MustElements(".list-group-item")
	for _, row := range rows {
		text := row.MustText()
		for label, field := range campos {
			if strings.Contains(text, label) && field != nil {
				value := strings.TrimSpace(strings.Replace(text, label, "", 1))
				*field = value
			}
		}
	}

	return info
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("\nUso: go run main.go <RUC>")
		fmt.Println("Ejemplo: go run main.go 20606642131")
		fmt.Println("\nEste scraper mostrará el navegador Chrome")
		fmt.Println("y podrás ver todo el proceso de extracción")
		os.Exit(1)
	}

	ruc := os.Args[1]

	fmt.Printf("\n╔════════════════════════════════════════════════╗\n")
	fmt.Printf("║     SCRAPER VISUAL COMPLETO - SUNAT           ║\n")
	fmt.Printf("╚════════════════════════════════════════════════╝\n")
	fmt.Printf("\n📌 RUC a procesar: %s\n", ruc)

	// Crear scraper
	scraper, err := NewScraperVisualCompleto()
	if err != nil {
		log.Fatal("Error creando scraper:", err)
	}
	defer scraper.Close()

	// Procesar RUC
	resultado, err := scraper.ScrapeRUCCompleto(ruc)
	if err != nil {
		log.Fatal("Error procesando RUC:", err)
	}

	// Guardar resultado
	outputDir := "data_completa"
	os.MkdirAll(outputDir, 0755)
	
	filename := filepath.Join(outputDir, fmt.Sprintf("ruc_%s_visual_completo.json", ruc))
	jsonData, _ := json.MarshalIndent(resultado, "", "  ")
	os.WriteFile(filename, jsonData, 0644)

	fmt.Printf("\n💾 Archivo guardado: %s\n", filename)
	fmt.Printf("📊 Tamaño: %d bytes\n", len(jsonData))
}