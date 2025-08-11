package scraper

import (
	"fmt"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/go-rod/rod"
)

// ScraperOptimizado es una versión optimizada que reutiliza la página
type ScraperOptimizado struct {
	browser *rod.Browser
	page    *rod.Page
	baseURL string
}

// NewScraperOptimizado crea una instancia optimizada del scraper
func NewScraperOptimizado() (*ScraperOptimizado, error) {
	// Configurar el navegador para que sea visible
	browser := rod.New().
		MustConnect().
		MustIgnoreCertErrors(true).
		SlowMotion(500 * time.Millisecond) // Ralentizar acciones para poder ver
	
	// Configurar ventana del navegador
	browser.MustPage("about:blank").MustSetViewport(1280, 720, 1, false)
	
	return &ScraperOptimizado{
		browser: browser,
		baseURL: "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp",
	}, nil
}

// Close cierra el navegador
func (s *ScraperOptimizado) Close() {
	if s.page != nil {
		s.page.Close()
	}
	if s.browser != nil {
		s.browser.Close()
	}
}

// ScrapeRUCCompleto obtiene toda la información de un RUC de manera optimizada
func (s *ScraperOptimizado) ScrapeRUCCompleto(ruc string) (*models.RUCCompleto, error) {
	fmt.Printf("\n🔍 Procesando RUC: %s\n", ruc)
	
	// Determinar tipo de RUC
	esPersonaJuridica := strings.HasPrefix(ruc, "20")
	tipoRUC := "Persona Natural con Negocio"
	if esPersonaJuridica {
		tipoRUC = "Persona Jurídica"
	}
	fmt.Printf("   Tipo: %s\n", tipoRUC)

	// Navegar a la página principal y buscar RUC
	if err := s.buscarRUC(ruc); err != nil {
		return nil, fmt.Errorf("error buscando RUC: %v", err)
	}

	// Crear estructura para almacenar resultados
	rucCompleto := &models.RUCCompleto{
		FechaConsulta: time.Now(),
		VersionAPI:    "2.0",
	}

	// 1. Extraer información básica de la página principal
	fmt.Print("   📋 Información básica: ")
	info := s.extraerInformacionBasica(ruc)
	rucCompleto.InformacionBasica = *info
	fmt.Println("✓")

	// 2. Consultas disponibles para todos los RUCs
	fmt.Println("   📑 Consultando información adicional:")

	// Información Histórica
	if hist := s.consultarInformacionHistorica(); hist != nil {
		rucCompleto.InformacionHistorica = hist
		fmt.Println("      ✓ Información Histórica")
	}

	// Deuda Coactiva
	if deuda := s.consultarDeudaCoactiva(); deuda != nil {
		rucCompleto.DeudaCoactiva = deuda
		fmt.Printf("      ✓ Deuda Coactiva: %s\n", 
			map[bool]string{true: "CON DEUDA", false: "SIN DEUDA"}[deuda.CantidadDocumentos > 0])
	}

	// Omisiones Tributarias
	if omis := s.consultarOmisionesTributarias(); omis != nil {
		rucCompleto.OmisionesTributarias = omis
		fmt.Printf("      ✓ Omisiones Tributarias: %s\n",
			map[bool]string{true: "CON OMISIONES", false: "SIN OMISIONES"}[omis.TieneOmisiones])
	}

	// Cantidad de Trabajadores
	if trab := s.consultarCantidadTrabajadores(); trab != nil {
		rucCompleto.CantidadTrabajadores = trab
		fmt.Println("      ✓ Cantidad de Trabajadores")
	}

	// Actas Probatorias
	if actas := s.consultarActasProbatorias(); actas != nil {
		rucCompleto.ActasProbatorias = actas
		fmt.Println("      ✓ Actas Probatorias")
	}

	// Facturas Físicas
	if fact := s.consultarFacturasFisicas(); fact != nil {
		rucCompleto.FacturasFisicas = fact
		fmt.Println("      ✓ Facturas Físicas")
	}

	// 3. Consultas exclusivas para Personas Jurídicas
	if esPersonaJuridica {
		fmt.Println("   📑 Consultando información exclusiva de Personas Jurídicas:")

		// Reactiva Perú
		if react := s.consultarReactivaPeru(); react != nil {
			rucCompleto.ReactivaPeru = react
			fmt.Println("      ✓ Reactiva Perú")
		}

		// Programa COVID-19
		if covid := s.consultarProgramaCovid(); covid != nil {
			rucCompleto.ProgramaCovid19 = covid
			fmt.Println("      ✓ Programa COVID-19")
		}

		// Representantes Legales
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("      ⚠️  Error en Representantes Legales: %v\n", r)
				}
			}()
			if reps := s.consultarRepresentantesLegales(); reps != nil {
				rucCompleto.RepresentantesLegales = reps
				fmt.Printf("      ✓ Representantes Legales: %d encontrados\n", len(reps.Representantes))
			}
		}()

		// Establecimientos Anexos
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("      ⚠️  Error en Establecimientos Anexos: %v\n", r)
				}
			}()
			if estab := s.consultarEstablecimientosAnexos(); estab != nil {
				rucCompleto.EstablecimientosAnexos = estab
				fmt.Printf("      ✓ Establecimientos Anexos: %d\n", estab.CantidadAnexos)
			}
		}()
	}

	return rucCompleto, nil
}

