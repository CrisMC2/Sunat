package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	fmt.Println("🤖 Sistema de Simulación Humana Avanzado v2.0")
	fmt.Println("===============================================")
	fmt.Println()

	// Ejecutar todas las demos
	demoPerfilesBiometricos()
	fmt.Println()

	demoPatronesTipeo()
	fmt.Println()

	demoComportamientoEmocional()
	fmt.Println()

	demoMovimientoMouse()
	fmt.Println()

	demoDistracciones()
	fmt.Println()

	demoNavegacionRealista()
	fmt.Println()

	demoAnalisisPatrones()
}

// Demo 1: Generación y comparación de diferentes perfiles biométricos
func demoPerfilesBiometricos() {
	fmt.Println("📊 DEMO 1: Perfiles Biométricos Únicos")
	fmt.Println("=====================================")

	// Generar varios perfiles únicos
	for i := 0; i < 5; i++ {
		simulator := NewAdvancedHumanBehaviorSimulator()
		profile := simulator.BiometricProfile

		fmt.Printf("Usuario %d:\n", i+1)
		fmt.Printf("  🖱️  Mouse: %s | Click: %s | Scroll: %s\n",
			profile.MouseMovementPattern, profile.ClickPattern, profile.ScrollBehavior)
		fmt.Printf("  ⌨️  Reacción: %.0fms | Atención: %ds | Mano: %s\n",
			profile.ReactionTime, profile.AttentionSpan, profile.Handedness)
		fmt.Printf("  🎯 Ritmo tipeo: [%.2f, %.2f, %.2f...]\n",
			profile.TypingRhythm[0], profile.TypingRhythm[1], profile.TypingRhythm[2])
		fmt.Println()
	}
}

// Demo 2: Simulación de diferentes patrones de tipeo
func demoPatronesTipeo() {
	fmt.Println("⌨️ DEMO 2: Patrones de Tipeo Biométricos")
	fmt.Println("=======================================")

	textosPrueba := []string{
		"Hola mundo",
		"Este es un texto más largo para probar",
		"¡Símbolos! @#$%^&*() y números: 1234567890",
		"MAYÚSCULAS y minúsculas mezcladas",
	}

	for i, texto := range textosPrueba {
		fmt.Printf("\n🔤 Prueba %d: \"%s\"\n", i+1, texto)

		simulator := NewAdvancedHumanBehaviorSimulator()

		// Simular diferentes estados emocionales
		estados := []string{"focused", "tired", "stressed", "distracted"}
		simulator.EmotionalState = estados[i%len(estados)]

		fmt.Printf("Estado emocional: %s\n", simulator.EmotionalState)

		start := time.Now()

		// Simular el tipeo (sin navegador real)
		baseCPM := simulator.TypingSpeedCPM
		rhythmIndex := 0

		fmt.Print("Simulando tipeo: ")
		for _, char := range texto {
			rhythmMultiplier := simulator.BiometricProfile.TypingRhythm[rhythmIndex]
			rhythmIndex = (rhythmIndex + 1) % len(simulator.BiometricProfile.TypingRhythm)

			charDelay := (60.0 / baseCPM) * rhythmMultiplier

			// Aplicar factores emocionales
			switch simulator.EmotionalState {
			case "tired":
				charDelay *= 1.2
			case "stressed":
				charDelay *= 0.8
			case "distracted":
				if rand.Float64() < 0.1 {
					charDelay *= 3.0
					fmt.Print("...")
				}
			}

			time.Sleep(time.Duration(charDelay * 100 * float64(time.Millisecond))) // Acelerado para demo
			fmt.Print(string(char))
		}

		duration := time.Since(start)
		wpm := float64(len(texto)) / 5.0 / duration.Minutes() // Aproximación de WPM

		fmt.Printf("\n⏱️  Tiempo: %.2fs | WPM: %.1f\n", duration.Seconds(), wpm)
	}
}

