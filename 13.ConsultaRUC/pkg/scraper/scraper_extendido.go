package scraper

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/consulta-ruc-scraper/pkg/database"
	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/utils"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

// HumanBehaviorSimulator simula comportamiento humano avanzado
type HumanBehaviorSimulator struct {
	startTime      time.Time
	actionCount    int
	fatigueLevel   float64
	lastActionTime time.Time
	userAgents     []string
	currentUAIndex int
	readingSpeed   float64 // palabras por minuto
	typingSpeed    float64 // caracteres por minuto
}

// ScraperExtendido incluye métodos para todas las consultas adicionales
type ScraperExtendido struct {
	*SUNATScraper
	humanSim *HumanBehaviorSimulator
}

// NewScraperExtendido crea una nueva instancia del scraper extendido
func NewScraperExtendido() (*ScraperExtendido, error) {
	base, err := NewSUNATScraper()
	if err != nil {
		return nil, err
	}

	// Inicializar simulador de comportamiento humano
	humanSim := &HumanBehaviorSimulator{
		startTime:    time.Now(),
		actionCount:  0,
		fatigueLevel: 0.0,
		userAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		},
		currentUAIndex: rand.Intn(5),
		readingSpeed:   200 + rand.Float64()*100, // 200-300 WPM
		typingSpeed:    180 + rand.Float64()*120, // 180-300 CPM (3-5 CPS)
	}

	return &ScraperExtendido{
		SUNATScraper: base,
		humanSim:     humanSim,
	}, nil
}

// updateFatigue actualiza el nivel de fatiga basado en actividad
func (h *HumanBehaviorSimulator) updateFatigue() {
	h.actionCount++
	elapsed := time.Since(h.startTime).Minutes()

	// Fatiga aumenta con número de acciones y tiempo transcurrido
	h.fatigueLevel = math.Min(1.0, float64(h.actionCount)*0.002+elapsed*0.01)
}

// shouldTakeBreak determina si debe tomar un descanso
func (h *HumanBehaviorSimulator) shouldTakeBreak() bool {
	// Descanso basado en tiempo desde última acción
	timeSinceLastAction := time.Since(h.lastActionTime)

	// Probabilidad de descanso aumenta con fatiga y actividad reciente
	if h.actionCount > 0 && h.actionCount%15 == 0 { // Cada 15 acciones
		return true
	}

	if h.fatigueLevel > 0.7 && rand.Float64() < 0.3 {
		return true
	}

	if timeSinceLastAction < 500*time.Millisecond && rand.Float64() < 0.1 {
		return true
	}

	return false
}

// takeBreak simula un descanso humano
func (h *HumanBehaviorSimulator) takeBreak() {
	breakType := rand.Intn(4)
	var breakDuration time.Duration

	switch breakType {
	case 0: // Micro-pausa
		breakDuration = h.generateLogNormalDelay(800, 300)
		log.Printf(" 😴 Micro-pausa: %v", breakDuration)
	case 1: // Pausa corta
		breakDuration = h.generateLogNormalDelay(2000, 800)
		log.Printf(" ☕ Pausa corta: %v", breakDuration)
	case 2: // Pausa media
		breakDuration = h.generateLogNormalDelay(5000, 2000)
		log.Printf(" 🚶 Pausa media: %v", breakDuration)
	case 3: // Pausa larga (fatiga alta)
		if h.fatigueLevel > 0.8 {
			breakDuration = h.generateLogNormalDelay(10000, 5000)
			log.Printf(" 🛋️  Pausa larga: %v", breakDuration)
			h.fatigueLevel *= 0.7 // Reduce fatiga después de pausa larga
		} else {
			breakDuration = h.generateLogNormalDelay(1500, 500)
		}
	}

	time.Sleep(breakDuration)
}

