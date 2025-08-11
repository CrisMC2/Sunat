package scraper

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"

	"github.com/PuerkitoBio/goquery"
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

	page := s.browser.MustPage(s.baseURL)
	defer page.MustClose()
	page.MustWaitLoad()
	page.MustWaitStable()

	// Ingresar RUC
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	rucInput.MustInput(ruc)

	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()
	searchBtn.MustClick()

	time.Sleep(5 * time.Second)

	// Consultas disponibles para todos los tipos de RUC
	fmt.Println("  📋 Consultando información principal...")
	fmt.Print("    - Información General: ")
	infob, err := s.ScrapeRUC(ruc, page)
	if err == nil {
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Crear estructura completa
	rucCompleto := &models.RUCCompleto{
		FechaConsulta:     time.Now(),
		InformacionBasica: *infob,
		VersionAPI:        "1.0.0",
	}

	// Determinar qué consultas están disponibles según el tipo de RUC
	esPersonaJuridica := strings.HasPrefix(ruc, "20")

	fmt.Printf("  ℹ️  RUC %s es: %s\n", ruc, map[bool]string{true: "Persona Jurídica", false: "Persona Natural"}[esPersonaJuridica])

	// Consultas disponibles para todos los tipos de RUC
	fmt.Println("  📋 Consultando información adicional...")

	// Deuda Coactiva
	fmt.Print("    - Deuda Coactiva: ")
	if deuda, err := s.ScrapeDeudaCoactiva(ruc, page); err == nil {
		rucCompleto.DeudaCoactiva = deuda
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Información Histórica
	fmt.Print("    - Información Histórica: ")
	if infoHist, err := s.ScrapeInformacionHistorica(ruc, page); err == nil {
		rucCompleto.InformacionHistorica = infoHist
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Omisiones Tributarias
	fmt.Print("    - Omisiones Tributarias: ")
	if omis, err := s.ScrapeOmisionesTributarias(ruc, page); err == nil {
		rucCompleto.OmisionesTributarias = omis
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Cantidad de Trabajadores
	fmt.Print("    - Cantidad de Trabajadores: ")
	if trab, err := s.ScrapeCantidadTrabajadores(ruc, page); err == nil {
		rucCompleto.CantidadTrabajadores = trab
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Actas Probatorias
	fmt.Print("    - Actas Probatorias: ")
	if actas, err := s.ScrapeActasProbatorias(ruc, page); err == nil {
		rucCompleto.ActasProbatorias = actas
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
	}

	// Facturas Físicas
	fmt.Print("    - Facturas Físicas: ")
	if fact, err := s.ScrapeFacturasFisicas(ruc, page); err == nil {
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
		if reps, err := s.ScrapeRepresentantesLegales(ruc, page); err == nil {
			rucCompleto.RepresentantesLegales = reps
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}

		// Establecimientos Anexos
		fmt.Print("    - Establecimientos Anexos: ")
		if estab, err := s.ScrapeEstablecimientosAnexos(ruc, page); err == nil {
			rucCompleto.EstablecimientosAnexos = estab
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}

		// Reactiva Perú
		fmt.Print("    - Reactiva Perú: ")
		if react, err := s.ScrapeReactivaPeru(ruc, page); err == nil {
			rucCompleto.ReactivaPeru = react
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}

		// Programa COVID-19
		fmt.Print("    - Programa COVID-19: ")
		if covid, err := s.ScrapeProgramaCovid19(ruc, page); err == nil {
			rucCompleto.ProgramaCovid19 = covid
			fmt.Println("✓")
		} else {
			fmt.Printf("✗ (%v)\n", err)
		}
	}

	return rucCompleto, nil
}

// ScrapeInformacionHistorica obtiene la información histórica del RUC
func (s *ScraperExtendido) ScrapeInformacionHistorica(ruc string, page *rod.Page) (*models.InformacionHistorica, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	histBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfHis')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := histBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de información histórica no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := histBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de información histórica está deshabilitado")
	}

	// Hacer clic en el botón
	err = histBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de información histórica: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de información histórica: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de información histórica no cargó correctamente: %w", err)
	}

	info := &models.InformacionHistorica{}

	// Extraer la información histórica con manejo de error
	s.extractHistoricalInfo(page, info)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return info, nil
}

// extractHistoricalInfo extrae la información histórica de la página
func (s *ScraperExtendido) extractHistoricalInfo(page *rod.Page, info *models.InformacionHistorica) {
	tables := page.MustElements("table")

	for i, table := range tables {
		headers := table.MustElements("thead th")
		if len(headers) == 0 {
			continue
		}

		firstHeaderText := strings.ToLower(strings.TrimSpace(headers[0].MustText()))
		log.Printf("📋 Procesando tabla %d con encabezado: '%s'", i+1, firstHeaderText)

		rows := table.MustElements("tbody tr")

		switch {
		case strings.Contains(firstHeaderText, "nombre") || strings.Contains(firstHeaderText, "razón social") || strings.Contains(firstHeaderText, "razon social"):
			log.Println("📝 Procesando razón social histórica...")
			s.procesarCambiosRazonSocial(rows, info)

		case strings.Contains(firstHeaderText, "condición") && strings.Contains(firstHeaderText, "contribuyente"):
			log.Println("📊 Procesando condición del contribuyente...")
			s.procesarCondicionContribuyente(rows, info)

		case strings.Contains(firstHeaderText, "dirección") || strings.Contains(firstHeaderText, "domicilio"):
			log.Println("🏠 Procesando domicilio fiscal histórico...")
			s.procesarCambiosDomicilio(rows, info)
		}
	}
}

func (s *ScraperExtendido) procesarCambiosRazonSocial(rows []*rod.Element, info *models.InformacionHistorica) {
	for _, row := range rows {
		cells := row.MustElements("td")
		if len(cells) >= 2 {
			nombre := strings.TrimSpace(cells[0].MustText())
			fechaBaja := strings.TrimSpace(cells[1].MustText())

			if nombre != "-" && !strings.Contains(nombre, "No hay Información") {
				cambio := models.RazonSocialHistorica{
					Nombre:      nombre,
					FechaDeBaja: fechaBaja,
				}
				info.RazonesSociales = append(info.RazonesSociales, cambio)
				log.Printf("  ✅ Razón social: %s (baja: %s)", nombre, fechaBaja)
			}
		}
	}
}

func (s *ScraperExtendido) procesarCondicionContribuyente(rows []*rod.Element, info *models.InformacionHistorica) {
	for _, row := range rows {
		cells := row.MustElements("td")
		if len(cells) >= 3 {
			condicion := strings.TrimSpace(cells[0].MustText())
			fechaDesde := strings.TrimSpace(cells[1].MustText())
			fechaHasta := strings.TrimSpace(cells[2].MustText())

			if condicion != "-" && condicion != "" {
				cambio := models.CondicionHistorica{
					Condicion: condicion,
					Desde:     fechaDesde,
					Hasta:     fechaHasta,
				}
				info.Condiciones = append(info.Condiciones, cambio)
				log.Printf("  ✅ Condición: %s (desde: %s hasta: %s)", condicion, fechaDesde, fechaHasta)
			}
		}
	}
}

func (s *ScraperExtendido) procesarCambiosDomicilio(rows []*rod.Element, info *models.InformacionHistorica) {
	for _, row := range rows {
		cells := row.MustElements("td")
		if len(cells) >= 2 {
			direccion := strings.TrimSpace(cells[0].MustText())
			fechaBaja := strings.TrimSpace(cells[1].MustText())

			if direccion != "-" && direccion != "" {
				cambio := models.DomicilioFiscalHistorico{
					Direccion:   direccion,
					FechaDeBaja: fechaBaja,
				}
				info.Domicilios = append(info.Domicilios, cambio)
				log.Printf("  ✅ Domicilio: %s (baja: %s)", direccion, fechaBaja)
			}
		}
	}
}

// ScrapeDeudaCoactiva obtiene información de deuda coactiva
func (s *ScraperExtendido) ScrapeDeudaCoactiva(ruc string, page *rod.Page) (*models.DeudaCoactiva, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	deudaBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfDeuCoa')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de deuda coactiva: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := deudaBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de deuda coactiva no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := deudaBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de deuda coactiva está deshabilitado")
	}

	// Hacer clic en el botón
	err = deudaBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de deuda coactiva: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de deuda coactiva: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de deuda coactiva no cargó correctamente: %w", err)
	}

	deuda := &models.DeudaCoactiva{}

	// Extraer la información de deuda coactiva
	s.extractDeudaInfo(page, deuda)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return deuda, nil
}

