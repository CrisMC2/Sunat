package scraper

import (
	"fmt"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/utils"
	"github.com/go-rod/rod"
)

// ScraperExtendido incluye métodos para todas las consultas adicionales
type ScraperExtendido struct {
	*SUNATScraper
}

// NewScraperExtendido crea una nueva instancia del scraper extendido
func NewScraperExtendido() (*ScraperExtendido, error) {
	base, err := NewSUNATScraper()
	if err != nil {
		return nil, err
	}
	
	return &ScraperExtendido{
		SUNATScraper: base,
	}, nil
}

// ScrapeRUCCompleto obtiene toda la información disponible de un RUC
func (s *ScraperExtendido) ScrapeRUCCompleto(ruc string) (*models.RUCCompleto, error) {
	// Primero obtener información básica
	infoBasica, err := s.ScrapeRUC(ruc)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo información básica: %v", err)
	}

	// Crear estructura completa
	rucCompleto := &models.RUCCompleto{
		InformacionBasica: *infoBasica,
		FechaConsulta:     time.Now(),
		VersionAPI:        "1.0.0",
	}

	// Determinar qué consultas están disponibles según el tipo de RUC
	esPersonaJuridica := strings.HasPrefix(ruc, "20")
	
	fmt.Printf("  ℹ️  RUC %s es: %s\n", ruc, map[bool]string{true: "Persona Jurídica", false: "Persona Natural"}[esPersonaJuridica])
	
	// Consultas disponibles para todos los tipos de RUC
	fmt.Println("  📋 Consultando información adicional...")
	
	// Información Histórica
	fmt.Print("    - Información Histórica: ")
	if infoHist, err := s.ScrapeInformacionHistorica(ruc); err == nil {
		rucCompleto.InformacionHistorica = infoHist
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Deuda Coactiva
	fmt.Print("    - Deuda Coactiva: ")
	if deuda, err := s.ScrapeDeudaCoactiva(ruc); err == nil {
		rucCompleto.DeudaCoactiva = deuda
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Omisiones Tributarias
	fmt.Print("    - Omisiones Tributarias: ")
	if omis, err := s.ScrapeOmisionesTributarias(ruc); err == nil {
		rucCompleto.OmisionesTributarias = omis
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Cantidad de Trabajadores
	fmt.Print("    - Cantidad de Trabajadores: ")
	if trab, err := s.ScrapeCantidadTrabajadores(ruc); err == nil {
		rucCompleto.CantidadTrabajadores = trab
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Actas Probatorias
	fmt.Print("    - Actas Probatorias: ")
	if actas, err := s.ScrapeActasProbatorias(ruc); err == nil {
		rucCompleto.ActasProbatorias = actas
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Facturas Físicas
	fmt.Print("    - Facturas Físicas: ")
	if fact, err := s.ScrapeFacturasFisicas(ruc); err == nil {
		rucCompleto.FacturasFisicas = fact
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Consultas solo disponibles para Personas Jurídicas
	if esPersonaJuridica {
		fmt.Println("  📋 Consultando información exclusiva de Personas Jurídicas...")
		
		// Representantes Legales
		fmt.Print("    - Representantes Legales: ")
		if reps, err := s.ScrapeRepresentantesLegales(ruc); err == nil {
			rucCompleto.RepresentantesLegales = reps
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}

		// Establecimientos Anexos
		fmt.Print("    - Establecimientos Anexos: ")
		if estab, err := s.ScrapeEstablecimientosAnexos(ruc); err == nil {
			rucCompleto.EstablecimientosAnexos = estab
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}

		// Reactiva Perú
		fmt.Print("    - Reactiva Perú: ")
		if react, err := s.ScrapeReactivaPeru(ruc); err == nil {
			rucCompleto.ReactivaPeru = react
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}

		// Programa COVID-19
		fmt.Print("    - Programa COVID-19: ")
		if covid, err := s.ScrapeProgramaCovid19(ruc); err == nil {
			rucCompleto.ProgramaCovid19 = covid
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}
	}

	return rucCompleto, nil
}

// ScrapeInformacionHistorica obtiene la información histórica del RUC
func (s *ScraperExtendido) ScrapeInformacionHistorica(ruc string) (*models.InformacionHistorica, error) {
	page := s.browser.MustPage(s.baseURL)
	defer page.MustClose()

	// Primero hacer la búsqueda del RUC
	page.MustWaitLoad()
	page.MustWaitStable()
	
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)
	
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()
	
	time.Sleep(5 * time.Second)

	// Buscar y hacer clic en el botón de Información Histórica
	histBtn := page.MustElementX("//button[contains(@class, 'btnInfHis')]")
	histBtn.MustClick()

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva ventana/pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		page.MustWaitLoad()
	}

	info := &models.InformacionHistorica{
		RUC: ruc,
	}

	// Extraer información histórica
	s.extractHistoricalInfo(page, info)

	return info, nil
}

// extractHistoricalInfo extrae la información histórica de la página
func (s *ScraperExtendido) extractHistoricalInfo(page *rod.Page, info *models.InformacionHistorica) {
	// Buscar tablas con información histórica
	tables := page.MustElements("table")
	
	for _, table := range tables {
		// Identificar el tipo de tabla por sus encabezados
		headers := table.MustElements("th")
		if len(headers) > 0 {
			headerText := strings.ToLower(headers[0].MustText())
			
			rows := table.MustElements("tbody tr")
			
			for _, row := range rows {
				cells := row.MustElements("td")
				if len(cells) >= 3 {
					cambio := models.CambioHistorico{
						Fecha:         strings.TrimSpace(cells[0].MustText()),
						ValorAnterior: strings.TrimSpace(cells[1].MustText()),
						ValorNuevo:    strings.TrimSpace(cells[2].MustText()),
					}
					
					if len(cells) >= 4 {
						cambio.Motivo = strings.TrimSpace(cells[3].MustText())
					}
					
					if strings.Contains(headerText, "razón social") || strings.Contains(headerText, "razon social") {
						info.CambiosRazonSocial = append(info.CambiosRazonSocial, cambio)
					} else if strings.Contains(headerText, "domicilio") {
						info.CambiosDomicilio = append(info.CambiosDomicilio, cambio)
					}
				}
			}
		}
	}
}

// ScrapeDeudaCoactiva obtiene información de deuda coactiva
func (s *ScraperExtendido) ScrapeDeudaCoactiva(ruc string) (*models.DeudaCoactiva, error) {
	page := s.browser.MustPage(s.baseURL)
	defer page.MustClose()

	// Hacer la búsqueda del RUC
	page.MustWaitLoad()
	page.MustWaitStable()
	
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)
	
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()
	
	time.Sleep(5 * time.Second)

	// Buscar y hacer clic en el botón de Deuda Coactiva
	deudaBtn := page.MustElementX("//button[contains(@class, 'btnInfDeuCoa')]")
	deudaBtn.MustClick()

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva ventana/pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		page.MustWaitLoad()
	}

	deuda := &models.DeudaCoactiva{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}

	// Extraer información de deuda
	s.extractDeudaInfo(page, deuda)

	return deuda, nil
}

// extractDeudaInfo extrae la información de deuda coactiva
func (s *ScraperExtendido) extractDeudaInfo(page *rod.Page, deuda *models.DeudaCoactiva) {
	// Buscar mensaje de "no tiene deuda" o tabla con deudas
	noDeudaElem := page.MustElements(".alert-info")
	if len(noDeudaElem) > 0 && strings.Contains(noDeudaElem[0].MustText(), "no registra") {
		deuda.TotalDeuda = 0
		deuda.CantidadDocumentos = 0
		return
	}

	// Buscar tabla de deudas
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")
		
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 7 {
				detalle := models.DetalleDeuda{
					NumeroDocumento:  strings.TrimSpace(cells[0].MustText()),
					TipoDocumento:    strings.TrimSpace(cells[1].MustText()),
					FechaEmision:     strings.TrimSpace(cells[2].MustText()),
					Periodo:          strings.TrimSpace(cells[3].MustText()),
					Tributo:          strings.TrimSpace(cells[4].MustText()),
				}
				
				// Parsear montos
				detalle.MontoOriginal = utils.ParseMonto(cells[5].MustText())
				detalle.MontoActualizado = utils.ParseMonto(cells[6].MustText())
				
				deuda.Deudas = append(deuda.Deudas, detalle)
			}
		}
		
		deuda.CantidadDocumentos = len(deuda.Deudas)
		// Calcular total de deuda
		for _, d := range deuda.Deudas {
			deuda.TotalDeuda += d.MontoActualizado
		}
	}
}

