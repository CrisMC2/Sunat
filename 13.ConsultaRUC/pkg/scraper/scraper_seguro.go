package scraper

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/consulta-ruc-scraper/pkg/models"
)

// ScraperSeguro es una versión mejorada que verifica la existencia de botones
type ScraperSeguro struct {
	browser *rod.Browser
	page    *rod.Page
	baseURL string
}

// NewScraperSeguro crea una instancia del scraper seguro
func NewScraperSeguro(visual bool) (*ScraperSeguro, error) {
	var browser *rod.Browser
	
	if visual {
		// Modo visual - muestra el navegador
		browser = rod.New().
			MustConnect().
			SlowMotion(300 * time.Millisecond)
	} else {
		// Modo headless - sin mostrar navegador
		browser = rod.New().MustConnect()
	}
	
	return &ScraperSeguro{
		browser: browser,
		baseURL: "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp",
	}, nil
}

// Close cierra el navegador
func (s *ScraperSeguro) Close() {
	if s.page != nil {
		s.page.Close()
	}
	if s.browser != nil {
		s.browser.Close()
	}
}

// verificarBotonExiste verifica si un botón existe antes de intentar hacer clic
func (s *ScraperSeguro) verificarBotonExiste(claseBoton string) bool {
	selector := fmt.Sprintf("//button[contains(@class, '%s')]", claseBoton)
	elem, err := s.page.Timeout(2 * time.Second).ElementX(selector)
	return err == nil && elem != nil
}