// Demo 3: Evolución del comportamiento emocional durante una sesión
func demoComportamientoEmocional() {
	fmt.Println("😊 DEMO 3: Evolución del Estado Emocional")
	fmt.Println("=======================================")

	simulator := NewAdvancedHumanBehaviorSimulator()

	// Simular una sesión de trabajo de varias horas (acelerada)
	fmt.Println("Simulando evolución emocional durante sesión de trabajo:")
	fmt.Println()

	for minutos := 0; minutos <= 180; minutos += 15 { // Cada 15 minutos hasta 3 horas
		// Simular paso del tiempo
		simulator.SessionStart = time.Now().Add(-time.Duration(minutos) * time.Minute)

		// Simular actividad
		actionsSimulated := minutos/2 + rand.Intn(10)
		simulator.ActionCount = actionsSimulated

		// Agregar algunas distracciones aleatorias
		if rand.Float64() < 0.3 {
			simulator.DistractionEvents = append(simulator.DistractionEvents, time.Now())
		}

		prevState := simulator.EmotionalState
		simulator.UpdateEmotionalState()

		if prevState != simulator.EmotionalState {
			fmt.Printf("⏰ %d min: %s → %s (acciones: %d, distracciones: %d)\n",
				minutos, prevState, simulator.EmotionalState,
				simulator.ActionCount, len(simulator.DistractionEvents))
		}

		// Mostrar métricas cada hora
		if minutos%60 == 0 && minutos > 0 {
			fatiga := 1.0 + (float64(minutos)/60.0)*0.2
			simulator.FatigureFactor = fatiga

			delay := simulator.AdvancedLogNormalDelay(1.0, 0.5, "click")
			fmt.Printf("📊 Hora %d - Fatiga: %.2f | Delay promedio: %.2fs | Ruido: %.2f\n",
				minutos/60, simulator.FatigureFactor, delay.Seconds(), simulator.EnvironmentNoise)
		}
	}
}

// Demo 4: Patrones de movimiento del mouse
func demoMovimientoMouse() {
	fmt.Println("🖱️ DEMO 4: Patrones de Movimiento del Mouse")
	fmt.Println("==========================================")

	patrones := []string{"smooth", "jittery", "precise"}

	for _, patron := range patrones {
		fmt.Printf("\n🎯 Patrón: %s\n", patron)

		simulator := NewAdvancedHumanBehaviorSimulator()
		simulator.BiometricProfile.MouseMovementPattern = patron
		simulator.LastMousePosition = [2]int{100, 100}

		// Simular movimientos a diferentes destinos
		destinos := [][2]int{{300, 200}, {500, 400}, {200, 300}, {400, 150}}

		for i, destino := range destinos {
			steps := calculateMouseSteps(
				simulator.LastMousePosition[0],
				simulator.LastMousePosition[1],
				destino[0],
				destino[1],
				patron,
			)

			fmt.Printf("  Movimiento %d: (%d,%d) → (%d,%d) en %d pasos\n",
				i+1,
				simulator.LastMousePosition[0], simulator.LastMousePosition[1],
				destino[0], destino[1],
				len(steps))

			// Mostrar algunos puntos de la trayectoria
			if len(steps) > 4 {
				fmt.Printf("    Trayectoria: (%d,%d) → (%d,%d) → ... → (%d,%d)\n",
					steps[0][0], steps[0][1],
					steps[len(steps)/2][0], steps[len(steps)/2][1],
					steps[len(steps)-1][0], steps[len(steps)-1][1])
			}

			simulator.LastMousePosition = destino
		}
	}
}

// Demo 5: Simulación de distracciones durante trabajo
func demoDistracciones() {
	fmt.Println("😵‍💫 DEMO 5: Simulación de Distracciones")
	fmt.Println("======================================")

	simulator := NewAdvancedHumanBehaviorSimulator()

	// Simular diferentes niveles de distracción
	escenarios := []struct {
		nombre   string
		estado   string
		duracion int
	}{
		{"Trabajo concentrado", "focused", 30},
		{"Después del almuerzo", "tired", 45},
		{"Deadline urgente", "stressed", 20},
		{"Tarde del viernes", "distracted", 60},
	}

	for _, escenario := range escenarios {
		fmt.Printf("\n📝 Escenario: %s (%s)\n", escenario.nombre, escenario.estado)

		simulator.EmotionalState = escenario.estado
		simulator.DistractionEvents = nil // Reset

		distracciones := 0

		// Simular acciones durante el período
		for minuto := 0; minuto < escenario.duracion; minuto++ {
			// Simular 2-4 acciones por minuto
			for accion := 0; accion < 2+rand.Intn(3); accion++ {
				simulator.ActionCount++

				// Crear contexto mock (sin navegador real)
				ctx := context.Background()

				// Verificar si hay distracción
				if simulator.DetectAndSimulateDistraction(ctx) {
					distracciones++
				}

				// Pequeña pausa entre acciones
				time.Sleep(10 * time.Millisecond) // Acelerado para demo
			}
		}

		fmt.Printf("  Total distracciones: %d/%d acciones (%.1f%%)\n",
			distracciones, simulator.ActionCount,
			float64(distracciones)/float64(simulator.ActionCount)*100)

		fmt.Printf("  Eventos de distracción registrados: %d\n", len(simulator.DistractionEvents))
	}
}