// ScrapeRepresentantesLegales obtiene los representantes legales
func (s *ScraperExtendido) ScrapeRepresentantesLegales(ruc string) (*models.RepresentantesLegales, error) {
	page := s.browser.MustPage(s.baseURL)
	defer page.MustClose()

	// Hacer la búsqueda del RUC
	page.MustWaitLoad()
	page.MustWaitStable()
	
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)
	
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()
	
	time.Sleep(5 * time.Second)

	// Buscar y hacer clic en el botón de Representantes Legales
	repBtn := page.MustElementX("//button[contains(@class, 'btnInfRepLeg')]")
	repBtn.MustClick()

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva ventana/pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		page.MustWaitLoad()
	}

	reps := &models.RepresentantesLegales{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}

	// Extraer información de representantes
	s.extractRepresentantesInfo(page, reps)

	return reps, nil
}

// extractRepresentantesInfo extrae la información de representantes legales
func (s *ScraperExtendido) extractRepresentantesInfo(page *rod.Page, reps *models.RepresentantesLegales) {
	// Buscar tabla de representantes
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")
		
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 6 {
				rep := models.RepresentanteLegal{
					TipoDocumento:   strings.TrimSpace(cells[0].MustText()),
					NumeroDocumento: strings.TrimSpace(cells[1].MustText()),
					Cargo:           strings.TrimSpace(cells[4].MustText()),
					FechaDesde:      strings.TrimSpace(cells[5].MustText()),
				}
				
				// Extraer nombres completos
				nombreCompleto := strings.TrimSpace(cells[2].MustText())
				partes := strings.Fields(nombreCompleto)
				if len(partes) >= 3 {
					rep.ApellidoPaterno = partes[0]
					rep.ApellidoMaterno = partes[1]
					rep.Nombres = strings.Join(partes[2:], " ")
				}
				
				// Verificar si está vigente
				if len(cells) >= 7 {
					rep.FechaHasta = strings.TrimSpace(cells[6].MustText())
					rep.Vigente = rep.FechaHasta == "" || rep.FechaHasta == "-"
				} else {
					rep.Vigente = true
				}
				
				reps.Representantes = append(reps.Representantes, rep)
			}
		}
	}
}