// extractDeudaInfo extrae la información de deuda coactiva
func (s *ScraperExtendido) extractDeudaInfo(page *rod.Page, deuda *models.DeudaCoactiva) {
	// Verificar si aparece el mensaje "no registra deuda coactiva"
	noDeudaElem := page.MustElements(".alert-info")
	if len(noDeudaElem) > 0 && strings.Contains(strings.ToLower(noDeudaElem[0].MustText()), "no registra deuda coactiva") {
		deuda.TotalDeuda = 0
		deuda.CantidadDocumentos = 0
		deuda.Deudas = []models.DetalleDeuda{}
		return
	}

	// Buscar la tabla de deudas
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")

		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 4 {
				detalle := models.DetalleDeuda{
					Monto:               utils.ParseMonto(cells[0].MustText()),
					PeriodoTributario:   strings.TrimSpace(cells[1].MustText()),
					FechaInicioCobranza: strings.TrimSpace(cells[2].MustText()),
					Entidad:             strings.TrimSpace(cells[3].MustText()),
				}

				deuda.Deudas = append(deuda.Deudas, detalle)
				deuda.TotalDeuda += detalle.Monto
			}
		}

		deuda.CantidadDocumentos = len(deuda.Deudas)
	}
}

// ScrapeRepresentantesLegales obtiene información de representantes legales
func (s *ScraperExtendido) ScrapeRepresentantesLegales(ruc string, page *rod.Page) (*models.RepresentantesLegales, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	repButton, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfRepLeg')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de representantes legales: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := repButton.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de representantes legales no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := repButton.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de representantes legales está deshabilitado")
	}

	// Hacer clic en el botón
	err = repButton.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de representantes legales: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de representantes legales: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de representantes legales no cargó correctamente: %w", err)
	}

	// Obtener HTML y parsear con goquery
	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("error al obtener HTML de representantes legales: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("error al parsear HTML: %w", err)
	}

	// Usar tu función existente extraerRepresentantes
	representantes := extraerRepresentantes(doc)

	representantesLegales := &models.RepresentantesLegales{
		Representantes: representantes,
	}
	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}
	return representantesLegales, nil
}