// Demo 6: Navegación web realista (simulada)
func demoNavegacionRealista() {
	fmt.Println("🌐 DEMO 6: Navegación Web Realista")
	fmt.Println("=================================")

	simulator := NewAdvancedHumanBehaviorSimulator()

	// Simular secuencia de navegación típica
	acciones := []struct {
		tipo        string
		descripcion string
		elemento    string
		texto       string
	}{
		{"navegar", "Abrir página de búsqueda", "", ""},
		{"esperar", "Cargar página", "", ""},
		{"click", "Hacer click en campo de búsqueda", "#search-input", ""},
		{"tipear", "Escribir término de búsqueda", "#search-input", "simulación comportamiento humano"},
		{"pausa", "Revisar sugerencias", "", ""},
		{"click", "Hacer click en buscar", "#search-button", ""},
		{"esperar", "Cargar resultados", "", ""},
		{"scroll", "Revisar resultados", "", ""},
		{"click", "Click en primer resultado", ".result-1", ""},
		{"esperar", "Cargar artículo", "", ""},
		{"scroll", "Leer contenido", "", ""},
		{"tipear", "Escribir comentario", "#comment-box", "Muy interesante este artículo sobre IA"},
	}

	fmt.Println("Simulando secuencia de navegación:")
	fmt.Println()

	for i, accion := range acciones {
		fmt.Printf("🔄 Paso %d: %s\n", i+1, accion.descripcion)

		switch accion.tipo {
		case "navegar":
			delay := simulator.AdvancedLogNormalDelay(2.0, 0.3, "navigate")
			fmt.Printf("   ⏱️  Tiempo de navegación: %.2fs\n", delay.Seconds())

		case "esperar":
			// Simular tiempo de carga + tiempo de lectura
			loadTime := time.Duration(1+rand.Intn(3)) * time.Second
			readTime := simulator.AdvancedLogNormalDelay(1.5, 0.4, "read")
			totalTime := loadTime + readTime

			fmt.Printf("   ⏱️  Carga: %.2fs + Lectura: %.2fs = %.2fs\n",
				loadTime.Seconds(), readTime.Seconds(), totalTime.Seconds())

		case "click":
			// Simular movimiento del mouse + click
			mouseTime := time.Duration(200+rand.Intn(800)) * time.Millisecond
			clickDelay := simulator.AdvancedLogNormalDelay(0.5, 0.2, "click")

			fmt.Printf("   🖱️  Movimiento: %.2fs + Click: %.2fs\n",
				mouseTime.Seconds(), clickDelay.Seconds())

		case "tipear":
			// Calcular tiempo de tipeo basado en texto
			caracteres := len(accion.texto)
			tiempoEstimado := float64(caracteres) * (60.0 / simulator.TypingSpeedCPM)

			// Aplicar variaciones biométricas
			variacion := 0.8 + rand.Float64()*0.4
			tiempoReal := tiempoEstimado * variacion

			fmt.Printf("   ⌨️  Tipeo \"%s\" (%d chars): %.2fs\n",
				accion.texto, caracteres, tiempoReal)

		case "scroll":
			scrollDelay := simulator.AdvancedLogNormalDelay(0.8, 0.3, "scroll")
			fmt.Printf("   📜 Tiempo de scroll: %.2fs\n", scrollDelay.Seconds())

		case "pausa":
			pauseDelay := simulator.AdvancedLogNormalDelay(1.2, 0.5, "think")
			fmt.Printf("   🤔 Pausa cognitiva: %.2fs\n", pauseDelay.Seconds())
		}

		// Actualizar métricas del simulador
		simulator.ActionCount++
		if i%3 == 0 {
			simulator.UpdateEmotionalState()
		}

		// Verificar distracciones ocasionales
		if rand.Float64() < 0.15 {
			fmt.Println("   😵‍💫 Distracción detectada")
		}

		// Pequeña pausa para visualización
		time.Sleep(100 * time.Millisecond)

		fmt.Println()
	}

	fmt.Printf("📊 Resumen de sesión:\n")
	fmt.Printf("   Acciones totales: %d\n", simulator.ActionCount)
	fmt.Printf("   Estado final: %s\n", simulator.EmotionalState)
	fmt.Printf("   Factor de fatiga: %.2f\n", simulator.FatigureFactor)
}