// ScrapeCantidadTrabajadores obtiene la cantidad de trabajadores
func (s *ScraperExtendido) ScrapeCantidadTrabajadores(ruc string) (*models.CantidadTrabajadores, error) {
	page := s.browser.MustPage(s.baseURL)
	defer page.MustClose()

	// Hacer la búsqueda del RUC
	page.MustWaitLoad()
	page.MustWaitStable()
	
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)
	
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()
	
	time.Sleep(5 * time.Second)

	// Buscar y hacer clic en el botón de Cantidad de Trabajadores
	trabBtn := page.MustElementX("//button[contains(@class, 'btnInfNumTra')]")
	trabBtn.MustClick()

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva ventana/pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		page.MustWaitLoad()
	}

	trab := &models.CantidadTrabajadores{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}

	// Extraer información de trabajadores
	s.extractTrabajadoresInfo(page, trab)

	return trab, nil
}

// extractTrabajadoresInfo extrae la información de trabajadores
func (s *ScraperExtendido) extractTrabajadoresInfo(page *rod.Page, trab *models.CantidadTrabajadores) {
	// Buscar select de periodos
	selectElem := page.MustElement("select[name='periodo']")
	options := selectElem.MustElements("option")
	
	for _, opt := range options {
		periodo := opt.MustAttribute("value")
		if periodo != nil && *periodo != "" {
			trab.PeriodosDisponibles = append(trab.PeriodosDisponibles, *periodo)
		}
	}

	// Buscar tabla con información de trabajadores
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")
		
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 5 {
				detalle := models.DetalleTrabajadores{
					Periodo: strings.TrimSpace(cells[0].MustText()),
				}
				
				// Parsear cantidades (convertir string a int)
				// TODO: Implementar conversión
				
				trab.DetallePorPeriodo = append(trab.DetallePorPeriodo, detalle)
			}
		}
	}
}

// ScrapeEstablecimientosAnexos obtiene los establecimientos anexos
func (s *ScraperExtendido) ScrapeEstablecimientosAnexos(ruc string) (*models.EstablecimientosAnexos, error) {
	page := s.browser.MustPage(s.baseURL)
	defer page.MustClose()

	// Hacer la búsqueda del RUC
	page.MustWaitLoad()
	page.MustWaitStable()
	
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)
	
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()
	
	time.Sleep(5 * time.Second)

	// Buscar y hacer clic en el botón de Establecimientos Anexos
	estabBtn := page.MustElementX("//button[contains(@class, 'btnInfLocAnex')]")
	estabBtn.MustClick()

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva ventana/pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		page.MustWaitLoad()
	}

	estab := &models.EstablecimientosAnexos{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}

	// Extraer información de establecimientos
	s.extractEstablecimientosInfo(page, estab)

	return estab, nil
}

