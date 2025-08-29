package scraper

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/utils"
	"github.com/go-rod/rod"
)

// extractHistoricalInfo extrae la información histórica de la página
func (s *ScraperExtendido) extractHistoricalInfo(page *rod.Page, info *models.InformacionHistorica) {
	tables := page.MustElements("table")

	for _, table := range tables {
		headers := table.MustElements("thead th")
		if len(headers) == 0 {
			continue
		}

		firstHeaderText := strings.ToLower(strings.TrimSpace(headers[0].MustText()))

		rows := table.MustElements("tbody tr")

		switch {
		case strings.Contains(firstHeaderText, "nombre") || strings.Contains(firstHeaderText, "razón social") || strings.Contains(firstHeaderText, "razon social"):
			s.procesarCambiosRazonSocial(rows, info)

		case strings.Contains(firstHeaderText, "condición") && strings.Contains(firstHeaderText, "contribuyente"):
			s.procesarCondicionContribuyente(rows, info)

		case strings.Contains(firstHeaderText, "dirección") || strings.Contains(firstHeaderText, "domicilio"):
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
			}
		}
	}
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

func (s *ScraperExtendido) extractActasProbatoriasInfo(page *rod.Page, actas *models.ActasProbatorias) {
	// Buscar la tabla con clase "table"
	tables := page.MustElements("table.table")
	if len(tables) == 0 {
		return
	}

	rows := tables[0].MustElements("tbody tr")
	if len(rows) == 0 {
		return
	}

	// Verificar si hay mensaje de "no existe información"
	celdas := rows[0].MustElements("td")
	if len(celdas) == 1 && strings.Contains(strings.ToLower(celdas[0].MustText()), "no existe información") {
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