// extraerRepresentantes extrae los representantes legales desde el HTML
func extraerRepresentantes(doc *goquery.Document) []models.RepresentanteLegal {
	var representantes []models.RepresentanteLegal

	doc.Find("table").Each(func(i int, tabla *goquery.Selection) {
		headers := extraerCabeceras(tabla)

		// Comprobamos si esta tabla contiene los encabezados correctos
		if contieneCabeceras(headers, []string{"Documento", "Cargo", "Nombre"}) {
			tabla.Find("tbody tr").Each(func(j int, fila *goquery.Selection) {
				celdas := fila.Find("td")
				if celdas.Length() < 5 {
					log.Printf("[WARN] Fila ignorada: %d columnas (esperado al menos 5)\n", celdas.Length())
					return
				}

				rep := models.RepresentanteLegal{
					TipoDocumento:   strings.TrimSpace(celdas.Eq(0).Text()),
					NumeroDocumento: strings.TrimSpace(celdas.Eq(1).Text()),
					NombreCompleto:  strings.TrimSpace(celdas.Eq(2).Text()),
					Cargo:           strings.TrimSpace(celdas.Eq(3).Text()),
					FechaDesde:      strings.TrimSpace(celdas.Eq(4).Text()),
					Vigente:         true,
				}

				if celdas.Length() >= 6 {
					rep.FechaHasta = strings.TrimSpace(celdas.Eq(5).Text())
					rep.Vigente = rep.FechaHasta == ""
				}

				representantes = append(representantes, rep)
			})
		}
	})

	return representantes
}

func extraerCabeceras(tabla *goquery.Selection) []string {
	var headers []string
	tabla.Find("thead th").Each(func(i int, s *goquery.Selection) {
		headers = append(headers, strings.TrimSpace(s.Text()))
	})
	return headers
}

func contieneCabeceras(headers []string, requeridos []string) bool {
	for _, req := range requeridos {
		encontrado := false
		for _, h := range headers {
			if strings.Contains(strings.ToLower(h), strings.ToLower(req)) {
				encontrado = true
				break
			}
		}
		if !encontrado {
			return false
		}
	}
	return true
}

// ScrapeCantidadTrabajadores obtiene información de cantidad de trabajadores
func (s *ScraperExtendido) ScrapeCantidadTrabajadores(ruc string, page *rod.Page) (*models.CantidadTrabajadores, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	trabBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfNumTra')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de cantidad de trabajadores: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := trabBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de cantidad de trabajadores no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := trabBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de cantidad de trabajadores está deshabilitado")
	}

	// Hacer clic en el botón
	err = trabBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de cantidad de trabajadores: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de cantidad de trabajadores: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de cantidad de trabajadores no cargó correctamente: %w", err)
	}

	cantidadTrabajadores := &models.CantidadTrabajadores{}

	// Extraer información de cantidad de trabajadores
	s.extractTrabajadoresInfo(page, cantidadTrabajadores)
	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}
	return cantidadTrabajadores, nil
}

