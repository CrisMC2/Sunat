package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
)

// BiometricProfile simula un perfil biométrico único por sesión
type BiometricProfile struct {
	MouseMovementPattern string    // "smooth", "jittery", "precise"
	ClickPattern         string    // "fast", "deliberate", "double"
	ScrollBehavior       string    // "smooth", "chunky", "reader"
	TypingRhythm         []float64 // Patrón único de velocidad de tipeo
	ReactionTime         float64   // Tiempo base de reacción (ms)
	AttentionSpan        int       // Segundos antes de distracción
	Handedness           string    // "right", "left" - afecta movimientos del mouse
}

// AdvancedHumanBehaviorSimulator con características biométricas y detección de contexto
type AdvancedHumanBehaviorSimulator struct {
	SessionStart          time.Time
	ActionCount           int
	FatigureFactor        float64
	LastRestTime          time.Time
	CurrentUserAgentIndex int
	UserAgents            []string
	MaxActionsPerHour     int
	MaxSessionDuration    time.Duration
	ReadingSpeedWPM       float64
	TypingSpeedCPM        float64

	// Nuevas características
	BiometricProfile  BiometricProfile
	EmotionalState    string  // "focused", "tired", "distracted", "stressed"
	EnvironmentNoise  float64 // Factor de ruido ambiental (0-1)
	DeviceType        string  // "desktop", "mobile", "tablet"
	NetworkLatency    time.Duration
	LastMousePosition [2]int
	MouseTrajectory   [][2]int
	ActiveTabs        int
	DistractionEvents []time.Time

	mutex sync.RWMutex
}

// NewAdvancedHumanBehaviorSimulator crea un simulador con perfil biométrico único
func NewAdvancedHumanBehaviorSimulator() *AdvancedHumanBehaviorSimulator {
	profile := generateUniqueProfile()

	return &AdvancedHumanBehaviorSimulator{
		SessionStart:          time.Now(),
		ActionCount:           0,
		FatigureFactor:        1.0,
		LastRestTime:          time.Now(),
		CurrentUserAgentIndex: 0,
		MaxActionsPerHour:     45,
		MaxSessionDuration:    2 * time.Hour,
		ReadingSpeedWPM:       230.0,
		TypingSpeedCPM:        300.0,
		BiometricProfile:      profile,
		EmotionalState:        "focused",
		EnvironmentNoise:      0.1,
		DeviceType:            "desktop",
		NetworkLatency:        time.Duration(50+rand.Intn(100)) * time.Millisecond,
		ActiveTabs:            1,
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:119.0) Gecko/20100101 Firefox/119.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:119.0) Gecko/20100101 Firefox/119.0",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
		},
	}
}

// generateUniqueProfile crea un perfil biométrico único y consistente
func generateUniqueProfile() BiometricProfile {
	mousePatterns := []string{"smooth", "jittery", "precise"}
	clickPatterns := []string{"fast", "deliberate", "double"}
	scrollBehaviors := []string{"smooth", "chunky", "reader"}
	handedness := []string{"right", "left"}

	// Crear patrón de tipeo único (10 valores)
	typingRhythm := make([]float64, 10)
	for i := range typingRhythm {
		typingRhythm[i] = 0.8 + rand.Float64()*0.4 // 0.8-1.2 multiplicador
	}

	return BiometricProfile{
		MouseMovementPattern: mousePatterns[rand.Intn(len(mousePatterns))],
		ClickPattern:         clickPatterns[rand.Intn(len(clickPatterns))],
		ScrollBehavior:       scrollBehaviors[rand.Intn(len(scrollBehaviors))],
		TypingRhythm:         typingRhythm,
		ReactionTime:         200 + rand.Float64()*300, // 200-500ms
		AttentionSpan:        300 + rand.Intn(600),     // 5-15 minutos
		Handedness:           handedness[rand.Intn(len(handedness))],
	}
}

