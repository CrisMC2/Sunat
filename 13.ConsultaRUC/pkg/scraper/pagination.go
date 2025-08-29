package scraper

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

func (s *ScraperExtendido) DetectarPaginacion(page *rod.Page) bool {
	// DETECTORES ESPECÍFICOS PARA SUNAT - MEJORADOS

	// 1. Buscar el patrón más específico de SUNAT: "Páginas:" seguido de enlaces
	paginasElement, err := page.Timeout(5 * time.Second).ElementX("//td[contains(text(), 'Páginas:')]")
	if err == nil && paginasElement != nil {
		visible, err := paginasElement.Visible()
		if err == nil && visible {
			log.Printf("✅ Detectada paginación SUNAT: elemento 'Páginas:' encontrado y visible")
			return true
		}
	}

	// 2. Buscar enlaces con javascript:paginacion() - MÁS ESPECÍFICO
	paginacionLinks, err := page.Timeout(5 * time.Second).ElementsX("//a[contains(@href, 'javascript:paginacion')]")
	if err == nil && len(paginacionLinks) > 0 {
		// Verificar que al menos uno sea visible
		for _, link := range paginacionLinks {
			if visible, err := link.Visible(); err == nil && visible {
				log.Printf("✅ Detectada paginación SUNAT: %d enlaces con javascript:paginacion() visibles", len(paginacionLinks))
				return true
			}
		}
	}

	// 3. Buscar patrón específico de SUNAT: "1 a X de Y" - CORREGIDO
	patronResultados, err := page.Timeout(3 * time.Second).ElementX("//td[contains(text(), ' a ') and contains(text(), ' de ')]")
	if err == nil && patronResultados != nil {
		visible, err := patronResultados.Visible()
		if err == nil && visible {
			texto, _ := patronResultados.Text()
			// Verificar que sigue el patrón "X a Y de Z"
			if matched, _ := regexp.MatchString(`\d+\s+a\s+\d+\s+de\s+\d+`, texto); matched {
				// NUEVA VALIDACIÓN: Solo considerar paginación si hay más de 1 página
				// Extraer el total de resultados
				re := regexp.MustCompile(`(\d+)\s+a\s+(\d+)\s+de\s+(\d+)`)
				matches := re.FindStringSubmatch(texto)
				if len(matches) == 4 {
					totalResultados, _ := strconv.Atoi(matches[3])
					resultadosPorPagina, _ := strconv.Atoi(matches[2])

					// Solo hay paginación si el total es mayor que los resultados mostrados
					if totalResultados > resultadosPorPagina {
						log.Printf("✅ Detectada paginación SUNAT: patrón de resultados '%s' (%d total, %d por página)", texto, totalResultados, resultadosPorPagina)
						return true
					} else {
						log.Printf("📄 Sin paginación SUNAT: solo una página '%s'", texto)
					}
				}
			}
		}
	}

	// 4. Buscar combinación de "Páginas:" + enlaces numerados + "Siguiente"
	siguienteLink, err1 := page.Timeout(3 * time.Second).ElementX("//a[contains(text(), 'Siguiente')]")
	numerosLinks, err2 := page.Timeout(3 * time.Second).ElementsX("//a[contains(@href, 'javascript:paginacion')]")

	if err1 == nil && err2 == nil && siguienteLink != nil && len(numerosLinks) > 0 {
		visible1, _ := siguienteLink.Visible()
		if visible1 {
			log.Printf("✅ Detectada paginación SUNAT: 'Siguiente' + %d enlaces numerados", len(numerosLinks))
			return true
		}
	}

	// 5. NUEVO: Buscar tabla con estructura específica de paginación SUNAT
	tablaPaginacion, err := page.Timeout(3 * time.Second).ElementX("//table[.//td[contains(text(), 'Páginas:')]]")
	if err == nil && tablaPaginacion != nil {
		visible, err := tablaPaginacion.Visible()
		if err == nil && visible {
			log.Printf("✅ Detectada paginación SUNAT: tabla de paginación encontrada")
			return true
		}
	}

	// Selectores adicionales para otros sistemas
	selectoresPaginacionEspecificos := []string{
		".pagination",
		".pager",
		".page-navigation",
		".gridview-pager",
		".dataTables_paginate",
		"ul.pagination",
		"div.pagination",
		"[role='navigation'][aria-label*='pag']",
		"[class='paging']",
		"[id*='DataPager']",
		"[id*='GridPager']",
		"//a[contains(@href, 'page=') or contains(@href, 'pagina=') or contains(@href, 'offset=')]",
	}

	// Verificar selectores específicos con timeout más corto
	for _, selector := range selectoresPaginacionEspecificos {
		var element *rod.Element
		var err error

		if strings.HasPrefix(selector, "//") {
			element, err = page.Timeout(2 * time.Second).ElementX(selector)
		} else {
			element, err = page.Timeout(2 * time.Second).Element(selector)
		}

		if err == nil && element != nil {
			visible, err := element.Visible()
			if err == nil && visible {
				log.Printf("✅ Detectada paginación genérica: selector %s", selector)
				return true
			}
		}
	}

	// Búsqueda por texto con patrones MUY específicos
	pageText, err := page.MustElement("body").Text()
	if err == nil {
		// PATRONES ESPECÍFICOS DE SUNAT - CON VALIDACIÓN MEJORADA
		if strings.Contains(pageText, "Páginas:") {
			// Verificar también que no sea "1 a 1 de 1"
			re := regexp.MustCompile(`(\d+)\s+a\s+(\d+)\s+de\s+(\d+)`)
			matches := re.FindStringSubmatch(pageText)
			if len(matches) == 4 {
				totalResultados, _ := strconv.Atoi(matches[3])
				resultadosPorPagina, _ := strconv.Atoi(matches[2])
				if totalResultados > resultadosPorPagina {
					log.Printf("✅ Detectada paginación SUNAT: texto 'Páginas:' encontrado en body con múltiples páginas")
					return true
				} else {
					log.Printf("📄 Sin paginación SUNAT: 'Páginas:' encontrado pero solo una página (%d de %d)", resultadosPorPagina, totalResultados)
				}
			} else {
				// Si no puede extraer números, asumir que hay paginación (caso edge)
				log.Printf("✅ Detectada paginación SUNAT: texto 'Páginas:' encontrado en body")
				return true
			}
		}

		// Patrones muy específicos que indican paginación real - SIN EL PATRÓN PROBLEMÁTICO
		patronesPaginacionEspecificos := []string{
			`Página\s+\d+\s+de\s+\d+`,
			`Mostrando\s+\d+\s*[-–]\s*\d+\s+de\s+\d+`,
			`Registros\s+\d+\s+al\s+\d+\s+de\s+\d+`,
			`\d+\s+to\s+\d+\s+of\s+\d+`,
			`Página\s+(anterior|siguiente)`,
			`(Primera|Última)\s+página`,
			`Ir\s+a\s+página`,
		}

		for _, patron := range patronesPaginacionEspecificos {
			matched, err := regexp.MatchString("(?i)"+patron, pageText)
			if err == nil && matched {
				// VALIDACIÓN ADICIONAL: Para patrones que pueden incluir "1 a 1 de 1"
				if strings.Contains(patron, `\d+\s*[-–]\s*\d+\s+de\s+\d+`) || strings.Contains(patron, `\d+\s+al\s+\d+\s+de\s+\d+`) || strings.Contains(patron, `\d+\s+to\s+\d+\s+of\s+\d+`) {
					// Extraer números para validar que no sea una sola página
					re := regexp.MustCompile(`(\d+).*?(\d+).*?(\d+)`)
					matches := re.FindStringSubmatch(pageText)
					if len(matches) >= 4 {
						inicio, _ := strconv.Atoi(matches[1])
						fin, _ := strconv.Atoi(matches[2])
						total, _ := strconv.Atoi(matches[3])
						if total > fin || (fin-inicio) > 0 {
							log.Printf("✅ Detectada paginación por patrón de texto validado: %s", patron)
							return true
						} else {
							log.Printf("📄 Patrón encontrado pero es una sola página: %s", patron)
							continue
						}
					}
				}
				log.Printf("✅ Detectada paginación por patrón de texto: %s", patron)
				return true
			}
		}
	}

	// Verificación adicional: buscar múltiples enlaces numerados (típico de paginación)
	numerosLinks, err = page.Timeout(3 * time.Second).ElementsX("//a[text() >= '1' and text() <= '99'] | //button[text() >= '1' and text() <= '99']")
	if err == nil && len(numerosLinks) >= 3 {
		// Verificar que al menos algunos sean visibles
		visibleCount := 0
		for _, link := range numerosLinks {
			if visible, _ := link.Visible(); visible {
				visibleCount++
			}
		}
		if visibleCount >= 3 {
			log.Printf("✅ Detectada paginación: %d enlaces numerados visibles", visibleCount)
			return true
		}
	}

	// Buscar botones "Siguiente" y "Anterior" juntos
	hasSiguiente := false
	hasAnterior := false

	siguiente, err1 := page.Timeout(2 * time.Second).ElementX("//button[contains(text(), 'Siguiente') or contains(text(), 'Next')] | //a[contains(text(), 'Siguiente') or contains(text(), 'Next')]")
	anterior, err2 := page.Timeout(2 * time.Second).ElementX("//button[contains(text(), 'Anterior') or contains(text(), 'Previous') or contains(text(), 'Prev')] | //a[contains(text(), 'Anterior') or contains(text(), 'Previous') or contains(text(), 'Prev')]")

	if err1 == nil && siguiente != nil {
		if visible, _ := siguiente.Visible(); visible {
			hasSiguiente = true
		}
	}

	if err2 == nil && anterior != nil {
		if visible, _ := anterior.Visible(); visible {
			hasAnterior = true
		}
	}

	// Si tiene ambos botones, muy probablemente sea paginación
	if hasSiguiente && hasAnterior {
		log.Printf("✅ Detectada paginación: botones Siguiente y Anterior presentes")
		return true
	}

	// Si solo tiene "Siguiente", también puede ser paginación (primera página)
	if hasSiguiente {
		// Buscar indicios adicionales de que es primera página - CON VALIDACIÓN MEJORADA
		pageText, err := page.MustElement("body").Text()
		if err == nil {
			if strings.Contains(pageText, "Páginas:") {
				// Validar que no sea "1 a 1 de 1"
				re := regexp.MustCompile(`(\d+)\s+a\s+(\d+)\s+de\s+(\d+)`)
				matches := re.FindStringSubmatch(pageText)
				if len(matches) == 4 {
					totalResultados, _ := strconv.Atoi(matches[3])
					resultadosPorPagina, _ := strconv.Atoi(matches[2])
					if totalResultados > resultadosPorPagina {
						log.Printf("✅ Detectada paginación: botón Siguiente + indicadores de primera página válidos")
						return true
					}
				} else {
					log.Printf("✅ Detectada paginación: botón Siguiente + indicadores de primera página")
					return true
				}
			}
		}
	}

	log.Printf("❌ No se detectó paginación")
	return false
}