// extractTrabajadoresInfo extrae la información de trabajadores
func (s *ScraperExtendido) extractTrabajadoresInfo(page *rod.Page, trab *models.CantidadTrabajadores) {
	// Buscar tabla con información de trabajadores
	tables := page.MustElements("table.table")
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")
		for _, row := range rows {
			cells := row.MustElements("td")
			if len(cells) >= 4 {
				// Extraer textos de las celdas
				periodoText := strings.TrimSpace(cells[0].MustText())
				trabajadoresText := strings.TrimSpace(cells[1].MustText())
				pensionistasText := strings.TrimSpace(cells[2].MustText())
				prestadoresText := strings.TrimSpace(cells[3].MustText())

				// Parsear cantidades manejando "NE"
				trabajadores := parseCantidadConNE(trabajadoresText)
				pensionistas := parseCantidadConNE(pensionistasText)
				prestadores := parseCantidadConNE(prestadoresText)

				detalle := models.DetalleTrabajadores{
					Periodo:                     periodoText,
					CantidadTrabajadores:        trabajadores,
					CantidadPensionistas:        pensionistas,
					CantidadPrestadoresServicio: prestadores,
					Total:                       trabajadores + pensionistas + prestadores, // Total calculado
				}

				trab.DetallePorPeriodo = append(trab.DetallePorPeriodo, detalle)

				// Agregar período a lista de disponibles si no existe
				if !contains(trab.PeriodosDisponibles, detalle.Periodo) {
					trab.PeriodosDisponibles = append(trab.PeriodosDisponibles, detalle.Periodo)
				}
			}
		}
	}
}

// parseCantidadConNE versión más robusta
func parseCantidadConNE(text string) int {
	text = strings.TrimSpace(text)

	// Casos donde se considera como "no existe" o "sin datos"
	switch strings.ToUpper(text) {
	case "NE", "N/A", "NO EXISTE", "SIN DATOS", "", "-":
		return 0
	}

	// Remover caracteres no numéricos comunes (comas, espacios)
	text = strings.ReplaceAll(text, ",", "")
	text = strings.ReplaceAll(text, " ", "")

	// Intentar convertir a número
	if num, err := strconv.Atoi(text); err == nil {
		return num
	}

	// Si no se puede convertir, retornar 0 por defecto
	return 0
}

// Función auxiliar para verificar si un slice contiene un elemento
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ScrapeEstablecimientosAnexos obtiene información de establecimientos anexos
func (s *ScraperExtendido) ScrapeEstablecimientosAnexos(ruc string, page *rod.Page) (*models.EstablecimientosAnexos, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	estabBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfLocAnex')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de establecimientos anexos: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := estabBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de establecimientos anexos no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := estabBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de establecimientos anexos está deshabilitado")
	}

	// Hacer clic en el botón
	err = estabBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de establecimientos anexos: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de establecimientos anexos: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de establecimientos anexos no cargó correctamente: %w", err)
	}

	establecimientosAnexos := &models.EstablecimientosAnexos{}

	// Extraer información de establecimientos anexos
	err = s.extractEstablecimientosInfo(page, establecimientosAnexos)
	if err != nil {
		return nil, fmt.Errorf("error al extraer información de establecimientos: %w", err)
	}

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}
	return establecimientosAnexos, nil
}

// extractEstablecimientosInfo extrae la información de establecimientos anexos
func (s *ScraperExtendido) extractEstablecimientosInfo(page *rod.Page, estab *models.EstablecimientosAnexos) error {
	// Buscar tabla de establecimientos con manejo de errores
	tables, err := page.Elements("table.table")
	if err != nil {
		return fmt.Errorf("error al buscar tablas: %w", err)
	}

	if len(tables) == 0 {
		// No hay establecimientos anexos, pero no es un error
		estab.CantidadAnexos = 0
		return nil
	}

	rows, err := tables[0].Elements("tbody tr")
	if err != nil {
		return fmt.Errorf("error al obtener filas de la tabla: %w", err)
	}

	for _, row := range rows {
		cells, err := row.Elements("td")
		if err != nil {
			continue // Saltar esta fila si hay error
		}

		if len(cells) >= 4 {
			// Extraer texto con manejo de errores individual
			codigo, err := cells[0].Text()
			if err != nil {
				continue
			}

			tipo, err := cells[1].Text()
			if err != nil {
				continue
			}

			direccion, err := cells[2].Text()
			if err != nil {
				continue
			}

			actividad, err := cells[3].Text()
			if err != nil {
				continue
			}

			// Limpiar y validar
			codigo = strings.TrimSpace(codigo)
			tipo = strings.TrimSpace(tipo)
			direccion = strings.TrimSpace(direccion)
			actividad = strings.TrimSpace(actividad)

			// Validar campos críticos
			if codigo != "" && tipo != "" {
				anexo := models.EstablecimientoAnexo{
					Codigo:              codigo,
					TipoEstablecimiento: tipo,
					Direccion:           direccion,
					ActividadEconomica:  actividad,
				}
				estab.Establecimientos = append(estab.Establecimientos, anexo)
			}
		}
	}

	estab.CantidadAnexos = len(estab.Establecimientos)
	return nil
}

