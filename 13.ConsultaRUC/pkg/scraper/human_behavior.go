package scraper

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

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