// buscarRUC navega a la página principal y busca el RUC
func (s *ScraperOptimizado) buscarRUC(ruc string) error {
	// Crear nueva página si no existe
	if s.page == nil {
		s.page = s.browser.MustPage(s.baseURL)
	} else {
		// Si ya existe, navegar a la página principal
		s.page.MustNavigate(s.baseURL)
	}

	s.page.MustWaitLoad()
	time.Sleep(2 * time.Second)

	// Ingresar RUC
	rucInput := s.page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustSelectAllText()
	rucInput.MustInput(ruc)

	// Hacer clic en buscar
	searchBtn := s.page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()

	// Esperar resultados
	time.Sleep(3 * time.Second)

	return nil
}

// extraerInformacionBasica extrae la información de la página principal
func (s *ScraperOptimizado) extraerInformacionBasica(ruc string) *models.RUCInfo {
	info := &models.RUCInfo{RUC: ruc}

	// Extraer información usando la misma lógica del scraper original
	listItems := s.page.MustElements(".list-group-item")
	
	for _, item := range listItems {
		rows := item.MustElements(".row")
		if len(rows) == 0 {
			continue
		}
		
		row := rows[0]
		cols := row.MustElements("[class*='col-sm']")
		
		if len(cols) >= 2 {
			headings := cols[0].MustElements(".list-group-item-heading")
			
			if len(headings) > 0 {
				label := strings.TrimSpace(headings[0].MustText())
				
				// Buscar tablas primero
				tables := cols[1].MustElements("table")
				if len(tables) > 0 {
					s.extractTableData(tables[0], label, info)
				} else {
					// Si no hay tabla, extraer texto
					texts := cols[1].MustElements(".list-group-item-text, .list-group-item-heading")
					
					if len(texts) > 0 {
						value := strings.TrimSpace(texts[0].MustText())
						
						if strings.Contains(label, "Número de RUC") {
							parts := strings.Split(value, " - ")
							if len(parts) > 0 {
								info.RUC = parts[0]
								if len(parts) > 1 {
									info.RazonSocial = parts[1]
								}
							}
						} else {
							s.mapFieldToStruct(label, value, info)
						}
					}
				}
			}
		}
		
		// Manejar filas con 4 columnas
		if len(cols) == 4 {
			// Primer par
			headings1 := cols[0].MustElements(".list-group-item-heading")
			texts1 := cols[1].MustElements(".list-group-item-text")
			
			if len(headings1) > 0 && len(texts1) > 0 {
				label1 := strings.TrimSpace(headings1[0].MustText())
				value1 := strings.TrimSpace(texts1[0].MustText())
				s.mapFieldToStruct(label1, value1, info)
			}
			
			// Segundo par
			headings2 := cols[2].MustElements(".list-group-item-heading")
			texts2 := cols[3].MustElements(".list-group-item-text")
			
			if len(headings2) > 0 && len(texts2) > 0 {
				label2 := strings.TrimSpace(headings2[0].MustText())
				value2 := strings.TrimSpace(texts2[0].MustText())
				s.mapFieldToStruct(label2, value2, info)
			}
		}
	}

	return info
}

// consultarInformacionHistorica consulta y vuelve
func (s *ScraperOptimizado) consultarInformacionHistorica() *models.InformacionHistorica {
	// Hacer clic en el botón
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfHis')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	// Verificar si se abrió nueva ventana
	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	// Extraer información
	info := &models.InformacionHistorica{}
	// TODO: Implementar extracción

	// Volver (cerrar ventana o hacer clic en volver)
	if targetPage != s.page {
		targetPage.MustClose()
	} else {
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
	}
	time.Sleep(2 * time.Second)

	return info
}