// Demo 7: Análisis de patrones y métricas
func demoAnalisisPatrones() {
	fmt.Println("📈 DEMO 7: Análisis de Patrones Biométricos")
	fmt.Println("==========================================")

	// Crear múltiples simuladores y analizar diferencias
	simuladores := make([]*AdvancedHumanBehaviorSimulator, 10)
	for i := range simuladores {
		simuladores[i] = NewAdvancedHumanBehaviorSimulator()
	}

	// Análisis de velocidades de tipeo
	fmt.Println("\n⌨️ Análisis de Velocidades de Tipeo:")
	var velocidades []float64
	for i, sim := range simuladores {
		velocidades = append(velocidades, sim.TypingSpeedCPM)
		fmt.Printf("Usuario %d: %.0f CPM (%.0f WPM)\n",
			i+1, sim.TypingSpeedCPM, sim.TypingSpeedCPM/5)
	}

	// Calcular estadísticas
	promedio, minimo, maximo := calcularEstadisticas(velocidades)
	fmt.Printf("Estadísticas: Promedio=%.0f, Mín=%.0f, Máx=%.0f CPM\n",
		promedio, minimo, maximo)

	// Análisis de tiempos de reacción
	fmt.Println("\n⚡ Análisis de Tiempos de Reacción:")
	var reacciones []float64
	for i, sim := range simuladores {
		reacciones = append(reacciones, sim.BiometricProfile.ReactionTime)
		fmt.Printf("Usuario %d: %.0fms\n", i+1, sim.BiometricProfile.ReactionTime)
	}

	promedio, minimo, maximo = calcularEstadisticas(reacciones)
	fmt.Printf("Estadísticas: Promedio=%.0f, Mín=%.0f, Máx=%.0fms\n",
		promedio, minimo, maximo)

	// Análisis de patrones de comportamiento
	fmt.Println("\n🎯 Distribución de Patrones de Comportamiento:")

	mousePatterns := make(map[string]int)
	clickPatterns := make(map[string]int)
	scrollPatterns := make(map[string]int)

	for _, sim := range simuladores {
		mousePatterns[sim.BiometricProfile.MouseMovementPattern]++
		clickPatterns[sim.BiometricProfile.ClickPattern]++
		scrollPatterns[sim.BiometricProfile.ScrollBehavior]++
	}

	fmt.Println("Patrones de Mouse:")
	for pattern, count := range mousePatterns {
		fmt.Printf("  %s: %d usuarios (%.0f%%)\n",
			pattern, count, float64(count)/float64(len(simuladores))*100)
	}

	fmt.Println("Patrones de Click:")
	for pattern, count := range clickPatterns {
		fmt.Printf("  %s: %d usuarios (%.0f%%)\n",
			pattern, count, float64(count)/float64(len(simuladores))*100)
	}

	fmt.Println("Patrones de Scroll:")
	for pattern, count := range scrollPatterns {
		fmt.Printf("  %s: %d usuarios (%.0f%%)\n",
			pattern, count, float64(count)/float64(len(simuladores))*100)
	}

	// Simulación de evolución temporal
	fmt.Println("\n⏰ Simulación de Evolución Temporal:")
	sim := NewAdvancedHumanBehaviorSimulator()

	for hora := 1; hora <= 8; hora++ { // 8 horas de trabajo
		// Simular paso del tiempo
		sim.SessionStart = time.Now().Add(-time.Duration(hora) * time.Hour)
		sim.ActionCount = hora * 30 // 30 acciones por hora

		// Agregar fatiga
		sim.FatigureFactor = 1.0 + float64(hora)*0.1

		// Actualizar estado
		sim.UpdateEmotionalState()

		// Calcular delay promedio
		delay := sim.AdvancedLogNormalDelay(1.0, 0.5, "click")

		fmt.Printf("Hora %d: Estado=%s, Fatiga=%.2f, Delay=%.2fs\n",
			hora, sim.EmotionalState, sim.FatigureFactor, delay.Seconds())
	}

	fmt.Println("\n✅ Análisis completado. El simulador demuestra variabilidad")
	fmt.Println("   biométrica realista y evolución temporal consistente.")
}

// Función auxiliar para calcular estadísticas básicas
func calcularEstadisticas(valores []float64) (promedio, minimo, maximo float64) {
	if len(valores) == 0 {
		return 0, 0, 0
	}

	suma := 0.0
	minimo = valores[0]
	maximo = valores[0]

	for _, valor := range valores {
		suma += valor
		if valor < minimo {
			minimo = valor
		}
		if valor > maximo {
			maximo = valor
		}
	}

	promedio = suma / float64(len(valores))
	return
}

// Función para demo con navegador real (opcional)
func demoNavegadorReal() {
	fmt.Println("🌐 DEMO EXTRA: Integración con Navegador Real")
	fmt.Println("============================================")

	// Crear contexto de Chrome
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	simulator := NewAdvancedHumanBehaviorSimulator()

	// Navegar y realizar acciones reales
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://example.com"),
		chromedp.WaitVisible("body"),
	)

	if err != nil {
		log.Printf("Error en navegación: %v", err)
		return
	}

	fmt.Println("✅ Navegación exitosa con simulador biométrico")

	// Ejemplo de tipeo realista
	if err := simulator.SimulateAdvancedTyping(ctx, "input[type='text']", "Texto de prueba con simulación biométrica"); err == nil {
		fmt.Println("✅ Tipeo biométrico completado")
	}
}