// HumanClick CORREGIDO - múltiples errores solucionados
func (s *ScraperExtendido) HumanClick(element *rod.Element, page *rod.Page) error {
	// Actualizar fatiga y verificar descansos
	s.humanSim.updateFatigue()

	if s.humanSim.shouldTakeBreak() {
		s.humanSim.takeBreak()
	}

	// Rotar user agent ocasionalmente
	s.humanSim.rotateUserAgent(page)

	// Delay pre-acción con distribución log-normal
	preDelay := s.humanSim.generateLogNormalDelay(200, 80)
	time.Sleep(preDelay)

	// VERIFICAR QUE EL ELEMENTO SIGUE SIENDO VÁLIDO
	if element == nil {
		return fmt.Errorf("elemento es nil")
	}

	// Verificar que el elemento sigue visible y clickeable
	visible, err := element.Visible()
	if err != nil || !visible {
		return fmt.Errorf("elemento ya no es visible: %w", err)
	}

	// Scroll al elemento si es necesario
	err = element.ScrollIntoView()
	if err != nil {
		log.Printf("⚠️ Warning: No se pudo hacer scroll al elemento: %v", err)
	}

	// Esperar un poco después del scroll
	time.Sleep(200 * time.Millisecond)

	// Obtener coordenadas del elemento (CON VALIDACIÓN)
	box, err := element.Shape()
	if err != nil {
		return fmt.Errorf("error obteniendo coordenadas del elemento: %w", err)
	}

	if len(box.Quads) == 0 {
		return fmt.Errorf("elemento no tiene coordenadas válidas")
	}

	// Calcular centro del elemento con variación humana realista
	quad := box.Quads[0]
	if len(quad) < 8 {
		return fmt.Errorf("coordenadas del quad incompletas")
	}

	centerX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	centerY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	// VALIDAR COORDENADAS
	if centerX <= 0 || centerY <= 0 {
		return fmt.Errorf("coordenadas inválidas: x=%.2f, y=%.2f", centerX, centerY)
	}

	// Variación basada en fatiga (más imprecisión cuando está cansado)
	maxOffset := 5.0 + s.humanSim.fatigueLevel*3.0 // REDUCIR offset para más precisión
	offsetX := (rand.Float64() - 0.5) * maxOffset
	offsetY := (rand.Float64() - 0.5) * maxOffset

	targetX := centerX + offsetX
	targetY := centerY + offsetY

	// Movimiento de mouse más simple y confiable
	steps := 2 + rand.Intn(3) // Reducir pasos para evitar problemas

	for i := 0; i < steps; i++ {
		progress := float64(i+1) / float64(steps)

		// Movimiento lineal simple (más confiable que Bézier)
		intermediateX := centerX + (targetX-centerX)*progress
		intermediateY := centerY + (targetY-centerY)*progress

		// Mover mouse CON VALIDACIÓN
		err = page.Mouse.MoveTo(proto.Point{X: intermediateX, Y: intermediateY})
		if err != nil {
			log.Printf("⚠️ Warning: Error moviendo mouse: %v", err)
			// Continuar anyway
		}

		// Delay más corto entre movimientos
		moveDelay := time.Duration(20+rand.Intn(30)) * time.Millisecond
		time.Sleep(moveDelay)
	}

	// Tiempo de reacción humano antes del clic (MÁS CORTO)
	reactionTime := time.Duration(100+rand.Intn(100)) * time.Millisecond
	time.Sleep(reactionTime)

	// CLICK SIMPLE Y CONFIABLE
	err = page.Mouse.MoveTo(proto.Point{X: targetX, Y: targetY})
	if err != nil {
		log.Printf("⚠️ Warning: Error moviendo mouse antes del click: %v", err)
	}
	err = page.Mouse.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		// Fallback: usar el método del elemento directamente
		log.Printf("⚠️ Warning: Click de mouse falló, usando elemento.Click(): %v", err)
		err = element.Click(proto.InputMouseButtonLeft, 1)
		if err != nil {
			return fmt.Errorf("error en ambos métodos de click: %w", err)
		}
	}

	// Actualizar tiempo de última acción
	s.humanSim.lastActionTime = time.Now()

	// Delay post-clic (MÁS CORTO)
	postDelay := time.Duration(200+rand.Intn(200)) * time.Millisecond
	time.Sleep(postDelay)

	log.Printf(" 🖱️ Click humano exitoso en (%.1f, %.1f)", targetX, targetY)
	return nil
}