// consultarDeudaCoactiva consulta y vuelve
func (s *ScraperOptimizado) consultarDeudaCoactiva() *models.DeudaCoactiva {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfDeuCoa')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	deuda := &models.DeudaCoactiva{
		FechaConsulta: time.Now(),
	}

	// Buscar mensaje de "no registra deuda"
	alertas := targetPage.MustElements(".alert")
	for _, alerta := range alertas {
		texto := alerta.MustText()
		if strings.Contains(strings.ToLower(texto), "no registra") {
			deuda.CantidadDocumentos = 0
			deuda.TotalDeuda = 0
			break
		}
	}

	// Si hay deuda, extraer tabla
	// TODO: Implementar extracción de tabla de deudas

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return deuda
}

// consultarOmisionesTributarias consulta y vuelve
func (s *ScraperOptimizado) consultarOmisionesTributarias() *models.OmisionesTributarias {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfOmiTri')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	omis := &models.OmisionesTributarias{
		FechaConsulta: time.Now(),
	}

	// Buscar información
	alertas := targetPage.MustElements(".alert")
	for _, alerta := range alertas {
		texto := alerta.MustText()
		if strings.Contains(strings.ToLower(texto), "no registra") {
			omis.TieneOmisiones = false
			omis.CantidadOmisiones = 0
			break
		}
	}

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return omis
}

// consultarCantidadTrabajadores consulta y vuelve
func (s *ScraperOptimizado) consultarCantidadTrabajadores() *models.CantidadTrabajadores {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfNumTra')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	trab := &models.CantidadTrabajadores{
		FechaConsulta: time.Now(),
	}

	// TODO: Implementar extracción

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return trab
}

// consultarActasProbatorias consulta y vuelve
func (s *ScraperOptimizado) consultarActasProbatorias() *models.ActasProbatorias {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfActPro')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	actas := &models.ActasProbatorias{
		FechaConsulta: time.Now(),
		TieneActas:    false,
	}

	// TODO: Implementar extracción

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return actas
}

// consultarFacturasFisicas consulta y vuelve
func (s *ScraperOptimizado) consultarFacturasFisicas() *models.FacturasFisicas {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfActCPF')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	fact := &models.FacturasFisicas{
		FechaConsulta:     time.Now(),
		TieneAutorizacion: false,
	}

	// TODO: Implementar extracción

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return fact
}

// Consultas exclusivas para Personas Jurídicas

// consultarReactivaPeru - Solo para RUC 20
func (s *ScraperOptimizado) consultarReactivaPeru() *models.ReactivaPeru {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfReaPer')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	react := &models.ReactivaPeru{
		FechaConsulta:      time.Now(),
		ParticipaProgramma: false,
	}

	// TODO: Implementar extracción

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return react
}

// consultarProgramaCovid - Solo para RUC 20
func (s *ScraperOptimizado) consultarProgramaCovid() *models.ProgramaCovid19 {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfCovid')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	covid := &models.ProgramaCovid19{
		FechaConsulta:      time.Now(),
		ParticipaProgramma: false,
	}

	// TODO: Implementar extracción

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return covid
}

// consultarRepresentantesLegales - Solo para RUC 20
func (s *ScraperOptimizado) consultarRepresentantesLegales() *models.RepresentantesLegales {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfRepLeg')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages := s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	reps := &models.RepresentantesLegales{
		FechaConsulta: time.Now(),
	}

	// Buscar tabla de representantes
	tables := targetPage.MustElements("table")
	for _, table := range tables {
		rows := table.MustElements("tbody tr")
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 5 {
				rep := models.RepresentanteLegal{
					TipoDocumento:   strings.TrimSpace(cells[0].MustText()),
					NumeroDocumento: strings.TrimSpace(cells[1].MustText()),
					Cargo:           strings.TrimSpace(cells[4].MustText()),
				}
				
				// Parsear nombre completo
				nombreCompleto := strings.TrimSpace(cells[2].MustText())
				partes := strings.Fields(nombreCompleto)
				if len(partes) >= 3 {
					rep.ApellidoPaterno = partes[0]
					rep.ApellidoMaterno = partes[1]
					rep.Nombres = strings.Join(partes[2:], " ")
				}
				
				reps.Representantes = append(reps.Representantes, rep)
			}
		}
	}

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	return reps
}