// AdvancedLogNormalDelay con contexto emocional y biométrico
func (h *AdvancedHumanBehaviorSimulator) AdvancedLogNormalDelay(mean, sigma float64, actionType string) time.Duration {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Ajuste base por estado emocional
	emotionalMultiplier := 1.0
	switch h.EmotionalState {
	case "tired":
		emotionalMultiplier = 1.4
	case "distracted":
		emotionalMultiplier = 1.6
	case "stressed":
		emotionalMultiplier = 1.2
	case "focused":
		emotionalMultiplier = 0.9
	}

	// Ajuste por tipo de acción
	actionMultiplier := 1.0
	switch actionType {
	case "click":
		actionMultiplier = h.BiometricProfile.ReactionTime / 300.0
	case "scroll":
		if h.BiometricProfile.ScrollBehavior == "reader" {
			actionMultiplier = 1.3
		}
	case "type":
		actionMultiplier = 1.0 // Se maneja en SimulateAdvancedTyping
	}

	// Ruido ambiental
	noiseMultiplier := 1.0 + (h.EnvironmentNoise * 0.3)

	// Generar delay con distribución log-normal
	normal := rand.NormFloat64()
	logNormal := math.Exp(mean + sigma*normal)

	// Aplicar todos los factores
	adjustedDelay := logNormal * h.FatigureFactor * emotionalMultiplier * actionMultiplier * noiseMultiplier

	// Limitar valores extremos
	if adjustedDelay < 0.1 {
		adjustedDelay = 0.1
	}
	if adjustedDelay > 15.0 {
		adjustedDelay = 15.0
	}

	return time.Duration(adjustedDelay * float64(time.Second))
}

// UpdateEmotionalState actualiza el estado emocional basado en la actividad
func (h *AdvancedHumanBehaviorSimulator) UpdateEmotionalState() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	sessionDuration := time.Since(h.SessionStart).Minutes()
	actionsPerMinute := float64(h.ActionCount) / sessionDuration

	// Determinar estado basado en patrones
	if sessionDuration > 60 {
		h.EmotionalState = "tired"
	} else if actionsPerMinute > 2.0 {
		h.EmotionalState = "stressed"
	} else if len(h.DistractionEvents) > 5 {
		h.EmotionalState = "distracted"
	} else {
		h.EmotionalState = "focused"
	}

	// Actualizar ruido ambiental
	h.EnvironmentNoise = 0.05 + rand.Float64()*0.2
}

// SimulateAdvancedTyping con patrón biométrico único
func (h *AdvancedHumanBehaviorSimulator) SimulateAdvancedTyping(ctx context.Context, selector, text string) error {
	// Limpiar campo
	err := chromedp.Run(ctx, chromedp.Clear(selector))
	if err != nil {
		return err
	}

	baseCPM := h.TypingSpeedCPM
	rhythmIndex := 0

	for i, char := range text {
		// Aplicar ritmo biométrico único
		rhythmMultiplier := h.BiometricProfile.TypingRhythm[rhythmIndex]
		rhythmIndex = (rhythmIndex + 1) % len(h.BiometricProfile.TypingRhythm)

		charDelay := (60.0 / baseCPM) * rhythmMultiplier

		// Ajustes por tipo de carácter (más sofisticados)
		switch {
		case char >= 'A' && char <= 'Z':
			charDelay *= 1.3 // Mayúsculas requieren Shift
		case strings.ContainsRune("!@#$%^&*()_+{}|:\"<>?", char):
			charDelay *= 1.6 // Símbolos complejos
		case strings.ContainsRune("1234567890", char):
			charDelay *= 1.1 // Números ligeramente más lentos
		case char == ' ':
			charDelay *= 0.7
		}

		// Pausas cognitivas más sofisticadas
		if strings.ContainsRune(" .,;:", char) && rand.Float64() < 0.15 {
			// Pausa para pensar en la siguiente palabra/frase
			cognitiveDelay := 0.3 + rand.Float64()*1.2
			time.Sleep(time.Duration(cognitiveDelay * float64(time.Second)))
		}

		// Errores más realistas con patrones de corrección
		if rand.Float64() < 0.025 && i > 0 {
			// Tipos de errores más variados
			errorType := rand.Intn(3)

			switch errorType {
			case 0: // Carácter adyacente en teclado QWERTY
				wrongChar := getAdjacentKey(char)
				chromedp.Run(ctx, chromedp.SendKeys(selector, wrongChar))
				time.Sleep(time.Duration(charDelay * float64(time.Second)))

				// Tiempo para notar el error (depende del perfil biométrico)
				noticeDelay := h.BiometricProfile.ReactionTime * 2
				time.Sleep(time.Duration(noticeDelay * float64(time.Millisecond)))

				chromedp.Run(ctx, chromedp.KeyEvent(kb.Backspace))
				time.Sleep(time.Duration(charDelay * float64(time.Second)))

			case 1: // Doble carácter (dedo pegajoso)
				chromedp.Run(ctx, chromedp.SendKeys(selector, string(char)+string(char)))
				time.Sleep(time.Duration(charDelay * float64(time.Second)))
				time.Sleep(time.Duration(h.BiometricProfile.ReactionTime * float64(time.Millisecond)))
				chromedp.Run(ctx, chromedp.KeyEvent(kb.Backspace))
				time.Sleep(time.Duration(charDelay * float64(time.Second)))

			case 2: // Transposición (escribir dos caracteres en orden incorrecto)
				if i < len(text)-1 {
					nextChar := rune(text[i+1])
					chromedp.Run(ctx, chromedp.SendKeys(selector, string(nextChar)+string(char)))
					time.Sleep(time.Duration(charDelay * 2 * float64(time.Second)))
					time.Sleep(time.Duration(h.BiometricProfile.ReactionTime * 1.5 * float64(time.Millisecond)))
					chromedp.Run(ctx, chromedp.KeyEvent(kb.Backspace))
					chromedp.Run(ctx, chromedp.KeyEvent(kb.Backspace))
					time.Sleep(time.Duration(charDelay * float64(time.Second)))
				}
			}
		}

		// Escribir carácter correcto
		err := chromedp.Run(ctx, chromedp.SendKeys(selector, string(char)))
		if err != nil {
			return err
		}

		// Variación humana en velocidad con factor biométrico
		variation := 0.7 + rand.Float64()*0.6 // 0.7-1.3
		actualDelay := charDelay * variation

		// Factor emocional
		switch h.EmotionalState {
		case "tired":
			actualDelay *= 1.2
		case "stressed":
			actualDelay *= 0.8 // Más rápido pero más errores
		case "distracted":
			// Pausas ocasionales más largas
			if rand.Float64() < 0.1 {
				actualDelay *= 3.0
			}
		}

		time.Sleep(time.Duration(actualDelay * float64(time.Second)))
	}

	return nil
}