// HumanPageLoad CORREGIDO - timeouts más razonables
func (s *ScraperExtendido) HumanPageLoad(page *rod.Page) error {
	// Esperar carga técnica CON TIMEOUT
	loadCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err := page.Context(loadCtx).WaitLoad()
	if err != nil {
		log.Printf("⚠️ Warning: WaitLoad falló: %v", err)
		// Continuar anyway, a veces la página ya está cargada
	}

	// WaitStable con timeout más corto
	stableCtx, cancel2 := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel2()

	err = page.Context(stableCtx).WaitStable(3 * time.Second)
	if err != nil {
		log.Printf("⚠️ Warning: WaitStable falló: %v", err)
		// Continuar anyway
	}

	// Simular tiempo de lectura del contenido visible (REDUCIDO)
	bodyText := ""
	if body, err := page.Timeout(5 * time.Second).Element("body"); err == nil {
		bodyText, _ = body.Text()
	}

	readingTime := s.humanSim.simulateReading(bodyText)

	// LIMITAR tiempo de lectura máximo (MÁS AGRESIVO)
	maxReadingTime := 5 * time.Second
	if readingTime > maxReadingTime {
		readingTime = maxReadingTime
	}

	// MÍNIMO tiempo de lectura
	minReadingTime := 500 * time.Millisecond
	if readingTime < minReadingTime {
		readingTime = minReadingTime
	}

	log.Printf(" 📖 Simulando lectura de página: %v", readingTime)
	time.Sleep(readingTime)

	return nil
}

// rotateUserAgent CORREGIDO - manejo de errores
func (h *HumanBehaviorSimulator) rotateUserAgent(page *rod.Page) {
	// Cambiar user agent ocasionalmente (2% de probabilidad - menos frecuente)
	if rand.Float64() < 0.02 {
		h.currentUAIndex = (h.currentUAIndex + 1) % len(h.userAgents)
		newUA := h.userAgents[h.currentUAIndex]

		// Cambiar UA con manejo de errores
		_, err := page.Eval(fmt.Sprintf(`
			try {
				Object.defineProperty(navigator, 'userAgent', {
					get: function() { return '%s'; }
				});
				'success';
			} catch(e) {
				'error: ' + e.message;
			}
		`, newUA))

		if err != nil {
			log.Printf("⚠️ Warning: Error rotando user agent: %v", err)
		} else {
			log.Printf(" 🔄 User agent rotado")
		}
	}
}

// simulateReading CORREGIDO - más conservador
func (h *HumanBehaviorSimulator) simulateReading(text string) time.Duration {
	words := len(strings.Fields(text))
	if words == 0 {
		return 300 * time.Millisecond // Mínimo para elementos vacíos
	}

	// Límite de palabras para evitar tiempos excesivos
	if words > 500 {
		words = 500
	}

	// Tiempo base de lectura más rápido
	readingTimeMs := float64(words) / h.readingSpeed * 60000

	// Reducir tiempo de procesamiento
	processingTime := 50 + rand.Float64()*100

	totalTime := readingTimeMs + processingTime

	// Limitar tiempo total
	if totalTime > 5000 {
		totalTime = 5000
	}

	return time.Duration(totalTime) * time.Millisecond
}

// generateLogNormalDelay CORREGIDO - valores más conservadores
func (h *HumanBehaviorSimulator) generateLogNormalDelay(meanMs, stdMs float64) time.Duration {
	// Usar distribución normal simple para más predictibilidad
	normal := rand.NormFloat64()
	result := meanMs + stdMs*normal

	// Aplicar factor de fatiga reducido
	fatigueMultiplier := 1.0 + h.fatigueLevel*0.1
	result = result * fatigueMultiplier

	// Limitar valores extremos más agresivamente
	if result < 20 {
		result = 20
	}
	if result > 2000 {
		result = 2000
	}

	return time.Duration(result) * time.Millisecond
}