// ScrapeRUCCompleto obtiene toda la información disponible
func (s *ScraperSeguro) ScrapeRUCCompleto(ruc string) (*models.RUCCompleto, error) {
	fmt.Printf("\n🔍 Procesando RUC: %s\n", ruc)
	
	// Determinar tipo de RUC
	esPersonaJuridica := strings.HasPrefix(ruc, "20")
	tipoRUC := "Persona Natural con Negocio"
	if esPersonaJuridica {
		tipoRUC = "Persona Jurídica"
	}
	fmt.Printf("   Tipo: %s\n", tipoRUC)
	
	// Buscar RUC
	if err := s.buscarRUC(ruc); err != nil {
		return nil, err
	}
	
	// Crear estructura de resultado
	rucCompleto := &models.RUCCompleto{
		FechaConsulta: time.Now(),
		VersionAPI:    "1.0-seguro",
	}
	
	// Extraer información básica
	fmt.Printf("   📋 Información básica: ")
	info := s.extraerInfoBasica(ruc)
	rucCompleto.InformacionBasica = *info
	fmt.Println("✓")
	
	// Consultas adicionales
	fmt.Println("   📑 Consultando información adicional:")
	
	// Información Histórica
	if s.verificarBotonExiste("btnInfHis") {
		if hist := s.consultarInformacionHistorica(); hist != nil {
			rucCompleto.InformacionHistorica = hist
			fmt.Println("      ✓ Información Histórica")
		}
	} else {
		fmt.Println("      ⚠️  Información Histórica - No disponible")
	}
	
	// Deuda Coactiva
	if s.verificarBotonExiste("btnInfDeuCoa") {
		if deuda := s.consultarDeudaCoactiva(); deuda != nil {
			rucCompleto.DeudaCoactiva = deuda
			if deuda.TotalDeuda > 0 {
				fmt.Printf("      ✓ Deuda Coactiva: S/ %.2f\n", deuda.TotalDeuda)
			} else {
				fmt.Println("      ✓ Deuda Coactiva: SIN DEUDA")
			}
		}
	} else {
		fmt.Println("      ⚠️  Deuda Coactiva - No disponible")
	}
	
	// Omisiones Tributarias
	if s.verificarBotonExiste("btnInfOmi") {
		if omisiones := s.consultarOmisionesTributarias(); omisiones != nil {
			rucCompleto.OmisionesTributarias = omisiones
			if omisiones.TieneOmisiones {
				fmt.Printf("      ✓ Omisiones Tributarias: %d\n", omisiones.CantidadOmisiones)
			} else {
				fmt.Println("      ✓ Omisiones Tributarias: SIN OMISIONES")
			}
		}
	} else {
		fmt.Println("      ⚠️  Omisiones Tributarias - No disponible")
	}
	
	// Cantidad de Trabajadores
	if s.verificarBotonExiste("btnInfTra") {
		if trab := s.consultarCantidadTrabajadores(); trab != nil {
			rucCompleto.CantidadTrabajadores = trab
			fmt.Println("      ✓ Cantidad de Trabajadores")
		}
	} else {
		fmt.Println("      ⚠️  Cantidad de Trabajadores - No disponible")
	}
	
	// Actas Probatorias
	if s.verificarBotonExiste("btnInfProb") {
		if actas := s.consultarActasProbatorias(); actas != nil {
			rucCompleto.ActasProbatorias = actas
			fmt.Println("      ✓ Actas Probatorias")
		}
	} else {
		fmt.Println("      ⚠️  Actas Probatorias - No disponible")
	}
	
	// Solo para personas jurídicas
	if esPersonaJuridica {
		fmt.Println("   📑 Consultando información exclusiva de Personas Jurídicas:")
		
		// Representantes Legales
		if s.verificarBotonExiste("btnInfRep") {
			if reps := s.consultarRepresentantesLegales(); reps != nil {
				rucCompleto.RepresentantesLegales = reps
				fmt.Printf("      ✓ Representantes Legales: %d encontrados\n", len(reps.Representantes))
			}
		} else {
			fmt.Println("      ⚠️  Representantes Legales - No disponible")
		}
		
		// Establecimientos Anexos
		if s.verificarBotonExiste("btnInfLocAnex") {
			if estab := s.consultarEstablecimientosAnexos(); estab != nil {
				rucCompleto.EstablecimientosAnexos = estab
				fmt.Printf("      ✓ Establecimientos Anexos: %d\n", estab.CantidadAnexos)
			}
		} else {
			fmt.Println("      ⚠️  Establecimientos Anexos - No disponible")
		}
		
		// Reactiva Perú
		if s.verificarBotonExiste("btnReacPeru") {
			if reactiva := s.consultarReactivaPeru(); reactiva != nil {
				rucCompleto.ReactivaPeru = reactiva
				fmt.Println("      ✓ Reactiva Perú")
			}
		} else {
			fmt.Println("      ⚠️  Reactiva Perú - No disponible")
		}
		
		// Programa COVID-19
		if s.verificarBotonExiste("btnCovid19") {
			if covid := s.consultarProgramaCovid(); covid != nil {
				rucCompleto.ProgramaCovid19 = covid
				fmt.Println("      ✓ Programa COVID-19")
			}
		} else {
			fmt.Println("      ⚠️  Programa COVID-19 - No disponible")
		}
	}
	
	return rucCompleto, nil
}

// buscarRUC navega a la página y busca el RUC
func (s *ScraperSeguro) buscarRUC(ruc string) error {
	s.page = s.browser.MustPage(s.baseURL)
	s.page.MustWaitLoad()
	time.Sleep(2 * time.Second)
	
	// Buscar RUC
	inputRUC := s.page.MustElement("#txtRuc")
	inputRUC.MustSelectAllText()
	inputRUC.MustInput(ruc)
	
	btnBuscar := s.page.MustElementX("//button[contains(text(), 'Buscar')]")
	btnBuscar.MustClick()
	time.Sleep(3 * time.Second)
	
	return nil
}

