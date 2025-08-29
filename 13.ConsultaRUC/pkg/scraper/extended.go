package scraper

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/consulta-ruc-scraper/pkg/database"
	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/go-rod/rod"
)

// ImprimirBotonesDisponibles encuentra e imprime todos los botones disponibles en la página
func (s *ScraperExtendido) ImprimirBotonesDisponibles(page *rod.Page) {
	fmt.Println("\n🔍 === ANÁLISIS DE BOTONES DISPONIBLES ===")

	// Mapa de botones conocidos con sus selectores y nombres amigables
	botonesConocidos := map[string]string{
		"btnInfHis":     "Información Histórica",
		"btnInfDeuCoa":  "Deuda Coactiva",
		"btnInfOmiTri":  "Omisiones Tributarias",
		"btnInfNumTra":  "Cantidad de Trabajadores",
		"btnInfActPro":  "Actas Probatorias",
		"btnInfActCPF":  "Facturas Físicas",
		"btnInfRepLeg":  "Representantes Legales",
		"btnInfLocAnex": "Establecimientos Anexos",
		"btnInfReaPer":  "Reactiva Perú",
		"btnInfCovid":   "Programa COVID-19",
	}

	// Contador de botones encontrados
	botonesEncontrados := 0
	botonesActivos := 0

	fmt.Printf("📋 Escaneando página en busca de botones...\n\n")

	// Buscar cada botón conocido
	for claseBoton, nombreBoton := range botonesConocidos {
		xpath := fmt.Sprintf("//button[contains(@class, '%s')]", claseBoton)

		// Intentar encontrar el botón
		boton, err := page.Timeout(2 * time.Second).ElementX(xpath)
		if err != nil {
			// Botón no encontrado
			fmt.Printf("❌ %-25s | %-30s | NO ENCONTRADO\n", nombreBoton, claseBoton)
			continue
		}

		botonesEncontrados++

		// Verificar si está visible
		visible, _ := boton.Visible()

		// Obtener el texto del botón
		textoBoton, _ := boton.Text()
		if textoBoton == "" {
			textoBoton = "Sin texto"
		}

		// Obtener atributos adicionales de forma segura
		titulo, _ := boton.Attribute("title")
		tituloStr := ""
		if titulo != nil {
			tituloStr = *titulo
		}
		if tituloStr == "" {
			ariaLabel, _ := boton.Attribute("aria-label")
			if ariaLabel != nil {
				tituloStr = *ariaLabel
			}
		}

		// Verificar si está deshabilitado usando un método más simple
		isDisabled := false
		disabledAttr, _ := boton.Attribute("disabled")
		if disabledAttr != nil {
			// Si el atributo disabled existe, el botón está deshabilitado
			isDisabled = true
		} else {
			// También verificar via JavaScript como alternativa
			jsResult, err := boton.Eval("() => this.disabled")
			if err == nil {
				isDisabled = jsResult.Value.Bool()
			}
		}

		// Determinar estado del botón
		estado := ""
		icono := ""

		if !visible {
			estado = "OCULTO"
			icono = "👻"
		} else if isDisabled {
			estado = "DESHABILITADO"
			icono = "🚫"
		} else {
			estado = "ACTIVO"
			icono = "✅"
			botonesActivos++
		}

		// Imprimir información del botón
		fmt.Printf("%s %-25s | %-30s | %-12s",
			icono, nombreBoton, claseBoton, estado)

		if textoBoton != "Sin texto" {
			fmt.Printf(" | Texto: '%s'", strings.TrimSpace(textoBoton))
		}

		if tituloStr != "" {
			fmt.Printf(" | Título: '%s'", tituloStr)
		}

		fmt.Println()
	}

	// Buscar botones adicionales que no estén en nuestra lista
	fmt.Printf("\n🔎 Buscando botones adicionales...\n")

	// XPath general para encontrar cualquier botón
	botonesGenerales, err := page.ElementsX("//button")
	if err == nil {
		botonesAdicionales := 0

		for _, boton := range botonesGenerales {
			// Obtener clases del botón
			clases, _ := boton.Attribute("class")
			clasesStr := ""
			if clases != nil {
				clasesStr = *clases
			}
			if clasesStr == "" {
				continue
			}

			// Verificar si ya lo conocemos
			esConocido := false
			for claseConocida := range botonesConocidos {
				if strings.Contains(clasesStr, claseConocida) {
					esConocido = true
					break
				}
			}

			if !esConocido {
				// Es un botón desconocido
				texto, _ := boton.Text()
				if texto == "" {
					texto = "Sin texto"
				}

				visible, _ := boton.Visible()

				// Verificar si está deshabilitado de forma simple
				isDisabled := false
				disabledAttr, _ := boton.Attribute("disabled")
				if disabledAttr != nil {
					isDisabled = true
				}

				estado := ""
				icono := ""

				if !visible {
					estado = "OCULTO"
					icono = "👻"
				} else if isDisabled {
					estado = "DESHABILITADO"
					icono = "🚫"
				} else {
					estado = "ACTIVO"
					icono = "❓"
				}

				fmt.Printf("%s %-25s | %-30s | %-12s | Texto: '%s'\n",
					icono, "BOTÓN DESCONOCIDO", clasesStr, estado, strings.TrimSpace(texto))

				botonesAdicionales++
			}
		}

		if botonesAdicionales == 0 {
			fmt.Printf("   No se encontraron botones adicionales\n")
		}
	}

	// Resumen final
	fmt.Printf("\n📊 === RESUMEN ===\n")
	fmt.Printf("🔢 Total de botones conocidos encontrados: %d/%d\n", botonesEncontrados, len(botonesConocidos))
	fmt.Printf("✅ Botones activos (disponibles para clic): %d\n", botonesActivos)
	fmt.Printf("📍 Página actual: %s\n", page.MustInfo().URL)
	fmt.Printf("=================================================\n\n")
}

