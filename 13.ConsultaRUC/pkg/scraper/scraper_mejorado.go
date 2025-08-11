package scraper

import (
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// ConfigScraper contiene la configuración del scraper
type ConfigScraper struct {
	Headless       bool
	Timeout        time.Duration
	MaxReintentos  int
	DelayReintentos time.Duration
}

// DefaultConfig retorna la configuración por defecto
func DefaultConfig() ConfigScraper {
	return ConfigScraper{
		Headless:        false,
		Timeout:         30 * time.Second,
		MaxReintentos:   3,
		DelayReintentos: 2 * time.Second,
	}
}

// SUNATScraperMejorado es una versión mejorada del scraper con reintentos y mejor manejo de errores
type SUNATScraperMejorado struct {
	browser *rod.Browser
	baseURL string
	config  ConfigScraper
}

// NewSUNATScraperMejorado crea una nueva instancia del scraper mejorado
func NewSUNATScraperMejorado(config ConfigScraper) (*SUNATScraperMejorado, error) {
	l := launcher.New().
		Headless(config.Headless).
		Devtools(false)

	url := l.MustLaunch()
	browser := rod.New().
		ControlURL(url).
		MustConnect().
		MustIgnoreCertErrors(true)

	return &SUNATScraperMejorado{
		browser: browser,
		baseURL: "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp",
		config:  config,
	}, nil
}

// Close cierra el navegador
func (s *SUNATScraperMejorado) Close() {
	if s.browser != nil {
		s.browser.MustClose()
	}
}

// buscarRUC realiza la búsqueda inicial del RUC
func (s *SUNATScraperMejorado) buscarRUC(page *rod.Page, ruc string) error {
	// Esperar a que la página cargue
	err := page.WaitLoad()
	if err != nil {
		return fmt.Errorf("error esperando carga de página: %v", err)
	}

	// Buscar el input del RUC con reintentos
	var rucInput *rod.Element
	for i := 0; i < s.config.MaxReintentos; i++ {
		rucInput, err = page.Element("#txtRuc")
		if err == nil && rucInput != nil {
			break
		}
		time.Sleep(s.config.DelayReintentos)
	}
	
	if rucInput == nil {
		return fmt.Errorf("no se pudo encontrar el campo de RUC")
	}

	// Limpiar e ingresar el RUC
	err = rucInput.Input(ruc)
	if err != nil {
		return fmt.Errorf("error ingresando RUC: %v", err)
	}

	// Buscar y hacer clic en el botón de búsqueda
	searchBtn, err := page.Element("#btnAceptar")
	if err != nil {
		return fmt.Errorf("no se pudo encontrar el botón de búsqueda: %v", err)
	}

	searchBtn.MustClick()

	// Esperar un momento para que carguen los resultados
	time.Sleep(3 * time.Second)

	// Verificar si hubo algún error en la búsqueda
	errorElems, _ := page.Elements(".alert-danger")
	if len(errorElems) > 0 {
		errorText, _ := errorElems[0].Text()
		return fmt.Errorf("error en la búsqueda: %s", errorText)
	}

	return nil
}

// navegarAConsulta navega a una consulta específica haciendo clic en el botón correspondiente
func (s *SUNATScraperMejorado) navegarAConsulta(page *rod.Page, ruc string, btnClass string) (*rod.Page, error) {
	// Primero hacer la búsqueda del RUC
	err := s.buscarRUC(page, ruc)
	if err != nil {
		return nil, err
	}

	// Buscar el botón específico
	btnXPath := fmt.Sprintf("//button[contains(@class, '%s')]", btnClass)
	btn, err := page.ElementX(btnXPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo encontrar el botón %s: %v", btnClass, err)
	}

	// Hacer clic en el botón
	btn.MustClick()

	// Esperar un momento
	time.Sleep(3 * time.Second)

	// Verificar si se abrió una nueva ventana/pestaña
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		newPage := pages[len(pages)-1]
		err = newPage.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error cargando nueva página: %v", err)
		}
		return newPage, nil
	}

	return page, nil
}

// ejecutarConReintentos ejecuta una función con reintentos
func (s *SUNATScraperMejorado) ejecutarConReintentos(fn func() error) error {
	var lastErr error
	
	for i := 0; i < s.config.MaxReintentos; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		
		lastErr = err
		if i < s.config.MaxReintentos-1 {
			time.Sleep(s.config.DelayReintentos)
		}
	}
	
	return fmt.Errorf("fallo después de %d intentos: %v", s.config.MaxReintentos, lastErr)
}