package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// ============================================================================
// 1. EJEMPLO: COMPARACIÓN DE DELAYS ENTRE VERSIONES
// ============================================================================

// Bot Básico - Siempre el mismo delay
func basicBotDelay() time.Duration {
	return 1 * time.Second // Siempre 1 segundo exacto
}

// Tu Bot Original - Log-normal con fatiga
func originalBotDelay(fatigureFactor float64) time.Duration {
	normal := rand.NormFloat64()
	logNormal := math.Exp(1.5 + 0.8*normal)
	adjustedDelay := logNormal * fatigureFactor

	if adjustedDelay < 0.2 {
		adjustedDelay = 0.2
	}
	if adjustedDelay > 10.0 {
		adjustedDelay = 10.0
	}
	return time.Duration(adjustedDelay * float64(time.Second))
}

// Bot Avanzado - Todo integrado
func advancedBotDelay(emotionalState string, actionType string, environmentNoise float64, fatigureFactor float64) time.Duration {
	// Base log-normal
	normal := rand.NormFloat64()
	logNormal := math.Exp(1.5 + 0.8*normal)

	// Factores emocionales
	emotionalMultiplier := 1.0
	switch emotionalState {
	case "tired":
		emotionalMultiplier = 1.4
	case "distracted":
		emotionalMultiplier = 1.6
	case "stressed":
		emotionalMultiplier = 1.2
	case "focused":
		emotionalMultiplier = 0.9
	}

	// Factor de acción
	actionMultiplier := 1.0
	switch actionType {
	case "click":
		actionMultiplier = 0.8
	case "scroll":
		actionMultiplier = 1.1
	case "type":
		actionMultiplier = 1.0
	}

	// Ruido ambiental
	noiseMultiplier := 1.0 + (environmentNoise * 0.3)

	// Calcular delay final
	adjustedDelay := logNormal * fatigureFactor * emotionalMultiplier * actionMultiplier * noiseMultiplier

	if adjustedDelay < 0.1 {
		adjustedDelay = 0.1
	}
	if adjustedDelay > 15.0 {
		adjustedDelay = 15.0
	}

	return time.Duration(adjustedDelay * float64(time.Second))
}

// ============================================================================
// 2. EJEMPLO: SIMULACIÓN DE TIPEO
// ============================================================================

// Bot Básico - Escribe todo instantáneo
func basicBotTyping(text string) {
	fmt.Printf("🤖 Bot Básico escribe: '%s' [INSTANTÁNEO]\n", text)
}

// Tu Bot Original - Carácter por carácter con errores básicos
func originalBotTyping(text string) {
	fmt.Printf("🔧 Bot Original escribe: '")

	for i, char := range text {
		// Error ocasional (3%)
		if rand.Float64() < 0.03 && i > 0 {
			wrongChar := string(rune('a' + rand.Intn(26)))
			fmt.Printf("%s", wrongChar)
			time.Sleep(50 * time.Millisecond) // Simular tiempo
			fmt.Printf("\b \b")               // Backspace visual
			time.Sleep(50 * time.Millisecond)
		}

		fmt.Printf("%s", string(char))

		// Delay variable
		charDelay := 60.0 / 300.0 // 300 CPM base
		if char >= 'A' && char <= 'Z' {
			charDelay *= 1.2
		}

		actualDelay := charDelay * (0.5 + rand.Float64())
		time.Sleep(time.Duration(actualDelay*100) * time.Millisecond) // Acelerado para demo
	}
	fmt.Printf("'\n")
}