// Métodos adicionales para las demás consultas...

// ScrapeOmisionesTributarias obtiene información de omisiones tributarias
func (s *ScraperExtendido) ScrapeOmisionesTributarias(ruc string, page *rod.Page) (*models.OmisionesTributarias, error) {
	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	omisBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfOmiTri')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de omisiones tributarias: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := omisBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de omisiones tributarias no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := omisBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de omisiones tributarias está deshabilitado")
	}

	// Hacer clic en el botón
	err = omisBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de omisiones tributarias: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de omisiones tributarias: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de omisiones tributarias no cargó correctamente: %w", err)
	}

	omisionesTributarias := &models.OmisionesTributarias{}

	// Extraer información de omisiones tributarias
	s.extractOmisionesInfo(page, omisionesTributarias)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return omisionesTributarias, nil
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

// ScrapeActasProbatorias obtiene información de actas probatorias
func (s *ScraperExtendido) ScrapeActasProbatorias(ruc string, page *rod.Page) (*models.ActasProbatorias, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	actasBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfActPro')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de actas probatorias: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := actasBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de actas probatorias no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := actasBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de actas probatorias está deshabilitado")
	}

	// Hacer clic en el botón
	err = actasBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de actas probatorias: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de actas probatorias: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de actas probatorias no cargó correctamente: %w", err)
	}

	actasProbatorias := &models.ActasProbatorias{}

	// Extraer información de actas probatorias
	s.extractActasProbatoriasInfo(page, actasProbatorias)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return actasProbatorias, nil
}

func (s *ScraperExtendido) extractActasProbatoriasInfo(page *rod.Page, actas *models.ActasProbatorias) {
	// Buscar la tabla con clase "table"
	tables := page.MustElements("table.table")
	if len(tables) == 0 {
		log.Println("[INFO] No se encontró tabla de Actas Probatorias.")
		return
	}

	rows := tables[0].MustElements("tbody tr")
	if len(rows) == 0 {
		log.Println("[INFO] La tabla de Actas Probatorias no contiene filas.")
		return
	}

	// Verificar si hay mensaje de "no existe información"
	celdas := rows[0].MustElements("td")
	if len(celdas) == 1 && strings.Contains(strings.ToLower(celdas[0].MustText()), "no existe información") {
		log.Println("[INFO] No hay actas probatorias registradas para el contribuyente.")
		return
	}

	// Procesar filas normales
	for _, row := range rows {
		cells := row.MustElements("td")
		if len(cells) < 8 {
			continue // Saltar filas incompletas
		}

		acta := models.ActaProbatoria{
			NumeroActa:            strings.TrimSpace(cells[0].MustText()),
			FechaActa:             strings.TrimSpace(cells[1].MustText()),
			LugarIntervencion:     strings.TrimSpace(cells[2].MustText()),
			ArticuloNumeral:       strings.TrimSpace(cells[3].MustText()),
			DescripcionInfraccion: strings.TrimSpace(cells[4].MustText()),
			NumeroRIROZ:           strings.TrimSpace(cells[5].MustText()),
			TipoRIROZ:             strings.TrimSpace(cells[6].MustText()),
			ActaReconocimiento:    strings.TrimSpace(cells[7].MustText()),
		}

		actas.Actas = append(actas.Actas, acta)
	}

	// Completar campos calculados
	actas.CantidadActas = len(actas.Actas)
	actas.TieneActas = actas.CantidadActas > 0
}

// ScrapeFacturasFisicas obtiene información de facturas físicas
func (s *ScraperExtendido) ScrapeFacturasFisicas(ruc string, page *rod.Page) (*models.FacturasFisicas, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout - primero XPath
	facturasBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfActCPF')]")
	if err != nil {
		// Intentar con selector CSS como alternativa
		facturasBtn, err = page.Timeout(5 * time.Second).Element(".btnInfActCPF")
		if err != nil {
			return nil, fmt.Errorf("no se encontró el botón de facturas físicas: %w", err)
		}
	}

	// Verificar si el botón está visible
	visible, err := facturasBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de facturas físicas no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := facturasBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de facturas físicas está deshabilitado")
	}

	// Hacer scroll hasta el botón para asegurarse de que esté en viewport
	err = facturasBtn.ScrollIntoView()
	if err != nil {
		return nil, fmt.Errorf("error al hacer scroll al botón: %w", err)
	}

	time.Sleep(1 * time.Second)

	// Hacer clic en el botón
	err = facturasBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de facturas físicas: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de facturas físicas: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de facturas físicas no cargó correctamente: %w", err)
	}

	facturasFisicas := &models.FacturasFisicas{}

	// Extraer información de facturas físicas
	s.extractFacturasFisicasInfo(page, facturasFisicas)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return facturasFisicas, nil
}