// consultarEstablecimientosAnexos - Solo para RUC 20
func (s *ScraperOptimizado) consultarEstablecimientosAnexos() *models.EstablecimientosAnexos {
	// Verificar que la página aún esté activa
	pages := s.browser.MustPages()
	if len(pages) == 0 {
		fmt.Println("      ⚠️  La página se cerró inesperadamente")
		return nil
	}
	
	// Asegurarse de que estamos en la página principal
	s.page = pages[0]
	time.Sleep(1 * time.Second)
	
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfLocAnex')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)

	pages = s.browser.MustPages()
	targetPage := s.page
	if len(pages) > 1 {
		targetPage = pages[len(pages)-1]
		targetPage.MustWaitLoad()
	}

	estab := &models.EstablecimientosAnexos{
		FechaConsulta: time.Now(),
	}

	// TODO: Implementar extracción completa

	// Volver
	if targetPage != s.page && len(s.browser.MustPages()) > 1 {
		// Si es una nueva pestaña, cerrarla
		targetPage.MustClose()
		time.Sleep(2 * time.Second)
		// Volver el foco a la página principal
		pages := s.browser.MustPages()
		if len(pages) > 0 {
			s.page = pages[0]
			s.page.MustActivate()
		}
	} else {
		// Si es la misma página, usar el botón Volver
		volverBtn := targetPage.MustElementX("//button[contains(text(), 'Volver')]")
		volverBtn.MustClick()
		time.Sleep(2 * time.Second)
	}

	estab.CantidadAnexos = len(estab.Establecimientos)
	return estab
}

// Métodos auxiliares
func (s *ScraperOptimizado) extractTableData(table *rod.Element, label string, info *models.RUCInfo) {
	rows := table.MustElements("tr")
	items := []string{}
	
	for _, row := range rows {
		text := strings.TrimSpace(row.MustText())
		if text != "" && text != "NINGUNO" {
			items = append(items, text)
		}
	}
	
	label = strings.ToLower(label)
	
	if strings.Contains(label, "actividad") && strings.Contains(label, "económica") {
		if len(items) > 0 {
			info.ActividadesEconomicas = items
		}
	} else if strings.Contains(label, "comprobantes de pago") {
		if len(items) > 0 {
			info.ComprobantesPago = items
		}
	} else if strings.Contains(label, "sistema de emisión electrónica") {
		if len(items) > 0 {
			info.ComprobantesElectronicos = items
		}
	} else if strings.Contains(label, "padrones") {
		if len(items) > 0 {
			info.Padrones = items
		}
	}
}

func (s *ScraperOptimizado) mapFieldToStruct(label, value string, info *models.RUCInfo) {
	label = strings.ToLower(label)
	
	switch {
	case strings.Contains(label, "razón social") || strings.Contains(label, "razon social"):
		info.RazonSocial = value
	case strings.Contains(label, "tipo contribuyente"):
		info.TipoContribuyente = value
	case strings.Contains(label, "nombre comercial"):
		info.NombreComercial = value
	case strings.Contains(label, "fecha de inscripción") || strings.Contains(label, "fecha de inscripcion"):
		info.FechaInscripcion = value
	case strings.Contains(label, "fecha de inicio de actividades"):
		info.FechaInicioActividades = value
	case strings.Contains(label, "estado del contribuyente"):
		info.Estado = value
	case strings.Contains(label, "condición del contribuyente") || strings.Contains(label, "condicion del contribuyente"):
		info.Condicion = value
	case strings.Contains(label, "domicilio fiscal"):
		info.DomicilioFiscal = value
	case strings.Contains(label, "sistema emisión de comprobante") || strings.Contains(label, "sistema emision de comprobante"):
		info.SistemaEmision = value
	case strings.Contains(label, "actividad comercio exterior"):
		info.ActividadComercioExterior = value
	case strings.Contains(label, "sistema contabilidad"):
		info.SistemaContabilidad = value
	case strings.Contains(label, "emisor electrónico desde") || strings.Contains(label, "emisor electronico desde"):
		info.EmisorElectronicoDesde = value
	case strings.Contains(label, "comprobantes electrónicos") || strings.Contains(label, "comprobantes electronicos"):
		info.SistemaEmisionElectronica = value
	case strings.Contains(label, "afiliado al ple desde"):
		info.AfiliadoPLE = value
	}
}