// Bot Avanzado - Con perfil biométrico y errores realistas
func advancedBotTyping(text string, profile BiometricProfile, emotionalState string) {
	fmt.Printf("🧠 Bot Avanzado escribe: '")

	rhythmIndex := 0

	for i, char := range text {
		// Aplicar ritmo biométrico único
		rhythmMultiplier := profile.TypingRhythm[rhythmIndex]
		rhythmIndex = (rhythmIndex + 1) % len(profile.TypingRhythm)

		charDelay := (60.0 / 300.0) * rhythmMultiplier

		// Pausas cognitivas más sofisticadas
		if strings.ContainsRune(" .,;:", char) && rand.Float64() < 0.15 {
			fmt.Printf("... ") // Pausa visible
			time.Sleep(time.Duration(300+rand.Intn(600)) * time.Millisecond)
		}

		// Errores más realistas
		if rand.Float64() < 0.025 && i > 0 {
			errorType := rand.Intn(3)

			switch errorType {
			case 0: // Carácter adyacente
				wrongChar := getAdjacentKey(char)
				fmt.Printf("%s", wrongChar)
				time.Sleep(time.Duration(charDelay*100) * time.Millisecond)
				time.Sleep(time.Duration(200+rand.Intn(400)) * time.Millisecond) // Tiempo para notar
				fmt.Printf("\b \b")
				time.Sleep(time.Duration(charDelay*100) * time.Millisecond)

			case 1: // Doble carácter
				fmt.Printf("%s%s", string(char), string(char))
				time.Sleep(time.Duration(charDelay*100) * time.Millisecond)
				time.Sleep(time.Duration(150+rand.Intn(300)) * time.Millisecond)
				fmt.Printf("\b \b")
				time.Sleep(time.Duration(charDelay*100) * time.Millisecond)

			case 2: // Transposición
				if i < len(text)-1 {
					nextChar := rune(text[i+1])
					fmt.Printf("%s%s", string(nextChar), string(char))
					time.Sleep(time.Duration(charDelay*200) * time.Millisecond)
					time.Sleep(time.Duration(300+rand.Intn(400)) * time.Millisecond)
					fmt.Printf("\b \b\b \b")
					time.Sleep(time.Duration(charDelay*100) * time.Millisecond)
				}
			}
		}

		// Escribir carácter correcto
		fmt.Printf("%s", string(char))

		// Factor emocional
		actualDelay := charDelay
		switch emotionalState {
		case "tired":
			actualDelay *= 1.2
		case "stressed":
			actualDelay *= 0.8
		case "distracted":
			if rand.Float64() < 0.1 {
				actualDelay *= 3.0 // Pausa larga ocasional
			}
		}

		time.Sleep(time.Duration(actualDelay*100) * time.Millisecond)
	}
	fmt.Printf("'\n")
}

// ============================================================================
// 3. EJEMPLO: MOVIMIENTO DEL MOUSE
// ============================================================================

type Point struct {
	X, Y int
}

// Bot Básico - Teleportación instantánea
func basicMouseMovement(from, to Point) []Point {
	fmt.Printf("🤖 Bot Básico: Mouse salta de (%d,%d) a (%d,%d) INSTANTÁNEAMENTE\n",
		from.X, from.Y, to.X, to.Y)
	return []Point{from, to}
}

// Tu Bot Original - Movimiento directo con pequeñas pausas
func originalMouseMovement(from, to Point) []Point {
	fmt.Printf("🔧 Bot Original: Mouse se mueve de (%d,%d) a (%d,%d) en línea recta\n",
		from.X, from.Y, to.X, to.Y)

	path := make([]Point, 0)
	steps := 10

	for i := 0; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		x := from.X + int(float64(to.X-from.X)*progress)
		y := from.Y + int(float64(to.Y-from.Y)*progress)
		path = append(path, Point{x, y})
	}

	return path
}

// Bot Avanzado - Curva natural con personalidad
func advancedMouseMovement(from, to Point, mousePattern string) []Point {
	fmt.Printf("🧠 Bot Avanzado: Mouse (%s) se mueve de (%d,%d) a (%d,%d) con curva natural\n",
		mousePattern, from.X, from.Y, to.X, to.Y)

	path := make([]Point, 0)
	deltaX := to.X - from.X
	deltaY := to.Y - from.Y
	distance := math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY))

	steps := int(distance / 20)
	if steps < 5 {
		steps = 5
	}

	for i := 0; i <= steps; i++ {
		progress := float64(i) / float64(steps)

		// Curva base
		x := from.X + int(float64(deltaX)*progress)
		y := from.Y + int(float64(deltaY)*progress)

		// Añadir curva Bézier suave (simulada)
		if i > 0 && i < steps {
			curveFactor := math.Sin(progress*math.Pi) * 0.2
			x += int(float64(-deltaY) * curveFactor * 0.1)
			y += int(float64(deltaX) * curveFactor * 0.1)
		}

		// Personalidad del mouse
		switch mousePattern {
		case "jittery":
			x += rand.Intn(6) - 3
			y += rand.Intn(6) - 3
		case "smooth":
			// Ya suave por defecto
		case "precise":
			if i > 0 && i < steps {
				x += rand.Intn(2) - 1
				y += rand.Intn(2) - 1
			}
		}

		path = append(path, Point{x, y})
	}

	return path
}