// extractFacturasFisicasInfo extrae la información de facturas físicas
func (s *ScraperExtendido) extractFacturasFisicasInfo(page *rod.Page, facturas *models.FacturasFisicas) {
	// Buscar todas las tablas con clase "table"
	tables := page.MustElements("table.table")

	if len(tables) == 0 {
		facturas.TieneAutorizacion = false
		return
	}

	// Procesar primera tabla: "Facturas autorizadas"
	if len(tables) > 0 {
		rows := tables[0].MustElements("tbody tr")

		for _, row := range rows {
			cells := row.MustElements("td")

			// Verificar si es fila de "No hay Información"
			if len(cells) == 1 {
				cellText := strings.TrimSpace(cells[0].MustText())
				if strings.Contains(strings.ToLower(cellText), "no hay información") {
					continue // Saltar esta fila
				}
			}

			// Procesar fila con datos (6 columnas según el HTML)
			if len(cells) >= 6 {
				// Extraer textos de las celdas según las columnas del HTML:
				// Último Nro Autorización | Fecha de Última Autorización | Comprobante | Serie | del | al
				numeroAutorizacion := strings.TrimSpace(cells[0].MustText())
				fechaAutorizacion := strings.TrimSpace(cells[1].MustText())
				tipoComprobante := strings.TrimSpace(cells[2].MustText())
				serie := strings.TrimSpace(cells[3].MustText())
				numeroInicial := strings.TrimSpace(cells[4].MustText()) // "del"
				numeroFinal := strings.TrimSpace(cells[5].MustText())   // "al"

				// Validar que no estén vacíos los campos críticos
				if numeroAutorizacion != "" && numeroAutorizacion != "NE" {
					facturaAutorizada := models.FacturaAutorizada{
						NumeroAutorizacion: numeroAutorizacion,
						FechaAutorizacion:  fechaAutorizacion,
						TipoComprobante:    tipoComprobante,
						Serie:              serie,
						NumeroInicial:      numeroInicial,
						NumeroFinal:        numeroFinal,
					}
					facturas.Autorizaciones = append(facturas.Autorizaciones, facturaAutorizada)
				}
			}
		}
	}

	// Procesar segunda tabla: "Facturas dadas de baja y/o canceladas"
	if len(tables) > 1 {
		rows := tables[1].MustElements("tbody tr")

		for _, row := range rows {
			cells := row.MustElements("td")

			// Verificar si es fila de "No hay Información"
			if len(cells) == 1 {
				cellText := strings.TrimSpace(cells[0].MustText())
				if strings.Contains(strings.ToLower(cellText), "no hay información") {
					continue // Saltar esta fila
				}
			}

			// Procesar fila con datos (6 columnas según el HTML)
			if len(cells) >= 6 {
				// Extraer textos de las celdas según las columnas del HTML:
				// Nro Orden | Fecha de baja y/o cancelación | Comprobante | Serie | del | al
				numeroOrden := strings.TrimSpace(cells[0].MustText()) // "Nro Orden" (usamos como NumeroAutorizacion)
				fechaBaja := strings.TrimSpace(cells[1].MustText())   // "Fecha de baja y/o cancelación"
				tipoComprobante := strings.TrimSpace(cells[2].MustText())
				serie := strings.TrimSpace(cells[3].MustText())
				numeroInicial := strings.TrimSpace(cells[4].MustText()) // "del"
				numeroFinal := strings.TrimSpace(cells[5].MustText())   // "al"

				// Validar que no estén vacíos los campos críticos
				if numeroOrden != "" && numeroOrden != "NE" {
					facturaBaja := models.FacturaBajaOCancelada{
						NumeroAutorizacion: numeroOrden, // Usando Nro Orden como identificador
						FechaAutorizacion:  fechaBaja,   // Fecha de baja/cancelación
						TipoComprobante:    tipoComprobante,
						Serie:              serie,
						NumeroInicial:      numeroInicial,
						NumeroFinal:        numeroFinal,
					}
					facturas.CanceladasOBajas = append(facturas.CanceladasOBajas, facturaBaja)
				}
			}
		}
	}

	// Determinar si tiene autorización basándose en los datos encontrados
	facturas.TieneAutorizacion = len(facturas.Autorizaciones) > 0 || len(facturas.CanceladasOBajas) > 0
}