// getAdjacentKey devuelve una tecla adyacente en el teclado QWERTY
func getAdjacentKey(char rune) string {
	adjacentMap := map[rune][]rune{
		'q': {'w', 'a'},
		'w': {'q', 'e', 's'},
		'e': {'w', 'r', 'd'},
		'r': {'e', 't', 'f'},
		't': {'r', 'y', 'g'},
		'y': {'t', 'u', 'h'},
		'u': {'y', 'i', 'j'},
		'i': {'u', 'o', 'k'},
		'o': {'i', 'p', 'l'},
		'p': {'o', 'l'},
		'a': {'q', 's', 'z'},
		's': {'w', 'a', 'd', 'x'},
		'd': {'e', 's', 'f', 'c'},
		'f': {'r', 'd', 'g', 'v'},
		'g': {'t', 'f', 'h', 'b'},
		'h': {'y', 'g', 'j', 'n'},
		'j': {'u', 'h', 'k', 'm'},
		'k': {'i', 'j', 'l'},
		'l': {'o', 'k', 'p'},
		'z': {'a', 's', 'x'},
		'x': {'z', 's', 'd', 'c'},
		'c': {'x', 'd', 'f', 'v'},
		'v': {'c', 'f', 'g', 'b'},
		'b': {'v', 'g', 'h', 'n'},
		'n': {'b', 'h', 'j', 'm'},
		'm': {'n', 'j', 'k'},
	}

	if adjacent, exists := adjacentMap[char]; exists && len(adjacent) > 0 {
		return string(adjacent[rand.Intn(len(adjacent))])
	}

	// Si no encuentra tecla adyacente, devolver una letra aleatoria
	return string(rune('a' + rand.Intn(26)))
}

// AdvancedMouseMovement simula movimiento natural del mouse
func (h *AdvancedHumanBehaviorSimulator) AdvancedMouseMovement(ctx context.Context, targetX, targetY int) error {
	currentX, currentY := h.LastMousePosition[0], h.LastMousePosition[1]

	// Calcular trayectoria basada en perfil biométrico
	steps := calculateMouseSteps(currentX, currentY, targetX, targetY, h.BiometricProfile.MouseMovementPattern)

	for _, point := range steps {
		// Simular movimiento del mouse con JavaScript
		script := fmt.Sprintf(`
			var event = new MouseEvent('mousemove', {
				clientX: %d,
				clientY: %d,
				bubbles: true
			});
			document.dispatchEvent(event);
		`, point[0], point[1])

		chromedp.Run(ctx, chromedp.Evaluate(script, nil))

		// Delay entre movimientos
		moveDelay := time.Duration(5+rand.Intn(10)) * time.Millisecond
		if h.BiometricProfile.MouseMovementPattern == "jittery" {
			moveDelay *= 2
		}
		time.Sleep(moveDelay)
	}

	// Actualizar posición
	h.LastMousePosition = [2]int{targetX, targetY}
	h.MouseTrajectory = append(h.MouseTrajectory, [2]int{targetX, targetY})

	return nil
}