// Nueva función para debugging específico y detallado
func (s *ScraperExtendido) DebugPaginacionDetallado(page *rod.Page) {
	log.Printf("🔍 === DEBUG DETECCIÓN DE PAGINACIÓN ===")

	// 1. Verificar elemento "Páginas:"
	paginasElement, err1 := page.ElementX("//td[contains(text(), 'Páginas:')]")
	log.Printf("1. Elemento 'Páginas:': existe=%v, error=%v", paginasElement != nil, err1)
	if paginasElement != nil {
		visible, _ := paginasElement.Visible()
		text, _ := paginasElement.Text()
		log.Printf("   - Visible: %v, Texto: '%s'", visible, text)
	}

	// 2. Verificar enlaces javascript:paginacion
	paginacionLinks, err2 := page.ElementsX("//a[contains(@href, 'javascript:paginacion')]")
	log.Printf("2. Links javascript:paginacion: cantidad=%d, error=%v", len(paginacionLinks), err2)
	for i, link := range paginacionLinks {
		if i < 5 { // Solo mostrar los primeros 5
			href, _ := link.Attribute("href")
			text, _ := link.Text()
			visible, _ := link.Visible()
			log.Printf("   - Link %d: href='%s', texto='%s', visible=%v", i+1, *href, text, visible)
		}
	}

	// 3. Verificar patrón "X a Y de Z"
	patronResultados, err3 := page.ElementX("//td[contains(text(), ' a ') and contains(text(), ' de ')]")
	log.Printf("3. Patrón resultados: existe=%v, error=%v", patronResultados != nil, err3)
	if patronResultados != nil {
		text, _ := patronResultados.Text()
		visible, _ := patronResultados.Visible()
		log.Printf("   - Texto: '%s', Visible: %v", text, visible)
	}

	// 4. Verificar botón "Siguiente"
	siguienteLink, err4 := page.ElementX("//a[contains(text(), 'Siguiente')]")
	log.Printf("4. Botón Siguiente: existe=%v, error=%v", siguienteLink != nil, err4)
	if siguienteLink != nil {
		href, _ := siguienteLink.Attribute("href")
		visible, _ := siguienteLink.Visible()
		log.Printf("   - Href: '%s', Visible: %v", *href, visible)
	}

	// 5. Verificar HTML completo de paginación
	tablaPaginacion, err5 := page.ElementX("//table[.//td[contains(text(), 'Páginas:')]]")
	log.Printf("5. Tabla paginación: existe=%v, error=%v", tablaPaginacion != nil, err5)
	if tablaPaginacion != nil {
		html, _ := tablaPaginacion.HTML()
		visible, _ := tablaPaginacion.Visible()
		log.Printf("   - Visible: %v", visible)
		log.Printf("   - HTML: %s", html[:min(200, len(html))])
	}

	// 6. Búsqueda en texto de la página
	pageText, err := page.MustElement("body").Text()
	if err == nil {
		hasPages := strings.Contains(pageText, "Páginas:")
		log.Printf("6. Texto página contiene 'Páginas:': %v", hasPages)

		// Buscar patrón específico
		patron := regexp.MustCompile(`\d+\s+a\s+\d+\s+de\s+\d+`)
		matches := patron.FindAllString(pageText, -1)
		log.Printf("   - Patrones 'X a Y de Z' encontrados: %v", matches)
	}

	log.Printf("🔍 === FIN DEBUG DETECCIÓN ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DetectarPaginacionConContexto detecta paginación y proporciona información adicional
func (s *ScraperExtendido) DetectarPaginacionConContexto(page *rod.Page, seccion string) (bool, string) {
	tienePaginacion := s.DetectarPaginacion(page)

	if !tienePaginacion {
		return false, "No se detectó paginación"
	}

	// Intentar obtener información específica sobre la paginación
	contexto := ""

	// INFORMACIÓN ESPECÍFICA DE SUNAT
	paginacionLinks, err := page.ElementsX("//a[contains(@href, 'javascript:paginacion')]")
	if err == nil && len(paginacionLinks) > 0 {
		// Obtener el número de la última página
		ultimaPagina := 1
		for _, link := range paginacionLinks {
			texto, err := link.Text()
			if err == nil {
				// Extraer número de la página (formato "2 | " o "3 | ")
				numeroStr := strings.TrimSpace(strings.Replace(texto, "|", "", -1))
				if numero, err := strconv.Atoi(numeroStr); err == nil && numero > ultimaPagina {
					ultimaPagina = numero
				}
			}
		}
		contexto = fmt.Sprintf("SUNAT - %d páginas disponibles (página actual: 1)", ultimaPagina+1)
	}

	// Si no es SUNAT, usar patrones generales
	if contexto == "" {
		pageText, err := page.MustElement("body").Text()
		if err == nil {
			// Verificar si contiene "Páginas:"
			if strings.Contains(pageText, "Páginas:") {
				contexto = "SUNAT - Paginación detectada"
			} else {
				// Patrones para extraer información específica de paginación
				patrones := map[string]*regexp.Regexp{
					"pagina_de":   regexp.MustCompile(`(?i)Página\s+(\d+)\s+de\s+(\d+)`),
					"mostrando":   regexp.MustCompile(`(?i)Mostrando\s+(\d+)\s*[-–]\s*(\d+)\s+de\s+(\d+)`),
					"registros":   regexp.MustCompile(`(?i)Registros\s+(\d+)\s+al\s+(\d+)\s+de\s+(\d+)`),
					"total_items": regexp.MustCompile(`(?i)(\d+)\s+to\s+(\d+)\s+of\s+(\d+)`),
				}

				for nombre, patron := range patrones {
					if matches := patron.FindStringSubmatch(pageText); len(matches) > 1 {
						switch nombre {
						case "pagina_de":
							contexto = fmt.Sprintf("Página %s de %s", matches[1], matches[2])
						case "mostrando", "registros":
							if len(matches) > 3 {
								contexto = fmt.Sprintf("Mostrando %s-%s de %s registros", matches[1], matches[2], matches[3])
							}
						case "total_items":
							if len(matches) > 3 {
								contexto = fmt.Sprintf("Items %s-%s de %s", matches[1], matches[2], matches[3])
							}
						}
						break
					}
				}
			}
		}
	}

	// Contar enlaces numerados para dar más contexto
	if contexto == "" {
		numerosLinks, err := page.ElementsX("//a[text() >= '1' and text() <= '99'] | //button[text() >= '1' and text() <= '99']")
		if err == nil && len(numerosLinks) > 0 {
			contexto = fmt.Sprintf("Paginación con %d páginas numeradas", len(numerosLinks))
		} else {
			contexto = "Controles de paginación detectados"
		}
	}

	return true, contexto
}

// ValidarPaginacionEnSeccion - función auxiliar para validar en secciones específicas
func (s *ScraperExtendido) ValidarPaginacionEnSeccion(page *rod.Page, seccion string) (bool, string) {
	// Log para debugging
	log.Printf("🔍 Verificando paginación en sección: %s", seccion)

	// Obtener el HTML de la página para análisis manual si es necesario
	pageHTML, err := page.HTML()
	if err == nil {
		// Log del tamaño del HTML para verificar que tenemos contenido
		log.Printf("📄 Tamaño del HTML: %d caracteres", len(pageHTML))
	}

	return s.DetectarPaginacionConContexto(page, seccion)
}

// DebugPaginacionSUNAT - función para debugging específico
func (s *ScraperExtendido) DebugPaginacionSUNAT(page *rod.Page) {
	// Buscar elementos específicos
	paginasElement, err1 := page.ElementX("//td[contains(text(), 'Páginas:')]")
	paginacionLinks, err2 := page.ElementsX("//a[contains(@href, 'javascript:paginacion')]")
	siguienteLink, err3 := page.ElementX("//a[contains(text(), 'Siguiente')]")

	log.Printf("🔍 DEBUG PAGINACIÓN SUNAT:")
	log.Printf("   - Elemento 'Páginas:': %v (error: %v)", paginasElement != nil, err1)
	log.Printf("   - Links javascript:paginacion: %d (error: %v)", len(paginacionLinks), err2)
	log.Printf("   - Link 'Siguiente': %v (error: %v)", siguienteLink != nil, err3)

	// Mostrar HTML de la tabla de paginación si existe
	tablaElement, err := page.ElementX("//table[.//td[contains(text(), 'Páginas:')]]")
	if err == nil && tablaElement != nil {
		html, _ := tablaElement.HTML()
		log.Printf("   - HTML tabla paginación: %s", html)
	}
}