// ============================================================================
// 4. EJEMPLO: SIMULACIÓN DE DISTRACCIONES
// ============================================================================

type DistractionEvent struct {
	Type        string
	Duration    time.Duration
	Description string
}

// Bot Básico - Sin distracciones
func basicBotDistraction() *DistractionEvent {
	return nil // Nunca se distrae
}

// Tu Bot Original - Solo descansos programados
func originalBotDistraction(actionCount int) *DistractionEvent {
	if actionCount%75 == 0 {
		return &DistractionEvent{
			Type:        "programmed_rest",
			Duration:    time.Duration(120+rand.Intn(180)) * time.Second,
			Description: "Descanso programado cada 75 acciones",
		}
	}
	return nil
}

// Bot Avanzado - Distracciones naturales y contextuales
func advancedBotDistraction(emotionalState string, sessionMinutes float64) *DistractionEvent {
	baseProb := 0.02 // 2% base

	if sessionMinutes > 30 {
		baseProb += 0.03
	}
	if sessionMinutes > 60 {
		baseProb += 0.05
	}
	if emotionalState == "distracted" {
		baseProb *= 2.0
	}

	if rand.Float64() < baseProb {
		distractionTypes := []DistractionEvent{
			{
				Type:        "notification",
				Duration:    time.Duration(3+rand.Intn(7)) * time.Second,
				Description: "📱 Leyendo notificación de WhatsApp",
			},
			{
				Type:        "tab_switch",
				Duration:    time.Duration(5+rand.Intn(15)) * time.Second,
				Description: "🔄 Revisando email en otra pestaña",
			},
			{
				Type:        "scroll_review",
				Duration:    time.Duration(2+rand.Intn(5)) * time.Second,
				Description: "↩️ Releyendo información anterior",
			},
			{
				Type:        "thinking_pause",
				Duration:    time.Duration(4+rand.Intn(8)) * time.Second,
				Description: "🤔 Pausa reflexiva sobre la información",
			},
		}

		selected := distractionTypes[rand.Intn(len(distractionTypes))]
		return &selected
	}

	return nil
}

// ============================================================================
// 5. ESTRUCTURAS Y FUNCIONES DE SOPORTE
// ============================================================================

type BiometricProfile struct {
	MouseMovementPattern string
	ClickPattern         string
	ScrollBehavior       string
	TypingRhythm         []float64
	ReactionTime         float64
	AttentionSpan        int
	Handedness           string
}

func generateExampleProfile() BiometricProfile {
	typingRhythm := make([]float64, 10)
	for i := range typingRhythm {
		typingRhythm[i] = 0.8 + rand.Float64()*0.4
	}

	mousePatterns := []string{"smooth", "jittery", "precise"}
	clickPatterns := []string{"fast", "deliberate", "double"}
	scrollBehaviors := []string{"smooth", "chunky", "reader"}
	handedness := []string{"right", "left"}

	return BiometricProfile{
		MouseMovementPattern: mousePatterns[rand.Intn(len(mousePatterns))],
		ClickPattern:         clickPatterns[rand.Intn(len(clickPatterns))],
		ScrollBehavior:       scrollBehaviors[rand.Intn(len(scrollBehaviors))],
		TypingRhythm:         typingRhythm,
		ReactionTime:         200 + rand.Float64()*300,
		AttentionSpan:        300 + rand.Intn(600),
		Handedness:           handedness[rand.Intn(len(handedness))],
	}
}