// ScrapeReactivaPeru obtiene información del programa Reactiva Perú
func (s *ScraperExtendido) ScrapeReactivaPeru(ruc string, page *rod.Page) (*models.ReactivaPeru, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	reactivaBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfReaPer')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de Reactiva Perú: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := reactivaBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de Reactiva Perú no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := reactivaBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de Reactiva Perú está deshabilitado")
	}

	// Hacer clic en el botón
	err = reactivaBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de Reactiva Perú: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de Reactiva Perú: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de Reactiva Perú no cargó correctamente: %w", err)
	}

	reactivaPeru := &models.ReactivaPeru{}

	// Extraer información de Reactiva Perú
	s.extractReactivaPeruInfo(page, reactivaPeru)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return reactivaPeru, nil
}

// extractReactivaPeruInfo extrae la información de Reactiva Perú desde la página
func (s *ScraperExtendido) extractReactivaPeruInfo(page *rod.Page, reactiva *models.ReactivaPeru) {
	defer func() {
		if r := recover(); r != nil {
			// Log error o manejar según necesites
			reactiva.TieneDeudaCoactiva = false
		}
	}()

	// Extraer RUC y Razón Social del título h3
	titleElements := page.MustElements("h3")
	if len(titleElements) > 0 {
		titleText := titleElements[0].MustText()
		// Ejemplo: "REACTIVA PERÚ DE 20606316977 - FERNANDEZ CONSULTORES SG & ASOCIADOS EIRL"
		if strings.Contains(titleText, " - ") {
			parts := strings.Split(titleText, " - ")
			if len(parts) >= 2 {
				reactiva.RazonSocial = strings.TrimSpace(parts[1])
			}
		}
	}

	// Buscar la respuesta a "¿Tiene deuda en cobranza coactiva mayor a una (1) UIT?"
	labelElements := page.MustElements(".label")
	for _, label := range labelElements {
		labelText := strings.TrimSpace(label.MustText())
		if labelText == "NO" {
			// Verificar si es label-success (NO)
			if classAttr, _ := label.Attribute("class"); classAttr != nil && strings.Contains(*classAttr, "label-success") {
				reactiva.TieneDeudaCoactiva = false
			}
		} else if labelText == "SÍ" || labelText == "SI" {
			// Verificar si es label-danger (SÍ)
			if classAttr, _ := label.Attribute("class"); classAttr != nil && strings.Contains(*classAttr, "label-danger") {
				reactiva.TieneDeudaCoactiva = true
			}
		}
	}

	// Buscar fecha de actualización y referencia legal en elementos h5
	h5Elements := page.MustElements("h5")
	for _, h5 := range h5Elements {
		h5Text := h5.MustText()
		if strings.Contains(h5Text, "información está actualizada al") {
			// Extraer la fecha usando regex
			datePattern := `(\d{2}/\d{2}/\d{4})`
			re := regexp.MustCompile(datePattern)
			matches := re.FindStringSubmatch(h5Text)
			if len(matches) > 1 {
				reactiva.FechaActualizacion = matches[1]
			}
		} else if strings.Contains(h5Text, "Decreto Legislativo") {
			// Extraer referencia legal
			reactiva.ReferenciaLegal = strings.TrimSpace(h5Text)
		}
	}
}

// ScrapeProgramaCovid19 obtiene información de programas COVID-19
func (s *ScraperExtendido) ScrapeProgramaCovid19(ruc string, page *rod.Page) (*models.ProgramaCovid19, error) {

	time.Sleep(5 * time.Second)

	// Buscar el botón usando ElementX (sin Must) con timeout
	covidBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfCovid')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de Programa COVID-19: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := covidBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de Programa COVID-19 no está visible o disponible")
	}

	// Verificar si el botón está deshabilitado
	if disabledAttr, _ := covidBtn.Attribute("disabled"); disabledAttr != nil {
		return nil, fmt.Errorf("el botón de Programa COVID-19 está deshabilitado")
	}

	// Hacer clic en el botón
	err = covidBtn.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de Programa COVID-19: %w", err)
	}

	time.Sleep(3 * time.Second)

	// Cambiar a la nueva pestaña si se abrió
	pages := s.browser.MustPages()
	if len(pages) > 1 {
		page = pages[len(pages)-1]
		err = page.WaitLoad()
		if err != nil {
			return nil, fmt.Errorf("error al cargar la página de Programa COVID-19: %w", err)
		}
	}

	err = page.WaitStable(10 * time.Second)
	if err != nil {
		return nil, fmt.Errorf("la página de Programa COVID-19 no cargó correctamente: %w", err)
	}

	programaCovid := &models.ProgramaCovid19{}

	// Extraer información del Programa COVID-19
	s.extractProgramaCovid19Info(page, programaCovid)

	// Buscar el botón usando ElementX (sin Must) con timeout
	volver, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	err = volver.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return nil, fmt.Errorf("error al hacer clic en el botón de volver: %w", err)
	}

	return programaCovid, nil
}