// HumanInput simula escritura humana avanzada con características realistas
func (s *ScraperExtendido) HumanInput(element *rod.Element, text string) error {
	// Limpiar campo con delay humano
	err := element.SelectAllText()
	if err == nil {
		err = element.Input("")
	}
	if err != nil {
		return fmt.Errorf("error limpiando campo: %w", err)
	}

	// Delay inicial antes de empezar a escribir
	startDelay := s.humanSim.generateLogNormalDelay(400, 200)
	time.Sleep(startDelay)

	// Calcular velocidad de escritura base (variable por fatiga)
	baseSpeed := s.humanSim.typingSpeed * (1.0 - s.humanSim.fatigueLevel*0.3)

	for i, char := range text {
		// Escribir carácter
		err = element.Input(string(char))
		if err != nil {
			return fmt.Errorf("error escribiendo carácter: %w", err)
		}

		// Calcular delay entre caracteres
		charDelay := 60000.0 / baseSpeed // ms por carácter

		// Variaciones según tipo de carácter
		switch {
		case char >= '0' && char <= '9': // Números más rápidos
			charDelay *= 0.8
		case char >= 'A' && char <= 'Z': // Mayúsculas más lentas
			charDelay *= 1.2
		case char == ' ': // Espacios más rápidos
			charDelay *= 0.6
		}

		// Pausas ocasionales como si pensara
		if rand.Float64() < 0.15 { // 15% probabilidad de pausa
			thinkPause := s.humanSim.generateLogNormalDelay(800, 400)
			time.Sleep(thinkPause)
		}

		// Errores de escritura ocasionales (más frecuentes con fatiga)
		errorProb := 0.02 + s.humanSim.fatigueLevel*0.03
		if rand.Float64() < errorProb && i < len(text)-1 {
			// Escribir carácter incorrecto
			wrongChar := rune('a' + rand.Intn(26))
			element.Input(string(wrongChar))

			// Pausa de "darse cuenta del error"
			errorRealizationDelay := s.humanSim.generateLogNormalDelay(300, 100)
			time.Sleep(errorRealizationDelay)

			// Borrar carácter incorrecto
			page := element.Page()
			page.Keyboard.Press(input.Backspace)

			// Pausa antes de escribir el carácter correcto
			correctionDelay := s.humanSim.generateLogNormalDelay(200, 80)
			time.Sleep(correctionDelay)
		}

		// Aplicar delay principal entre caracteres
		finalDelay := s.humanSim.generateLogNormalDelay(charDelay, charDelay*0.3)
		time.Sleep(finalDelay)
	}

	// Pausa final después de escribir
	endDelay := s.humanSim.generateLogNormalDelay(400, 200)
	time.Sleep(endDelay)

	return nil
}

// Función modificada que guarda datos parciales antes de terminar
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