func getAdjacentKey(char rune) string {
	adjacentMap := map[rune][]rune{
		'h': {'g', 'j', 'y', 'n', 'u'},
		'e': {'w', 'r', 'd', 's'},
		'l': {'k', 'o', 'p'},
		'o': {'i', 'p', 'l', 'k'},
		// ... más mapeos
	}

	if adjacent, exists := adjacentMap[char]; exists && len(adjacent) > 0 {
		return string(adjacent[rand.Intn(len(adjacent))])
	}
	return string(rune('a' + rand.Intn(26)))
}

// ============================================================================
// 6. EJEMPLOS DE COMPARACIÓN EN ACCIÓN
// ============================================================================

func runDelayComparison() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMPARACIÓN DE DELAYS PARA 10 ACCIONES CONSECUTIVAS")
	fmt.Println(strings.Repeat("=", 80))

	fatigureFactor := 1.0
	emotionalState := "focused"
	environmentNoise := 0.15

	fmt.Printf("%-15s %-20s %-20s %-25s\n", "Acción", "Bot Básico", "Bot Original", "Bot Avanzado")
	fmt.Println(strings.Repeat("-", 80))

	for i := 1; i <= 10; i++ {
		// Simular fatiga creciente
		if i > 5 {
			fatigureFactor = 1.2
			emotionalState = "tired"
		}
		if i > 8 {
			emotionalState = "distracted"
		}

		basicDelay := basicBotDelay()
		originalDelay := originalBotDelay(fatigureFactor)
		advancedDelay := advancedBotDelay(emotionalState, "click", environmentNoise, fatigureFactor)

		fmt.Printf("%-15d %-20s %-20s %-25s\n",
			i,
			fmt.Sprintf("%.3fs (fijo)", basicDelay.Seconds()),
			fmt.Sprintf("%.3fs", originalDelay.Seconds()),
			fmt.Sprintf("%.3fs (%s)", advancedDelay.Seconds(), emotionalState),
		)
	}
}

func runTypingComparison() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMPARACIÓN DE SIMULACIÓN DE TIPEO")
	fmt.Println(strings.Repeat("=", 80))

	testText := "hello world"
	profile := generateExampleProfile()

	fmt.Println("\n1. BOT BÁSICO:")
	basicBotTyping(testText)

	fmt.Println("\n2. TU BOT ORIGINAL:")
	originalBotTyping(testText)

	fmt.Println("\n3. BOT AVANZADO (Estado: focused):")
	advancedBotTyping(testText, profile, "focused")

	fmt.Println("\n4. BOT AVANZADO (Estado: stressed):")
	advancedBotTyping(testText, profile, "stressed")
}

func runMouseMovementComparison() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMPARACIÓN DE MOVIMIENTO DEL MOUSE")
	fmt.Println(strings.Repeat("=", 80))

	from := Point{100, 150}
	to := Point{500, 350}

	fmt.Println("\n1. BOT BÁSICO:")
	basicPath := basicMouseMovement(from, to)
	fmt.Printf("   Puntos: %d\n", len(basicPath))

	fmt.Println("\n2. TU BOT ORIGINAL:")
	originalPath := originalMouseMovement(from, to)
	fmt.Printf("   Puntos: %d\n", len(originalPath))
	fmt.Printf("   Primeros 5: %v\n", originalPath[:5])

	fmt.Println("\n3. BOT AVANZADO (Personalidad: smooth):")
	smoothPath := advancedMouseMovement(from, to, "smooth")
	fmt.Printf("   Puntos: %d\n", len(smoothPath))
	fmt.Printf("   Primeros 5: %v\n", smoothPath[:5])

	fmt.Println("\n4. BOT AVANZADO (Personalidad: jittery):")
	jitteryPath := advancedMouseMovement(from, to, "jittery")
	fmt.Printf("   Puntos: %d\n", len(jitteryPath))
	fmt.Printf("   Primeros 5: %v\n", jitteryPath[:5])
}