// extractEstablecimientosInfo extrae la información de establecimientos anexos
func (s *ScraperExtendido) extractEstablecimientosInfo(page *rod.Page, estab *models.EstablecimientosAnexos) {
	// Buscar tabla de establecimientos
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")
		
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 6 {
				anexo := models.EstablecimientoAnexo{
					CodigoEstablecimiento: strings.TrimSpace(cells[0].MustText()),
					TipoEstablecimiento:   strings.TrimSpace(cells[1].MustText()),
					Direccion:             strings.TrimSpace(cells[2].MustText()),
					Estado:                strings.TrimSpace(cells[5].MustText()),
				}
				
				// Extraer ubicación
				ubicacion := strings.TrimSpace(cells[3].MustText())
				partes := strings.Split(ubicacion, " - ")
				if len(partes) >= 3 {
					anexo.Departamento = partes[0]
					anexo.Provincia = partes[1]
					anexo.Distrito = partes[2]
				}
				
				estab.Establecimientos = append(estab.Establecimientos, anexo)
			}
		}
		
		estab.CantidadAnexos = len(estab.Establecimientos)
	}
}

// Métodos adicionales para las demás consultas...

// ScrapeOmisionesTributarias obtiene las omisiones tributarias
func (s *ScraperExtendido) ScrapeOmisionesTributarias(ruc string) (*models.OmisionesTributarias, error) {
	page := s.browser.MustPage(s.baseURL)
	defer func() {
		if page != nil {
			page.Close()
		}
	}()

	// Hacer la búsqueda del RUC
	page.MustWaitLoad()
	page.MustWaitStable()
	
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)
	
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()
	
	time.Sleep(5 * time.Second)

	// Buscar y hacer clic en el botón de Omisiones Tributarias
	omisBtn := page.MustElementX("//button[contains(@class, 'btnInfOmiTri')]")
	omisBtn.MustClick()

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva ventana/pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		page.MustWaitLoad()
	}

	omis := &models.OmisionesTributarias{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}

	// Extraer información de omisiones
	s.extractOmisionesInfo(page, omis)

	return omis, nil
}

// extractOmisionesInfo extrae la información de omisiones tributarias
func (s *ScraperExtendido) extractOmisionesInfo(page *rod.Page, omis *models.OmisionesTributarias) {
	// Buscar mensaje de "no tiene omisiones" o tabla con omisiones
	noOmisionesElem := page.MustElements(".alert-info")
	if len(noOmisionesElem) > 0 && strings.Contains(noOmisionesElem[0].MustText(), "no registra") {
		omis.TieneOmisiones = false
		omis.CantidadOmisiones = 0
		return
	}

	// Buscar tabla de omisiones
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")
		
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 4 {
				omision := models.Omision{
					Periodo:          strings.TrimSpace(cells[0].MustText()),
					Tributo:          strings.TrimSpace(cells[1].MustText()),
					TipoDeclaracion:  strings.TrimSpace(cells[2].MustText()),
					FechaVencimiento: strings.TrimSpace(cells[3].MustText()),
				}
				
				if len(cells) >= 5 {
					omision.Estado = strings.TrimSpace(cells[4].MustText())
				}
				
				omis.Omisiones = append(omis.Omisiones, omision)
			}
		}
		
		omis.TieneOmisiones = len(omis.Omisiones) > 0
		omis.CantidadOmisiones = len(omis.Omisiones)
	}
}

// ScrapeActasProbatorias obtiene las actas probatorias
func (s *ScraperExtendido) ScrapeActasProbatorias(ruc string) (*models.ActasProbatorias, error) {
	// Implementación similar a las anteriores
	actas := &models.ActasProbatorias{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}
	// TODO: Implementar scraping
	return actas, nil
}

// ScrapeFacturasFisicas obtiene información de facturas físicas
func (s *ScraperExtendido) ScrapeFacturasFisicas(ruc string) (*models.FacturasFisicas, error) {
	// Implementación similar a las anteriores
	fact := &models.FacturasFisicas{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}
	// TODO: Implementar scraping
	return fact, nil
}

// ScrapeReactivaPeru obtiene información del programa Reactiva Perú
func (s *ScraperExtendido) ScrapeReactivaPeru(ruc string) (*models.ReactivaPeru, error) {
	// Implementación similar a las anteriores
	react := &models.ReactivaPeru{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}
	// TODO: Implementar scraping
	return react, nil
}

// ScrapeProgramaCovid19 obtiene información del programa COVID-19
func (s *ScraperExtendido) ScrapeProgramaCovid19(ruc string) (*models.ProgramaCovid19, error) {
	// Implementación similar a las anteriores
	covid := &models.ProgramaCovid19{
		RUC:           ruc,
		FechaConsulta: time.Now(),
	}
	// TODO: Implementar scraping
	return covid, nil
}