func (s *ScraperExtendido) retryScrape2(maxRetries int, name string, scrapeFunc func() error) {
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
}

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

	// 3. Buscar patrón específico de SUNAT: "1 a X de Y"
	patronResultados, err := page.Timeout(3 * time.Second).ElementX("//td[contains(text(), ' a ') and contains(text(), ' de ')]")
	if err == nil && patronResultados != nil {
		visible, err := patronResultados.Visible()
		if err == nil && visible {
			texto, _ := patronResultados.Text()
			// Verificar que sigue el patrón "X a Y de Z"
			if matched, _ := regexp.MatchString(`\d+\s+a\s+\d+\s+de\s+\d+`, texto); matched {
				log.Printf("✅ Detectada paginación SUNAT: patrón de resultados '%s'", texto)
				return true
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
		// PATRONES ESPECÍFICOS DE SUNAT
		if strings.Contains(pageText, "Páginas:") {
			log.Printf("✅ Detectada paginación SUNAT: texto 'Páginas:' encontrado en body")
			return true
		}

		// Patrones muy específicos que indican paginación real
		patronesPaginacionEspecificos := []string{
			`\d+\s+a\s+\d+\s+de\s+\d+`, // "1 a 30 de 225"
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
		// Buscar indicios adicionales de que es primera página
		pageText, err := page.MustElement("body").Text()
		if err == nil {
			if strings.Contains(pageText, "Páginas:") ||
				regexp.MustCompile(`\d+\s+a\s+\d+\s+de\s+\d+`).MatchString(pageText) {
				log.Printf("✅ Detectada paginación: botón Siguiente + indicadores de primera página")
				return true
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

// ScrapeRUCCompleto obtiene toda la información disponible de un RUC
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

	// ========================================
	// CONSULTAS PRINCIPALES (SIEMPRE DISPONIBLES)
	// ========================================
	fmt.Println(" 📋 Consultando información principal obligatoria...")

	// 1. Información Histórica (PRINCIPAL)
	fmt.Print(" - Información Histórica: ")
	s.retryScrapeWithPartialSave(3, "Información Histórica", func() error {
		infoHist, tienePaginacion, err := s.ScrapeInformacionHistorica(ruc, page)
		if err == nil {
			rucCompleto.InformacionHistorica = infoHist
			rucCompleto.DeteccionPaginacion["Información Histórica"] = tienePaginacion
		}
		return err
	}, rucCompleto, dbService, ruc, page)

	// 2. Deuda Coactiva (PRINCIPAL)
	fmt.Print(" - Deuda Coactiva: ")
	s.retryScrapeWithPartialSave(3, "Deuda Coactiva", func() error {
		deuda, tienePaginacion, err := s.ScrapeDeudaCoactiva(ruc, page)
		if err == nil {
			rucCompleto.DeudaCoactiva = deuda
			rucCompleto.DeteccionPaginacion["Deuda Coactiva"] = tienePaginacion
		}
		return err
	}, rucCompleto, dbService, ruc, page)

	// 3. Omisiones Tributarias (PRINCIPAL)
	fmt.Print(" - Omisiones Tributarias: ")
	s.retryScrapeWithPartialSave(3, "Omisiones Tributarias", func() error {
		omis, tienePaginacion, err := s.ScrapeOmisionesTributarias(ruc, page)
		if err == nil {
			rucCompleto.OmisionesTributarias = omis
			rucCompleto.DeteccionPaginacion["Omisiones Tributarias"] = tienePaginacion
		}
		return err
	}, rucCompleto, dbService, ruc, page)

	// 4. Cantidad de Trabajadores (PRINCIPAL)
	fmt.Print(" - Cantidad de Trabajadores: ")
	s.retryScrapeWithPartialSave(3, "Cantidad de Trabajadores", func() error {
		trab, tienePaginacion, err := s.ScrapeCantidadTrabajadores(ruc, page)
		if err == nil {
			rucCompleto.CantidadTrabajadores = trab
			rucCompleto.DeteccionPaginacion["Cantidad de Trabajadores"] = tienePaginacion
		}
		return err
	}, rucCompleto, dbService, ruc, page)

	// 5. Actas Probatorias (PRINCIPAL)
	fmt.Print(" - Actas Probatorias: ")
	s.retryScrapeWithPartialSave(3, "Actas Probatorias", func() error {
		actas, tienePaginacion, err := s.ScrapeActasProbatorias(ruc, page)
		if err == nil {
			rucCompleto.ActasProbatorias = actas
			rucCompleto.DeteccionPaginacion["Actas Probatorias"] = tienePaginacion
		}
		return err
	}, rucCompleto, dbService, ruc, page)

	// 6. Facturas Físicas (PRINCIPAL)
	fmt.Print(" - Facturas Físicas: ")
	s.retryScrapeWithPartialSave(3, "Facturas Físicas", func() error {
		fact, tienePaginacion, err := s.ScrapeFacturasFisicas(ruc, page)
		if err == nil {
			rucCompleto.FacturasFisicas = fact
			rucCompleto.DeteccionPaginacion["Facturas Físicas"] = tienePaginacion
		}
		return err
	}, rucCompleto, dbService, ruc, page)

	// Si hubo un error crítico en las primeras consultas, retornar los datos parciales
	// (Este bloque ya no es necesario porque retryScrapeWithPartialSave maneja todo)

	// ========================================
	// CONSULTAS ESPECÍFICAS POR TIPO DE RUC
	// ========================================
	if esPersonaJuridica {
		// PERSONAS JURÍDICAS (RUC 20): 3 consultas principales adicionales
		fmt.Println(" 📋 Consultando información principal de Personas Jurídicas...")

		// 7. Representantes Legales (PRINCIPAL para RUC 20)
		fmt.Print(" - Representantes Legales: ")
		s.retryScrapeWithPartialSave(3, "Representantes Legales", func() error {
			reps, tienePaginacion, err := s.ScrapeRepresentantesLegales(ruc, page)
			if err == nil {
				rucCompleto.RepresentantesLegales = reps
				rucCompleto.DeteccionPaginacion["Representantes Legales"] = tienePaginacion
			}
			return err
		}, rucCompleto, dbService, ruc, page)

		// ========================================
		// CONSULTAS OCASIONALES PARA RUC 20
		// ========================================
		fmt.Println(" 📋 Consultando información ocasional de Personas Jurídicas...")

		// 8. Reactiva Perú (PRINCIPAL para RUC 20)
		fmt.Print(" - Reactiva Perú: ")
		s.retryScrape2(3, "Reactiva Perú", func() error {
			react, err := s.ScrapeReactivaPeru(ruc, page)
			if err == nil {
				rucCompleto.ReactivaPeru = react
			}
			return err
		})

		// 9. Programa COVID-19 (PRINCIPAL para RUC 20)
		fmt.Print(" - Programa COVID-19: ")
		s.retryScrape2(3, "Programa COVID-19", func() error {
			covid, err := s.ScrapeProgramaCovid19(ruc, page)
			if err == nil {
				rucCompleto.ProgramaCovid19 = covid
			}
			return err
		})

		// Establecimientos Anexos (OCASIONAL para RUC 20)
		fmt.Print(" - Establecimientos Anexos: ")
		s.retryScrape2(2, "Establecimientos Anexos", func() error {
			estab, tienePaginacion, err := s.ScrapeEstablecimientosAnexos(ruc, page)
			if err == nil {
				rucCompleto.EstablecimientosAnexos = estab
				rucCompleto.DeteccionPaginacion["Establecimientos Anexos"] = tienePaginacion
			}
			return err
		})

	} else {
		// PERSONAS NATURALES (RUC 10): consultas ocasionales
		fmt.Println(" 📋 Consultando información ocasional de Personas Naturales...")

		// Reactiva Perú (OCASIONAL para RUC 10)
		fmt.Print(" - Reactiva Perú: ")
		s.retryScrape2(2, "Reactiva Perú", func() error {
			react, err := s.ScrapeReactivaPeru(ruc, page)
			if err == nil {
				rucCompleto.ReactivaPeru = react
			}
			return err
		})

		// Programa COVID-19 (OCASIONAL para RUC 10)
		fmt.Print(" - Programa COVID-19: ")
		s.retryScrape2(2, "Programa COVID-19", func() error {
			covid, err := s.ScrapeProgramaCovid19(ruc, page)
			if err == nil {
				rucCompleto.ProgramaCovid19 = covid
			}
			return err
		})
	}

	return rucCompleto, nil
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

	// Esperar que página original esté lista
	page.WaitLoad()
	page.WaitStable(2 * time.Second)

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