func runDistractionComparison() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMPARACIÓN DE SIMULACIÓN DE DISTRACCIONES")
	fmt.Println(strings.Repeat("=", 80))

	// Simular 50 acciones en diferentes estados
	actionCount := 0
	sessionMinutes := 0.0

	fmt.Printf("%-10s %-15s %-15s %-40s\n", "Minuto", "Bot Básico", "Bot Original", "Bot Avanzado")
	fmt.Println(strings.Repeat("-", 80))

	for minute := 0; minute < 90; minute += 15 {
		sessionMinutes = float64(minute)
		actionCount += 15

		emotionalState := "focused"
		if minute > 30 {
			emotionalState = "tired"
		}
		if minute > 60 {
			emotionalState = "distracted"
		}

		basic := basicBotDistraction()
		original := originalBotDistraction(actionCount)
		advanced := advancedBotDistraction(emotionalState, sessionMinutes)

		basicStr := "Sin distracción"
		if basic != nil {
			basicStr = basic.Description
		}

		originalStr := "Sin distracción"
		if original != nil {
			originalStr = "Descanso programado"
		}

		advancedStr := "Sin distracción"
		if advanced != nil {
			advancedStr = fmt.Sprintf("%s (%.1fs)", advanced.Description, advanced.Duration.Seconds())
		}

		fmt.Printf("%-10d %-15s %-15s %-40s\n", minute, basicStr, originalStr, advancedStr)
	}
}

func runBiometricProfileExample() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("EJEMPLO DE PERFILES BIOMÉTRICOS ÚNICOS")
	fmt.Println(strings.Repeat("=", 80))

	// Generar 3 perfiles diferentes
	profiles := []BiometricProfile{
		generateExampleProfile(),
		generateExampleProfile(),
		generateExampleProfile(),
	}

	profileNames := []string{"Usuario A (Programador)", "Usuario B (Diseñador)", "Usuario C (Manager)"}

	for i, profile := range profiles {
		fmt.Printf("\n%s:\n", profileNames[i])
		fmt.Printf("  Mouse: %s | Clicks: %s | Scroll: %s | Mano dominante: %s\n",
			profile.MouseMovementPattern, profile.ClickPattern,
			profile.ScrollBehavior, profile.Handedness)
		fmt.Printf("  Tiempo reacción: %.0fms | Atención: %d seg\n",
			profile.ReactionTime, profile.AttentionSpan)
		fmt.Printf("  Ritmo tipeo: [%.2f, %.2f, %.2f, %.2f, %.2f, ...]\n",
			profile.TypingRhythm[0], profile.TypingRhythm[1],
			profile.TypingRhythm[2], profile.TypingRhythm[3], profile.TypingRhythm[4])
	}
}

// ============================================================================
// FUNCIÓN PRINCIPAL CON TODOS LOS EJEMPLOS
// ============================================================================

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("DEMOSTRACIÓN COMPLETA: EVOLUCIÓN DE BOTS DE WEB SCRAPING")
	fmt.Println("Bot Básico → Tu Bot Original → Bot Avanzado con IA")

	// Ejecutar todas las comparaciones
	runDelayComparison()
	runTypingComparison()
	runMouseMovementComparison()
	runDistractionComparison()
	runBiometricProfileExample()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("RESUMEN DE DIFERENCIAS CLAVE:")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Println(`
🤖 BOT BÁSICO:
   - Delays fijos (1s siempre)
   - Tipeo instantáneo
   - Mouse teleporting
   - Sin distracciones
   - Fácilmente detectable

🔧 TU BOT ORIGINAL:
   - Delays log-normales con fatiga ✅
   - Tipeo carácter por carácter con errores ✅
   - Descansos programados ✅
   - Configuración stealth ✅
   - Mucho mejor, nivel avanzado

🧠 BOT AVANZADO:
   - Todo lo anterior PLUS:
   - Perfiles biométricos únicos ✅
   - Estados emocionales dinámicos ✅
   - Movimientos de mouse naturales ✅
   - Errores de tipeo realistas ✅
   - Distracciones contextuales ✅
   - Factores ambientales ✅
   - Prácticamente indetectable

CONCLUSIÓN: Tu implementación ya es excelente (8.5/10).
Las mejoras propuestas la llevarían a nivel experto (9.5/10).
	`)
}