// extractProgramaCovid19Info extrae la información del Programa COVID-19 desde la página
func (s *ScraperExtendido) extractProgramaCovid19Info(page *rod.Page, programaCovid *models.ProgramaCovid19) {
	defer func() {
		if r := recover(); r != nil {
			// Log error o manejar según necesites
			programaCovid.ParticipaPrograma = false
			programaCovid.TieneDeudaCoactiva = false
		}
	}()

	// El RUC y FechaConsulta ya están establecidos en la función principal

	// Extraer RUC y Razón Social del título h3
	titleElements := page.MustElements("h3")
	if len(titleElements) > 0 {
		titleText := titleElements[0].MustText()
		// Ejemplo: "PROGRAMA DE GARANTÍAS COVID-19 DE 20606316977 - FERNANDEZ CONSULTORES SG & ASOCIADOS EIRL"
		if strings.Contains(titleText, " - ") {
			parts := strings.Split(titleText, " - ")
			if len(parts) >= 2 {
				programaCovid.RazonSocial = strings.TrimSpace(parts[1])
			}
		}
	}

	// Buscar información de participación en el programa
	// Puede aparecer como "¿Participa en el Programa de Garantías COVID-19?" o similar
	h5Elements := page.MustElements("h5")
	for _, h5 := range h5Elements {
		h5Text := h5.MustText()
		// Buscar preguntas relacionadas con participación en el programa
		if strings.Contains(strings.ToLower(h5Text), "participa") &&
			strings.Contains(strings.ToLower(h5Text), "programa") &&
			strings.Contains(strings.ToLower(h5Text), "covid") {
			// El siguiente elemento debería contener la respuesta
			break
		}
	}

	// Buscar la respuesta sobre deuda coactiva y participación en programa
	labelElements := page.MustElements(".label")
	for _, label := range labelElements {
		labelText := strings.TrimSpace(label.MustText())
		if labelText == "NO" {
			// Verificar si es label-success (NO)
			if classAttr, _ := label.Attribute("class"); classAttr != nil && strings.Contains(*classAttr, "label-success") {
				// Determinar si es para participación o deuda coactiva según el contexto
				programaCovid.TieneDeudaCoactiva = false
				programaCovid.ParticipaPrograma = false
			}
		} else if labelText == "SÍ" || labelText == "SI" {
			// Verificar si es label-danger (SÍ) o label-success
			if classAttr, _ := label.Attribute("class"); classAttr != nil {
				if strings.Contains(*classAttr, "label-success") {
					programaCovid.ParticipaPrograma = true
				} else if strings.Contains(*classAttr, "label-danger") {
					programaCovid.TieneDeudaCoactiva = true
				}
			}
		}
	}

	// Buscar información adicional en elementos h5
	for _, h5 := range h5Elements {
		h5Text := h5.MustText()

		// Buscar fecha de actualización
		if strings.Contains(h5Text, "información está actualizada al") ||
			strings.Contains(h5Text, "actualizada al") {
			// Extraer la fecha - buscar patrón XX/XX/XXXX
			words := strings.Fields(h5Text)
			for _, word := range words {
				if len(word) == 10 && strings.Count(word, "/") == 2 {
					programaCovid.FechaActualizacion = word
					break
				}
			}
		}

		// Buscar base legal (puede ser Ley N° 31050 o similar)
		if strings.Contains(h5Text, "Ley N°") ||
			strings.Contains(h5Text, "Decreto Legislativo") ||
			strings.Contains(h5Text, "Decreto Supremo") {
			programaCovid.BaseLegal = strings.TrimSpace(h5Text)
		}
	}

	// Buscar información en párrafos o divs adicionales
	// Puede haber información específica sobre el programa COVID-19
	paragraphs := page.MustElements("p.list-group-item-text")
	for _, p := range paragraphs {
		pText := p.MustText()
		// Buscar información relevante sobre COVID-19 o programas de garantías
		if strings.Contains(strings.ToLower(pText), "covid") ||
			strings.Contains(strings.ToLower(pText), "programa") ||
			strings.Contains(strings.ToLower(pText), "garantía") {
			// Procesar información adicional si es necesario
		}
	}

	// Si no se encontró información específica, verificar elementos alert
	alertElements := page.MustElements(".alert-info, .alert-success, .alert-warning")
	for _, alert := range alertElements {
		alertText := alert.MustText()
		if strings.Contains(strings.ToLower(alertText), "no participa") ||
			strings.Contains(strings.ToLower(alertText), "sin programa") {
			programaCovid.ParticipaPrograma = false
		} else if strings.Contains(strings.ToLower(alertText), "participa") {
			programaCovid.ParticipaPrograma = true
		}
	}
}