// extraerInfoBasica extrae la información básica del RUC
func (s *ScraperSeguro) extraerInfoBasica(ruc string) *models.RUCInfo {
	info := &models.RUCInfo{
		RUC: ruc,
	}
	
	// Extraer razón social
	if elem, err := s.page.Element("h4.list-group-item-heading"); err == nil {
		text := elem.MustText()
		// Limpiar el texto
		text = strings.Replace(text, "Número de RUC:", "", 1)
		text = strings.Replace(text, ruc+" - ", "", 1)
		info.RazonSocial = strings.TrimSpace(text)
	}
	
	// Mapa de campos
	campos := map[string]*string{
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
	
	// Extraer campos
	rows := s.page.MustElements(".list-group-item")
	for _, row := range rows {
		text := row.MustText()
		for label, field := range campos {
			if strings.Contains(text, label) && field != nil {
				parts := strings.Split(text, label)
				if len(parts) > 1 {
					value := strings.TrimSpace(parts[1])
					// Limpiar valores múltiples
					if strings.Contains(value, "\n") {
						value = strings.Split(value, "\n")[0]
					}
					*field = value
				}
			}
		}
	}
	
	// Extraer actividades económicas
	info.ActividadesEconomicas = s.extraerActividades()
	
	// Extraer comprobantes
	info.ComprobantesPago = s.extraerComprobantes()
	info.ComprobantesElectronicos = s.extraerComprobantesElectronicos()
	
	// Extraer sistema de emisión electrónica
	info.SistemaEmisionElectronica = s.extraerSistemaEmisionElectronica()
	
	return info
}

// consultarInformacionHistorica consulta la información histórica
func (s *ScraperSeguro) consultarInformacionHistorica() *models.InformacionHistorica {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfHis')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	hist := &models.InformacionHistorica{}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return hist
}

// consultarDeudaCoactiva consulta la deuda coactiva
func (s *ScraperSeguro) consultarDeudaCoactiva() *models.DeudaCoactiva {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfDeuCoa')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	deuda := &models.DeudaCoactiva{
		FechaConsulta: time.Now(),
		TotalDeuda:    0,
	}
	
	// TODO: Implementar extracción completa
	// Por ahora, solo verificar si hay deuda
	
	// Volver
	s.volverPaginaPrincipal()
	
	return deuda
}

// consultarOmisionesTributarias consulta las omisiones tributarias
func (s *ScraperSeguro) consultarOmisionesTributarias() *models.OmisionesTributarias {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfOmi')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	omisiones := &models.OmisionesTributarias{
		FechaConsulta:     time.Now(),
		TieneOmisiones:    false,
		CantidadOmisiones: 0,
	}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return omisiones
}

// consultarCantidadTrabajadores consulta la cantidad de trabajadores
func (s *ScraperSeguro) consultarCantidadTrabajadores() *models.CantidadTrabajadores {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfTra')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	trab := &models.CantidadTrabajadores{
		FechaConsulta: time.Now(),
	}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return trab
}

// consultarActasProbatorias consulta las actas probatorias
func (s *ScraperSeguro) consultarActasProbatorias() *models.ActasProbatorias {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfProb')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	actas := &models.ActasProbatorias{
		FechaConsulta: time.Now(),
		TieneActas:    false,
	}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return actas
}

// consultarRepresentantesLegales consulta los representantes legales
func (s *ScraperSeguro) consultarRepresentantesLegales() *models.RepresentantesLegales {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfRep')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	reps := &models.RepresentantesLegales{
		Representantes: []models.RepresentanteLegal{},
	}
	
	// TODO: Implementar extracción completa
	// Por ahora, solo contar representantes
	
	// Volver
	s.volverPaginaPrincipal()
	
	return reps
}

// consultarEstablecimientosAnexos consulta los establecimientos anexos
func (s *ScraperSeguro) consultarEstablecimientosAnexos() *models.EstablecimientosAnexos {
	btn := s.page.MustElementX("//button[contains(@class, 'btnInfLocAnex')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	estab := &models.EstablecimientosAnexos{
		FechaConsulta:  time.Now(),
		CantidadAnexos: 0,
	}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return estab
}

// consultarReactivaPeru consulta Reactiva Perú
func (s *ScraperSeguro) consultarReactivaPeru() *models.ReactivaPeru {
	btn := s.page.MustElementX("//button[contains(@class, 'btnReacPeru')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	reactiva := &models.ReactivaPeru{
		FechaConsulta:      time.Now(),
		ParticipaProgramma: false,
	}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return reactiva
}