// VerificarDisponibilidadBoton verifica si un botón específico está disponible para hacer clic
func (s *ScraperExtendido) VerificarDisponibilidadBoton(page *rod.Page, claseBoton string) bool {
	xpath := fmt.Sprintf("//button[contains(@class, '%s')]", claseBoton)

	boton, err := page.Timeout(2 * time.Second).ElementX(xpath)
	if err != nil {
		return false
	}

	visible, _ := boton.Visible()
	if !visible {
		return false
	}

	// Verificar si está deshabilitado
	disabledAttr, _ := boton.Attribute("disabled")
	if disabledAttr != nil {
		return false
	}

	// Verificación adicional via JavaScript
	jsResult, err := boton.Eval("() => this.disabled")
	if err == nil && jsResult.Value.Bool() {
		return false
	}

	return true
}

// ObtenerBotonesDisponibles devuelve un mapa con los botones disponibles
func (s *ScraperExtendido) ObtenerBotonesDisponibles(page *rod.Page) map[string]bool {
	botonesDisponibles := make(map[string]bool)

	botones := map[string]string{
		"btnInfHis":     "Información Histórica",
		"btnInfDeuCoa":  "Deuda Coactiva",
		"btnInfOmiTri":  "Omisiones Tributarias",
		"btnInfNumTra":  "Cantidad de Trabajadores",
		"btnInfActPro":  "Actas Probatorias",
		"btnInfActCPF":  "Facturas Físicas",
		"btnInfRepLeg":  "Representantes Legales",
		"btnInfLocAnex": "Establecimientos Anexos",
		"btnInfReaPer":  "Reactiva Perú",
		"btnInfCovid":   "Programa COVID-19",
	}

	for clase := range botones {
		botonesDisponibles[clase] = s.VerificarDisponibilidadBoton(page, clase)
	}

	return botonesDisponibles
}

// retryScrapeWithPartialSave - Función mejorada que guarda datos parciales antes de terminar
func (s *ScraperExtendido) retryScrapeWithPartialSave(maxRetries int, name string, scrapeFunc func() error, rucCompleto *models.RUCCompleto, dbService *database.DatabaseService, ruc string, page *rod.Page) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := scrapeFunc(); err == nil {
			fmt.Println("✓")
			return
		} else {
			fmt.Printf("✗ intento %d/%d (%v)\n", attempt, maxRetries, err)
			// Delay inteligente entre reintentos (aumenta con cada intento)
			retryDelay := s.humanSim.generateLogNormalDelay(2000*float64(attempt), 800)
			time.Sleep(retryDelay)
		}
	}

	fmt.Printf("❌ Falló %s después de %d intentos.\n", name, maxRetries)

	// GUARDAR DATOS PARCIALES ANTES DE TERMINAR
	if rucCompleto != nil && dbService != nil {
		fmt.Printf("💾 Guardando datos parciales obtenidos hasta el momento para RUC %s...\n", ruc)
		if err := dbService.InsertRUCCompleto(rucCompleto); err != nil {
			fmt.Printf("❌ Error guardando datos parciales: %v\n", err)
		} else {
			fmt.Printf("✅ Datos parciales guardados exitosamente para RUC %s\n", ruc)
		}
	}

	fmt.Printf("🛑 Terminando programa debido a falla en: %s\n", name)
	fmt.Printf("📄 Último HTML obtenido: %s\n", page.MustHTML())
	os.Exit(1)
}

