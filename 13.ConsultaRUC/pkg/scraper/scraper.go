package scraper

import (
	"fmt"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type SUNATScraper struct {
	browser *rod.Browser
	baseURL string
}

func NewSUNATScraper() (*SUNATScraper, error) {
	l := launcher.New().
		Headless(false).
		Devtools(false)

	url := l.MustLaunch()
	browser := rod.New().
		ControlURL(url).
		MustConnect()

	return &SUNATScraper{
		browser: browser,
		baseURL: "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp",
	}, nil
}

func (s *SUNATScraper) Close() {
	s.browser.MustClose()
}

func (s *SUNATScraper) ScrapeRUC(ruc string, page *rod.Page) (*models.RUCInfo, error) {
	// Wait for results to load
	time.Sleep(5 * time.Second)

	info := &models.RUCInfo{
		RUC: ruc,
	}

	err := s.extractRUCInfo(page, info)
	if err != nil {
		return nil, fmt.Errorf("error extracting RUC info: %v", err)
	}

	return info, nil
}

func (s *SUNATScraper) extractRUCInfo(page *rod.Page, info *models.RUCInfo) error {
	// Extract data from Bootstrap list-group structure
	listItems := page.MustElements(".list-group-item")

	for _, item := range listItems {
		// Check if item has the standard structure
		rows := item.MustElements(".row")
		if len(rows) == 0 {
			continue
		}

		row := rows[0]
		cols := row.MustElements("[class*='col-sm']")

		if len(cols) >= 2 {
			// Extract heading
			headings := cols[0].MustElements(".list-group-item-heading")

			if len(headings) > 0 {
				label := strings.TrimSpace(headings[0].MustText())

				// Check for tables FIRST (before extracting text)
				tables := cols[1].MustElements("table")
				if len(tables) > 0 {
					s.extractTableData(tables[0], label, info)
				} else {
					// If no table, extract text normally
					texts := cols[1].MustElements(".list-group-item-text, .list-group-item-heading")

					if len(texts) > 0 {
						value := strings.TrimSpace(texts[0].MustText())

						// Handle special case for RUC number
						if strings.Contains(label, "Número de RUC") {
							// Extract just the RUC number
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

		// Handle rows with 4 columns (2 label-value pairs)
		if len(cols) == 4 {
			// First pair
			headings1 := cols[0].MustElements(".list-group-item-heading")
			texts1 := cols[1].MustElements(".list-group-item-text")

			if len(headings1) > 0 && len(texts1) > 0 {
				label1 := strings.TrimSpace(headings1[0].MustText())
				value1 := strings.TrimSpace(texts1[0].MustText())
				s.mapFieldToStruct(label1, value1, info)
			}

			// Second pair
			headings2 := cols[2].MustElements(".list-group-item-heading")
			texts2 := cols[3].MustElements(".list-group-item-text")

			if len(headings2) > 0 && len(texts2) > 0 {
				label2 := strings.TrimSpace(headings2[0].MustText())
				value2 := strings.TrimSpace(texts2[0].MustText())
				s.mapFieldToStruct(label2, value2, info)
			}
		}
	}

	return nil
}

func (s *SUNATScraper) mapFieldToStruct(label, value string, info *models.RUCInfo) {
	label = strings.ToLower(label)

	switch {
	case strings.Contains(label, "razón social") || strings.Contains(label, "razon social"):
		info.RazonSocial = value
	case strings.Contains(label, "tipo contribuyente"):
		info.TipoContribuyente = value
	case strings.Contains(label, "tipo de documento"):
		// Extraer información del documento
		// Formato esperado: "DNI  76366932  - TARRILLO MARRUFO, YAMELITH MARILYN"
		s.parseTipoDocumento(value, info)
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
		// Separar por comas y convertir a array
		if value != "" {
			items := strings.Split(value, ",")
			// Limpiar cada item
			for i, item := range items {
				items[i] = strings.TrimSpace(item)
			}
			info.ComprobantesElectronicos = items
		}
	case strings.Contains(label, "afiliado al ple desde"):
		info.AfiliadoPLE = value
	}
}

// Nueva función para parsear el tipo de documento
func (s *SUNATScraper) parseTipoDocumento(value string, info *models.RUCInfo) {
	// Limpiar espacios extra
	cleanValue := strings.ReplaceAll(value, "  ", " ")
	cleanValue = strings.TrimSpace(cleanValue)

	// Ejemplo: "DNI  76366932 - TARRILLO MARRUFO, YAMELITH MARILYN"
	if strings.Contains(cleanValue, "DNI") {
		// Extraer número de DNI y nombre
		parts := strings.Split(cleanValue, " - ")
		if len(parts) >= 2 {
			// Primera parte contiene "DNI" y el número
			dniPart := strings.TrimSpace(parts[0])
			dniFields := strings.Fields(dniPart) // Divide por espacios

			if len(dniFields) >= 2 {
				tipoDoc := dniFields[0]                       // "DNI"
				numeroDoc := dniFields[1]                     // "76366932"
				nombreCompleto := strings.TrimSpace(parts[1]) // "TARRILLO MARRUFO, YAMELITH MARILYN"

				// Formatear tipo de documento
				info.TipoDocumento = fmt.Sprintf("%s %s", tipoDoc, numeroDoc)

				// Si no se ha establecido RazonSocial desde el RUC, usar el nombre del documento
				if info.RazonSocial == "" {
					info.RazonSocial = nombreCompleto
				}
			}
		}
	} else {
		// Para otros tipos de documento, guardar tal como está
		info.TipoDocumento = cleanValue
	}
}

func (s *SUNATScraper) extractTableData(table *rod.Element, label string, info *models.RUCInfo) {
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
	} else if strings.Contains(label, "sistema de emisión electrónica") || strings.Contains(label, "sistema de emision electronica") {
		if len(items) > 0 {
			info.SistemaEmisionElectronica = items
		}
	} else if strings.Contains(label, "padrones") {
		if len(items) > 0 {
			info.Padrones = items
		}
	}
}