// calculateMouseSteps genera una trayectoria de mouse natural
func calculateMouseSteps(fromX, fromY, toX, toY int, pattern string) [][2]int {
	steps := make([][2]int, 0)

	deltaX := toX - fromX
	deltaY := toY - fromY
	distance := math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY))

	numSteps := int(distance / 20) // Un paso cada 20 pixels aproximadamente
	if numSteps < 2 {
		numSteps = 2
	}

	for i := 0; i <= numSteps; i++ {
		progress := float64(i) / float64(numSteps)

		// Curva Bézier para movimiento natural
		x := fromX + int(float64(deltaX)*progress)
		y := fromY + int(float64(deltaY)*progress)

		// Añadir variación según el patrón
		switch pattern {
		case "jittery":
			x += rand.Intn(6) - 3 // ±3 pixels de jitter
			y += rand.Intn(6) - 3
		case "smooth":
			// Curva suave sin modificación
		case "precise":
			// Movimiento muy directo, poca variación
			if i > 0 && i < numSteps {
				x += rand.Intn(2) - 1
				y += rand.Intn(2) - 1
			}
		}

		steps = append(steps, [2]int{x, y})
	}

	return steps
}

// DetectAndSimulateDistraction simula distracciones humanas naturales
func (h *AdvancedHumanBehaviorSimulator) DetectAndSimulateDistraction(ctx context.Context) bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Probabilidad de distracción basada en tiempo de sesión y estado emocional
	distractionProb := 0.02 // Base 2%

	sessionMinutes := time.Since(h.SessionStart).Minutes()
	if sessionMinutes > 30 {
		distractionProb += 0.03 // +3% después de 30 min
	}
	if sessionMinutes > 60 {
		distractionProb += 0.05 // +5% después de 1 hora
	}

	if h.EmotionalState == "distracted" {
		distractionProb *= 2.0
	}

	if rand.Float64() < distractionProb {
		// Simular distracción
		distractionType := rand.Intn(4)

		switch distractionType {
		case 0: // Pausa para leer notificación
			fmt.Println("Simulando distracción: notificación")
			time.Sleep(time.Duration(3+rand.Intn(7)) * time.Second)

		case 1: // Cambiar a otra pestaña brevemente
			fmt.Println("Simulando distracción: cambio de pestaña")
			h.ActiveTabs++
			time.Sleep(time.Duration(5+rand.Intn(15)) * time.Second)
			h.ActiveTabs--

		case 2: // Scrollear hacia arriba (revisión)
			fmt.Println("Simulando distracción: scroll de revisión")
			script := "window.scrollBy(0, -200);"
			chromedp.Run(ctx, chromedp.Evaluate(script, nil))
			time.Sleep(time.Duration(2+rand.Intn(5)) * time.Second)
			chromedp.Run(ctx, chromedp.Evaluate("window.scrollBy(0, 200);", nil))

		case 3: // Pausa para reflexión
			fmt.Println("Simulando distracción: pausa reflexiva")
			time.Sleep(time.Duration(4+rand.Intn(8)) * time.Second)
		}

		h.DistractionEvents = append(h.DistractionEvents, time.Now())
		return true
	}

	return false
}

// Ejemplo de uso de la versión avanzada
func ejemploUsoAvanzado() {
	rand.Seed(time.Now().UnixNano())

	simulator := NewAdvancedHumanBehaviorSimulator()
	fmt.Printf("Perfil biométrico generado: %+v\n", simulator.BiometricProfile)

	// Simulación de actividad
	for i := 0; i < 10; i++ {
		simulator.ActionCount++
		simulator.UpdateEmotionalState()

		delay := simulator.AdvancedLogNormalDelay(1.0, 0.5, "click")
		fmt.Printf("Acción %d - Estado: %s - Delay: %.2fs\n",
			i+1, simulator.EmotionalState, delay.Seconds())

		time.Sleep(delay)
	}
}