// ScrapeRUCCompleto obtiene toda la información disponible de un RUC usando detección de botones
func (s *ScraperExtendido) ScrapeRUCCompleto(ruc string, dbService *database.DatabaseService) (*models.RUCCompleto, error) {
	page := s.browser.MustPage(s.baseURL)
	defer func() {
		page.MustClose()
		s.browser.MustClose() // Cierra el navegador entero
	}()

	// Carga humana de página
	err := s.HumanPageLoad(page)
	if err != nil {
		return nil, fmt.Errorf("error en carga humana de página: %w", err)
	}

	// Ingresar RUC
	rucInput := page.MustElement("#txtRuc")
	rucInput.MustWaitVisible()
	err = s.HumanInput(rucInput, ruc)
	if err != nil {
		return nil, fmt.Errorf("error ingresando RUC: %w", err)
	}
	searchBtn := page.MustElement("#btnAceptar")
	searchBtn.MustWaitVisible()

	err = s.HumanClick(searchBtn, page)
	if err != nil {
		return nil, fmt.Errorf("error haciendo clic en buscar: %w", err)
	}

	// Esperar resultados con comportamiento humano
	err = s.HumanPageLoad(page)
	if err != nil {
		return nil, fmt.Errorf("error cargando resultados: %w", err)
	}

	// Consulta información general (siempre disponible)
	fmt.Println(" 📋 Consultando información principal...")
	fmt.Print(" - Información General: ")
	infob, err := s.ScrapeRUC(ruc, page)
	if err == nil {
		fmt.Println("✓")
	} else {
		fmt.Printf("✗ (%v)\n", err)
		// Si falla la información básica, no podemos continuar
		return nil, fmt.Errorf("error obteniendo información básica: %w", err)
	}

	// Crear estructura completa
	rucCompleto := &models.RUCCompleto{
		FechaConsulta:       time.Now(),
		InformacionBasica:   *infob,
		VersionAPI:          "1.0.0",
		DeteccionPaginacion: make(map[string]bool),
	}

	// Determinar tipo de RUC
	esPersonaJuridica := strings.HasPrefix(ruc, "20")
	fmt.Printf(" ℹ️ RUC %s es: %s (Fatiga: %.2f)\n", ruc,
		map[bool]string{true: "Persona Jurídica", false: "Persona Natural"}[esPersonaJuridica],
		s.humanSim.fatigueLevel)

	// Analizar botones disponibles
	s.ImprimirBotonesDisponibles(page)
	botonesDisponibles := s.ObtenerBotonesDisponibles(page)

	// ========================================
	// CONSULTAS BASADAS EN DISPONIBILIDAD DE BOTONES
	// ========================================
	fmt.Println(" 📋 Iniciando scraping basado en botones disponibles...")

	// 1. Información Histórica
	if botonesDisponibles["btnInfHis"] {
		fmt.Print(" - Información Histórica: ")
		s.retryScrapeWithPartialSave(3, "Información Histórica", func() error {
			infoHist, tienePaginacion, err := s.ScrapeInformacionHistorica(ruc, page)
			if err == nil {
				rucCompleto.InformacionHistorica = infoHist
				rucCompleto.DeteccionPaginacion["Información Histórica"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Información Histórica: ❌ Botón no disponible")
	}

	// 2. Deuda Coactiva
	if botonesDisponibles["btnInfDeuCoa"] {
		fmt.Print(" - Deuda Coactiva: ")
		s.retryScrapeWithPartialSave(3, "Deuda Coactiva", func() error {
			deuda, tienePaginacion, err := s.ScrapeDeudaCoactiva(ruc, page)
			if err == nil {
				rucCompleto.DeudaCoactiva = deuda
				rucCompleto.DeteccionPaginacion["Deuda Coactiva"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Deuda Coactiva: ❌ Botón no disponible")
	}

	// 3. Omisiones Tributarias
	if botonesDisponibles["btnInfOmiTri"] {
		fmt.Print(" - Omisiones Tributarias: ")
		s.retryScrapeWithPartialSave(3, "Omisiones Tributarias", func() error {
			omis, tienePaginacion, err := s.ScrapeOmisionesTributarias(ruc, page)
			if err == nil {
				rucCompleto.OmisionesTributarias = omis
				rucCompleto.DeteccionPaginacion["Omisiones Tributarias"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Omisiones Tributarias: ❌ Botón no disponible")
	}

	// 4. Cantidad de Trabajadores
	if botonesDisponibles["btnInfNumTra"] {
		fmt.Print(" - Cantidad de Trabajadores: ")
		s.retryScrapeWithPartialSave(3, "Cantidad de Trabajadores", func() error {
			trab, tienePaginacion, err := s.ScrapeCantidadTrabajadores(ruc, page)
			if err == nil {
				rucCompleto.CantidadTrabajadores = trab
				rucCompleto.DeteccionPaginacion["Cantidad de Trabajadores"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Cantidad de Trabajadores: ❌ Botón no disponible")
	}

	// 5. Actas Probatorias
	if botonesDisponibles["btnInfActPro"] {
		fmt.Print(" - Actas Probatorias: ")
		s.retryScrapeWithPartialSave(3, "Actas Probatorias", func() error {
			actas, tienePaginacion, err := s.ScrapeActasProbatorias(ruc, page)
			if err == nil {
				rucCompleto.ActasProbatorias = actas
				rucCompleto.DeteccionPaginacion["Actas Probatorias"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Actas Probatorias: ❌ Botón no disponible")
	}

	// 6. Facturas Físicas
	if botonesDisponibles["btnInfActCPF"] {
		fmt.Print(" - Facturas Físicas: ")
		s.retryScrapeWithPartialSave(3, "Facturas Físicas", func() error {
			fact, tienePaginacion, err := s.ScrapeFacturasFisicas(ruc, page)
			if err == nil {
				rucCompleto.FacturasFisicas = fact
				rucCompleto.DeteccionPaginacion["Facturas Físicas"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Facturas Físicas: ❌ Botón no disponible")
	}

	// 7. Representantes Legales
	if botonesDisponibles["btnInfRepLeg"] {
		fmt.Print(" - Representantes Legales: ")
		s.retryScrapeWithPartialSave(3, "Representantes Legales", func() error {
			reps, tienePaginacion, err := s.ScrapeRepresentantesLegales(ruc, page)
			if err == nil {
				rucCompleto.RepresentantesLegales = reps
				rucCompleto.DeteccionPaginacion["Representantes Legales"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Representantes Legales: ❌ Botón no disponible")
	}

	// 8. Establecimientos Anexos
	if botonesDisponibles["btnInfLocAnex"] {
		fmt.Print(" - Establecimientos Anexos: ")
		s.retryScrapeWithPartialSave(3, "Establecimientos Anexos", func() error {
			estab, tienePaginacion, err := s.ScrapeEstablecimientosAnexos(ruc, page)
			if err == nil {
				rucCompleto.EstablecimientosAnexos = estab
				rucCompleto.DeteccionPaginacion["Establecimientos Anexos"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Establecimientos Anexos: ❌ Botón no disponible")
	}

	// 9. Reactiva Perú
	if botonesDisponibles["btnInfReaPer"] {
		fmt.Print(" - Reactiva Perú: ")
		s.retryScrapeWithPartialSave(3, "Reactiva Perú", func() error {
			react, err := s.ScrapeReactivaPeru(ruc, page)
			if err == nil {
				rucCompleto.ReactivaPeru = react
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Reactiva Perú: ❌ Botón no disponible")
	}

	// 10. Programa COVID-19
	if botonesDisponibles["btnInfCovid"] {
		fmt.Print(" - Programa COVID-19: ")
		s.retryScrapeWithPartialSave(3, "Programa COVID-19", func() error {
			covid, err := s.ScrapeProgramaCovid19(ruc, page)
			if err == nil {
				rucCompleto.ProgramaCovid19 = covid
			}
			return err
		}, rucCompleto, dbService, ruc, page)
	} else {
		fmt.Println(" - Programa COVID-19: ❌ Botón no disponible")
	}

	fmt.Printf("\n✅ Scraping completado para RUC %s\n", ruc)
	fmt.Printf("📊 Tipo de RUC: %s\n", map[bool]string{true: "Persona Jurídica (20)", false: "Persona Natural (10)"}[esPersonaJuridica])

	return rucCompleto, nil
}

// VERSIÓN OPTIMIZADA - Reduce tiempo de ejecución significativamente
func (s *ScraperExtendido) VerificarPaginaCorrecta(page *rod.Page, tipoSeccion string) bool {
	// 1. ELIMINAR SLEEP INNECESARIO - ya tienes WaitLoad()
	// time.Sleep(1 * time.Second) // ❌ ELIMINADO

	// Esperar que la página esté completamente cargada
	page.WaitLoad()

	// 2. USAR TIMEOUTS CORTOS para elementos que pueden no existir
	timeout := 3 * time.Second

	// 3. BÚSQUEDA PARALELA de elementos usando goroutines para mayor velocidad
	type ElementResult struct {
		texto string
		found bool
	}

	results := make(chan ElementResult, 4)

	// Función helper para buscar elemento con timeout
	buscarElemento := func(selector string) {
		defer func() {
			if r := recover(); r != nil {
				results <- ElementResult{"", false}
			}
		}()

		element, err := page.Timeout(timeout).Element(selector)
		if err != nil {
			results <- ElementResult{"", false}
			return
		}

		texto, err := element.Text()
		if err != nil {
			results <- ElementResult{"", false}
			return
		}

		results <- ElementResult{texto, true}
	}

	// Lanzar búsquedas en paralelo
	go buscarElemento("title")
	go buscarElemento("h1")
	go buscarElemento("h2")
	go buscarElemento("h3")

	// Recolectar resultados
	var textos []string
	for i := 0; i < 4; i++ {
		result := <-results
		if result.found {
			textos = append(textos, strings.ToLower(strings.TrimSpace(result.texto)))
		}
	}

	// 4. OPTIMIZAR CONCATENACIÓN - usar strings.Builder
	var builder strings.Builder
	for _, texto := range textos {
		if texto != "" {
			builder.WriteString(texto)
			builder.WriteString(" ")
		}
	}
	textoCompleto := builder.String()

	log.Printf("🔍 Verificando página - Texto encontrado: '%s', Esperado: %s",
		strings.TrimSpace(textoCompleto), tipoSeccion)

	// 5. MAPA ESTÁTICO GLOBAL (mover fuera de la función para mejor performance)
	// Mejor práctica: definir como variable global o campo del struct
	patrones := s.obtenerPatronesSeccion(strings.ToLower(tipoSeccion))
	if len(patrones) == 0 {
		log.Printf("⚠️ Tipo de sección no reconocido: %s", tipoSeccion)
		return false
	}

	// 6. BÚSQUEDA OPTIMIZADA - salir temprano al primer match
	for _, patron := range patrones {
		if strings.Contains(textoCompleto, patron) {
			log.Printf("✅ Patrón encontrado: '%s'", patron)
			return true
		}
	}

	log.Printf("❌ Ningún patrón coincide para: %s", tipoSeccion)
	return false
}

// MOVER MAPA A MÉTODO SEPARADO O VARIABLE GLOBAL
func (s *ScraperExtendido) obtenerPatronesSeccion(tipoSeccion string) []string {
	// Usar map estático - considerar hacerlo variable global para mejor performance
	patronesPorSeccion := map[string][]string{
		"informacion_historica": {
			"información histórica de",
			"informacion historica de",
		},
		"deuda_coactiva": {
			"deuda coactiva remitida a centrales de riesgo de",
			"deuda coactiva",
		},
		"omisiones_tributarias": {
			"omisiones tributarias de",
			"omisiones tributarias",
		},
		"cantidad_trabajadores": {
			"cantidad de trabajadores",
			"número de trabajadores",
			"numero de trabajadores",
		},
		"actas_probatorias": {
			"actas probatorias de",
			"actas probatorias",
		},
		"facturas_fisicas": {
			"facturas físicas de",
			"facturas fisicas de",
		},
		"reactiva_peru": {
			"reactiva perú de",
			"reactiva peru de",
			"programa reactiva",
		},
		"garantias_covid19": {
			"programa de garantías covid",
			"programa de garantias covid",
			"garantías covid-19",
			"garantias covid-19",
		},
		"representantes_legales": {
			"representantes legales de",
		},
		"establecimientos_anexos": {
			"establecimientos anexos de",
		},
	}

	return patronesPorSeccion[tipoSeccion]
}

// ALTERNATIVA MÁS RÁPIDA - Versión ultra optimizada
func (s *ScraperExtendido) VerificarPaginaCorrectaRapida(page *rod.Page, tipoSeccion string) bool {
	// Solo esperar carga, sin sleep
	page.WaitLoad()

	// Buscar SOLO el elemento más probable primero (title)
	titulo, err := page.Timeout(2 * time.Second).Element("title")
	if err == nil {
		if texto, err := titulo.Text(); err == nil {
			textoLower := strings.ToLower(strings.TrimSpace(texto))
			if s.coincidePatron(textoLower, tipoSeccion) {
				return true
			}
		}
	}

	// Si title no coincide, buscar h1
	h1, err := page.Timeout(2 * time.Second).Element("h1")
	if err == nil {
		if texto, err := h1.Text(); err == nil {
			textoLower := strings.ToLower(strings.TrimSpace(texto))
			if s.coincidePatron(textoLower, tipoSeccion) {
				return true
			}
		}
	}

	// Solo si es necesario, buscar h2 y h3
	for _, selector := range []string{"h2", "h3"} {
		if element, err := page.Timeout(1 * time.Second).Element(selector); err == nil {
			if texto, err := element.Text(); err == nil {
				textoLower := strings.ToLower(strings.TrimSpace(texto))
				if s.coincidePatron(textoLower, tipoSeccion) {
					return true
				}
			}
		}
	}

	return false
}

func (s *ScraperExtendido) coincidePatron(texto, tipoSeccion string) bool {
	patrones := s.obtenerPatronesSeccion(strings.ToLower(tipoSeccion))
	for _, patron := range patrones {
		if strings.Contains(texto, patron) {
			return true
		}
	}
	return false
}

// VerificarErroresPagina detecta mensajes de error y maneja botones de retroceso
func (s *ScraperExtendido) VerificarErroresPagina(page *rod.Page) error {
	// Buscar mensajes de error comunes
	mensajesError := []string{
		"La aplicación ha retornado el siguiente problema",
		"The requested URL was rejected. Please consult with your administrator",
		"error en la aplicación",
		"página no encontrada",
	}

	// Verificar texto de la página
	bodyText, err := page.Timeout(3 * time.Second).Element("body")
	if err == nil {
		texto, err := bodyText.Text()
		if err == nil {
			textoLower := strings.ToLower(texto)
			for _, mensajeError := range mensajesError {
				if strings.Contains(textoLower, strings.ToLower(mensajeError)) {
					log.Printf("❌ Error detectado en página: %s", mensajeError)

					// 📌 Imprimir HTML actual antes de retroceder
					if html, err := page.HTML(); err == nil {
						log.Printf("📄 HTML actual:\n%s", html)
					} else {
						log.Printf("⚠️ No se pudo obtener HTML: %v", err)
					}

					// Intentar hacer clic en botón de retroceso
					s.intentarRetroceso(page)

					return fmt.Errorf("página retornó error: %s", mensajeError)
				}
			}
		}
	}

	return nil
}

// intentarRetroceso busca y hace clic en botones de retroceso
func (s *ScraperExtendido) intentarRetroceso(page *rod.Page) {
	// Botón tipo: <input class="form-button" type="button" value="Anterior" onclick="history.go(-1)">
	if btn, err := page.Timeout(2 * time.Second).ElementX("//input[@type='button' and @value='Anterior']"); err == nil {
		log.Println("🔙 Haciendo clic en botón 'Anterior'")
		s.HumanClick(btn, page)
		return
	}

	// Link tipo: <a href="javascript:history.back();">[Go Back]</a>
	if link, err := page.Timeout(2 * time.Second).ElementX("//a[contains(@href, 'history.back')]"); err == nil {
		log.Println("🔙 Haciendo clic en link 'Go Back'")
		s.HumanClick(link, page)
		return
	}

	// Cualquier botón que contenga "volver", "back", "anterior"
	botones := []string{
		"//button[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'volver')]",
		"//button[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'back')]",
		"//button[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'anterior')]",
	}

	for _, xpath := range botones {
		if btn, err := page.Timeout(2 * time.Second).ElementX(xpath); err == nil {
			log.Println("🔙 Haciendo clic en botón de retroceso encontrado")
			s.HumanClick(btn, page)
			return
		}
	}

	log.Println("⚠️ No se encontró botón de retroceso, usando history.back() por JavaScript")
	page.Eval("() => history.back()")
}

// ScrapeInformacionHistorica SIMPLIFICADO - retorna información de paginación
func (s *ScraperExtendido) ScrapeInformacionHistorica(ruc string, page *rod.Page) (*models.InformacionHistorica, bool, error) {
	// Buscar y hacer clic en botón
	histBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfHis')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de información histórica: %w", err)
	}

	visible, err := histBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de información histórica no está visible")
	}

	// Click humano
	err = s.HumanClick(histBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()

	var targetPage *rod.Page
	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "informacion_historica") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de información histórica - página incorrecta o no cargada")
	}

	// Extraer información
	info := &models.InformacionHistorica{}
	s.extractHistoricalInfo(targetPage, info)

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Información Histórica")
	log.Printf("🔍 Información Histórica - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	// RETORNAR CON INFORMACIÓN DE PAGINACIÓN
	return info, tienePaginacion, nil
}

// ScrapeDeudaCoactiva obtiene información de deuda coactiva
func (s *ScraperExtendido) ScrapeDeudaCoactiva(ruc string, page *rod.Page) (*models.DeudaCoactiva, bool, error) {
	// Buscar el botón usando ElementX (sin Must) con timeout
	deudaBtn, err := page.Timeout(10 * time.Second).ElementX("//button[contains(@class, 'btnInfDeuCoa')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de deuda coactiva: %w", err)
	}

	// Verificar si el botón está visible
	visible, err := deudaBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de deuda coactiva no está visible o disponible")
	}

	// Click humano
	err = s.HumanClick(deudaBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page
	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "deuda_coactiva") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de deuda coactiva - página incorrecta o no cargada")
	}

	deuda := &models.DeudaCoactiva{}
	s.extractDeudaInfo(targetPage, deuda)

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Deuda coactiva")
	log.Printf("🔍 Deuda coactiva - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return deuda, tienePaginacion, nil
}

// ScrapeRepresentantesLegales SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeRepresentantesLegales(ruc string, page *rod.Page) (*models.RepresentantesLegales, bool, error) {
	// Buscar y hacer clic en botón
	repButton, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfRepLeg')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de representantes legales: %w", err)
	}

	visible, err := repButton.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de representantes legales no está visible")
	}

	// Click humano
	err = s.HumanClick(repButton, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "representantes_legales") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de representantes legales - página incorrecta o no cargada")
	}

	// Extraer información
	html, err := targetPage.HTML()
	if err != nil {
		return nil, false, fmt.Errorf("error al obtener HTML: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, false, fmt.Errorf("error al parsear HTML: %w", err)
	}

	representantes := extraerRepresentantes(doc)
	representantesLegales := &models.RepresentantesLegales{
		Representantes: representantes,
	}

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Representantes Legales")
	log.Printf("🔍 Representantes Legales - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return representantesLegales, tienePaginacion, nil
}

// ScrapeCantidadTrabajadores SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeCantidadTrabajadores(ruc string, page *rod.Page) (*models.CantidadTrabajadores, bool, error) {
	// Buscar y hacer clic en botón
	trabBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfNumTra')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de cantidad de trabajadores: %w", err)
	}

	visible, err := trabBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de cantidad de trabajadores no está visible")
	}

	// Click humano
	err = s.HumanClick(trabBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "cantidad_trabajadores") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de cantidad de trabajadores - página incorrecta o no cargada")
	}

	// Extraer información
	cantidadTrabajadores := &models.CantidadTrabajadores{}
	s.extractTrabajadoresInfo(targetPage, cantidadTrabajadores)

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Cantidad Trabajadores")
	log.Printf("🔍 Cantidad Trabajadores - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return cantidadTrabajadores, tienePaginacion, nil
}

// ScrapeEstablecimientosAnexos SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeEstablecimientosAnexos(ruc string, page *rod.Page) (*models.EstablecimientosAnexos, bool, error) {
	// Buscar y hacer clic en botón
	estabBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfLocAnex')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de establecimientos anexos: %w", err)
	}

	visible, err := estabBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de establecimientos anexos no está visible")
	}

	// Click humano
	err = s.HumanClick(estabBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "establecimientos_anexos") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de establecimientos anexos - página incorrecta o no cargada")
	}

	// Extraer información
	establecimientosAnexos := &models.EstablecimientosAnexos{}
	err = s.extractEstablecimientosInfo(targetPage, establecimientosAnexos)
	if err != nil {
		return nil, false, fmt.Errorf("error al extraer información de establecimientos: %w", err)
	}

	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Establemientos anexos")
	log.Printf("🔍 Establemientos anexos - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return establecimientosAnexos, tienePaginacion, nil
}

// Métodos adicionales para las demás consultas...

// ScrapeOmisionesTributarias SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeOmisionesTributarias(ruc string, page *rod.Page) (*models.OmisionesTributarias, bool, error) {
	// Buscar y hacer clic en botón
	omisBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfOmiTri')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de omisiones tributarias: %w", err)
	}

	visible, err := omisBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de omisiones tributarias no está visible")
	}

	// Click humano
	err = s.HumanClick(omisBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "omisiones_tributarias") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de omisiones tributarias - página incorrecta o no cargada")
	}

	// Extraer información
	omisionesTributarias := &models.OmisionesTributarias{}
	s.extractOmisionesInfo(targetPage, omisionesTributarias)

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Omisiones Tributarias")
	log.Printf("🔍 Omisiones tributarias - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return omisionesTributarias, tienePaginacion, nil
}

// ScrapeActasProbatorias SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeActasProbatorias(ruc string, page *rod.Page) (*models.ActasProbatorias, bool, error) {
	// Buscar y hacer clic en botón
	actasBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfActPro')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de actas probatorias: %w", err)
	}

	visible, err := actasBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de actas probatorias no está visible")
	}

	// Click humano
	err = s.HumanClick(actasBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "actas_probatorias") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de actas probatorias - página incorrecta o no cargada")
	}

	// Extraer información
	actasProbatorias := &models.ActasProbatorias{}
	s.extractActasProbatoriasInfo(targetPage, actasProbatorias)

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Actas Probatorias")
	log.Printf("🔍 Actas probatorias - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return actasProbatorias, tienePaginacion, nil
}

// ScrapeFacturasFisicas SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeFacturasFisicas(ruc string, page *rod.Page) (*models.FacturasFisicas, bool, error) {
	// Buscar y hacer clic en botón
	facturasBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfActCPF')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón de facturas físicas: %w", err)
	}

	visible, err := facturasBtn.Visible()
	if err != nil || !visible {
		return nil, false, fmt.Errorf("el botón de facturas físicas no está visible")
	}

	// Click humano
	err = s.HumanClick(facturasBtn, page)
	if err != nil {
		return nil, false, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "facturas_fisicas") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, false, fmt.Errorf("no se pudo acceder a la página de facturas físicas - página incorrecta o no cargada")
	}

	// Extraer información
	facturasFisicas := &models.FacturasFisicas{}
	s.extractFacturasFisicasInfo(targetPage, facturasFisicas)

	// DETECTAR PAGINACIÓN
	tienePaginacion, contexto := s.DetectarPaginacionConContexto(targetPage, "Facturas fisicas")
	log.Printf("🔍 Facturas fisicas - Paginación: %v (%s)", tienePaginacion, contexto)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, false, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, false, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return facturasFisicas, tienePaginacion, nil
}

// ScrapeReactivaPeru SIMPLIFICADO - solo lo esencial
func (s *ScraperExtendido) ScrapeReactivaPeru(ruc string, page *rod.Page) (*models.ReactivaPeru, error) {
	// Buscar y hacer clic en botón
	reactivaBtn, err := page.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnInfReaPer')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón de Reactiva Perú: %w", err)
	}

	visible, err := reactivaBtn.Visible()
	if err != nil || !visible {
		return nil, fmt.Errorf("el botón de Reactiva Perú no está visible")
	}

	// Click humano
	err = s.HumanClick(reactivaBtn, page)
	if err != nil {
		return nil, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "reactiva_peru") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, fmt.Errorf("no se pudo acceder a la página de reactiva perú - página incorrecta o no cargada")
	}

	// Extraer información
	reactivaPeru := &models.ReactivaPeru{}
	s.extractReactivaPeruInfo(targetPage, reactivaPeru)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return reactivaPeru, nil
}

// ScrapeProgramaCovid19 obtiene información de programas COVID-19
func (s *ScraperExtendido) ScrapeProgramaCovid19(ruc string, page *rod.Page) (*models.ProgramaCovid19, error) {

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

	// Click humano
	err = s.HumanClick(covidBtn, page)
	if err != nil {
		return nil, fmt.Errorf("error en click humano: %w", err)
	}

	// Esperar respuesta y detectar si se abrió nueva pestaña
	time.Sleep(2 * time.Second)
	pages := s.browser.MustPages()
	var targetPage *rod.Page

	if len(pages) > 1 {
		// Nueva pestaña
		targetPage = pages[len(pages)-1]
		targetPage.MustActivate()
	} else {
		// Misma pestaña
		targetPage = page
	}

	// Carga humana
	err = s.HumanPageLoad(targetPage)
	if err != nil {
		log.Printf("⚠️ Warning: Error en carga humana: %v", err)
	}

	// ✅ AÑADIR AQUÍ - Verificar errores de página ANTES de validar contenido
	if errorPagina := s.VerificarErroresPagina(targetPage); errorPagina != nil {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, errorPagina
	}

	// VALIDAR QUE ESTAMOS EN LA PÁGINA CORRECTA
	if !s.VerificarPaginaCorrecta(targetPage, "garantias_covid19") {
		// Cleanup si era nueva pestaña
		if targetPage != page {
			targetPage.MustClose()
			page.MustActivate()
		}
		return nil, fmt.Errorf("no se pudo acceder a la página de programa covid-19 - página incorrecta o no cargada")
	}

	programaCovid := &models.ProgramaCovid19{}
	// Extraer información del Programa COVID-19
	s.extractProgramaCovid19Info(page, programaCovid)

	// Buscar y hacer clic en volver
	volver, err := targetPage.Timeout(8 * time.Second).ElementX("//button[contains(@class, 'btnNuevaConsulta')]")
	if err != nil {
		return nil, fmt.Errorf("no se encontró el botón volver: %w", err)
	}

	err = s.HumanClick(volver, targetPage)
	if err != nil {
		return nil, fmt.Errorf("error en click volver: %w", err)
	}

	// Cleanup si era nueva pestaña
	if targetPage != page {
		targetPage.MustClose()
		time.Sleep(500 * time.Millisecond)
		page.MustActivate()
	}

	page.WaitLoad()
	page.WaitStable(2 * time.Second)

	return programaCovid, nil
}