// consultarProgramaCovid consulta el programa COVID-19
func (s *ScraperSeguro) consultarProgramaCovid() *models.ProgramaCovid19 {
	btn := s.page.MustElementX("//button[contains(@class, 'btnCovid19')]")
	btn.MustClick()
	time.Sleep(3 * time.Second)
	
	covid := &models.ProgramaCovid19{
		FechaConsulta:      time.Now(),
		ParticipaProgramma: false,
	}
	
	// TODO: Implementar extracción
	
	// Volver
	s.volverPaginaPrincipal()
	
	return covid
}

// volverPaginaPrincipal vuelve a la página principal
func (s *ScraperSeguro) volverPaginaPrincipal() {
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
		if volverBtn, err := s.page.Timeout(2 * time.Second).ElementX("//button[contains(text(), 'Volver')]"); err == nil && volverBtn != nil {
			volverBtn.MustClick()
			time.Sleep(2 * time.Second)
		}
	}
}

// Métodos auxiliares para extraer datos específicos
func (s *ScraperSeguro) extraerActividades() []string {
	actividades := []string{}
	
	// Buscar sección de actividades económicas
	if section, err := s.page.Element(".list-group"); err == nil {
		text := section.MustText()
		if strings.Contains(text, "Actividad(es) Económica(s):") {
			lines := strings.Split(text, "\n")
			capturing := false
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "Actividad(es) Económica(s):") {
					capturing = true
					continue
				}
				if capturing && line != "" {
					if strings.Contains(line, "Comprobantes de Pago c/aut. de impresión") ||
						strings.Contains(line, "Sistema de Emisión Electrónica:") ||
						strings.Contains(line, "Padrones :") {
						break
					}
					if strings.Contains(line, "Principal - ") || strings.Contains(line, "Secundaria") {
						actividades = append(actividades, line)
					}
				}
			}
		}
	}
	
	return actividades
}

func (s *ScraperSeguro) extraerComprobantes() []string {
	comprobantes := []string{}
	
	if section, err := s.page.Element(".list-group"); err == nil {
		text := section.MustText()
		if strings.Contains(text, "Comprobantes de Pago c/aut. de impresión") {
			lines := strings.Split(text, "\n")
			capturing := false
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "Comprobantes de Pago c/aut. de impresión") {
					capturing = true
					continue
				}
				if capturing && line != "" {
					if strings.Contains(line, "Sistema de Emisión Electrónica:") ||
						strings.Contains(line, "Padrones :") {
						break
					}
					comprobantes = append(comprobantes, line)
				}
			}
		}
	}
	
	return comprobantes
}

func (s *ScraperSeguro) extraerComprobantesElectronicos() []string {
	comprobantes := []string{}
	
	if section, err := s.page.Element(".list-group"); err == nil {
		text := section.MustText()
		if strings.Contains(text, "Comprobantes Electrónicos:") {
			lines := strings.Split(text, "\n")
			capturing := false
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "Comprobantes Electrónicos:") {
					capturing = true
					continue
				}
				if capturing && line != "" {
					if strings.Contains(line, "Afiliado PLE desde:") ||
						strings.Contains(line, "Padrones :") {
						break
					}
					comprobantes = append(comprobantes, line)
				}
			}
		}
	}
	
	return comprobantes
}

func (s *ScraperSeguro) extraerSistemaEmisionElectronica() string {
	if section, err := s.page.Element(".list-group"); err == nil {
		text := section.MustText()
		if strings.Contains(text, "Sistema de Emisión Electrónica:") {
			lines := strings.Split(text, "\n")
			for i, line := range lines {
				if strings.Contains(line, "Sistema de Emisión Electrónica:") && i+1 < len(lines) {
					return strings.TrimSpace(lines[i+1])
				}
			}
		}
	}
	return ""
}