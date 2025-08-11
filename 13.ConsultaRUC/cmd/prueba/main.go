package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	"github.com/consulta-ruc-scraper/pkg/scraper"
)

func main() {
	rucs := []string{"20606316977"}

	if len(os.Args) > 1 {
		rucs = os.Args[1:]
	}

	// Crear scraper extendido optimizado
	scraperExt := scraper.NewScraperExtendido("https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp")
	defer scraperExt.CloseBrowser() // Solo cerrar al final de todo

	for _, ruc := range rucs {
		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		fmt.Printf("🔍 CONSULTANDO RUC: %s\n", ruc)
		fmt.Printf("%s\n", strings.Repeat("=", 60))

		// =================================================================
		// 1. INFORMACIÓN HISTÓRICA
		// =================================================================
		fmt.Println("\n📊 1. INFORMACIÓN HISTÓRICA")
		fmt.Println(strings.Repeat("-", 40))

		infoHistorica, err := scraperExt.ScrapeInformacionHistorica(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información histórica obtenida exitosamente\n")
			fmt.Printf("   📅 Fecha consulta: %s\n", infoHistorica.FechaConsulta.Format("02/01/2006 15:04"))
			fmt.Printf("   🏢 Cambios razón social: %d\n", len(infoHistorica.CambiosRazonSocial))
			fmt.Printf("   🏠 Cambios domicilio: %d\n", len(infoHistorica.CambiosDomicilio))
			fmt.Printf("   📋 Cambios estado: %d\n", len(infoHistorica.CambiosEstado))
			fmt.Printf("   📊 Cambios condición: %d\n", len(infoHistorica.CambiosCondicion))
			fmt.Printf("   💼 Actividades históricas: %d\n", len(infoHistorica.ActividadesHistoricas))
		}

		// =================================================================
		// 2. DEUDA COACTIVA
		// =================================================================
		fmt.Println("\n💰 2. DEUDA COACTIVA")
		fmt.Println(strings.Repeat("-", 40))

		deudaCoactiva, err := scraperExt.ScrapeDeudaCoactiva(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información de deuda coactiva obtenida\n")
			if deudaCoactiva.CantidadDocumentos == 0 {
				fmt.Printf("   🟢 No registra deuda coactiva\n")
			} else {
				fmt.Printf("   🔴 Total deuda: S/ %.2f\n", deudaCoactiva.TotalDeuda)
				fmt.Printf("   📄 Documentos: %d\n", deudaCoactiva.CantidadDocumentos)
			}
		}

		// =================================================================
		// 3. REPRESENTANTES LEGALES
		// =================================================================
		fmt.Println("\n👥 3. REPRESENTANTES LEGALES")
		fmt.Println(strings.Repeat("-", 40))

		representantes, err := scraperExt.ScrapeRepresentantesLegales(ruc)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "no se encontró el botón"):
				fmt.Println("   ⚠️  No tiene representantes legales registrados")
			case strings.Contains(err.Error(), "deshabilitado"):
				fmt.Println("   🚫 Consulta no disponible para este RUC")
			default:
				fmt.Printf("   ❌ Error: %v\n", err)
			}
		} else {
			fmt.Printf("   ✅ Representantes legales obtenidos\n")
			if len(representantes.Representantes) == 0 {
				fmt.Printf("   📝 No se encontraron representantes registrados\n")
			} else {
				vigentes := 0
				for _, rep := range representantes.Representantes {
					if rep.Vigente {
						vigentes++
					}
				}
				fmt.Printf("   👤 Total representantes: %d\n", len(representantes.Representantes))
				fmt.Printf("   ✅ Vigentes: %d\n", vigentes)
				fmt.Printf("   ❌ No vigentes: %d\n", len(representantes.Representantes)-vigentes)
			}
		}

		// =================================================================
		// 4. CANTIDAD DE TRABAJADORES
		// =================================================================
		fmt.Println("\n👷 4. CANTIDAD DE TRABAJADORES")
		fmt.Println(strings.Repeat("-", 40))

		trabajadores, err := scraperExt.ScrapeCantidadTrabajadores(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información de trabajadores obtenida\n")
			fmt.Printf("   📅 Períodos disponibles: %d\n", len(trabajadores.PeriodosDisponibles))
			fmt.Printf("   📊 Detalles por período: %d\n", len(trabajadores.DetallePorPeriodo))

			if len(trabajadores.DetallePorPeriodo) > 0 {
				ultimo := trabajadores.DetallePorPeriodo[0] // Asumiendo que el primero es el más reciente
				fmt.Printf("   🕐 Último período: %s\n", ultimo.Periodo)
				fmt.Printf("   👥 Trabajadores: %d\n", ultimo.CantidadTrabajadores)
				fmt.Printf("   🤝 Prestadores: %d\n", ultimo.CantidadPrestadoresServicio)
				fmt.Printf("   👴 Pensionistas: %d\n", ultimo.CantidadPensionistas)
				fmt.Printf("   📈 Total: %d\n", ultimo.Total)
			}
		}

		// =================================================================
		// 5. ESTABLECIMIENTOS ANEXOS
		// =================================================================
		fmt.Println("\n🏢 5. ESTABLECIMIENTOS ANEXOS")
		fmt.Println(strings.Repeat("-", 40))

		establecimientos, err := scraperExt.ScrapeEstablecimientosAnexos(ruc)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "no se encontró el botón"):
				fmt.Println("   ℹ️  No tiene establecimientos anexos disponibles")
			case strings.Contains(err.Error(), "deshabilitado"):
				fmt.Println("   🚫 Consulta no disponible para este RUC")
			case strings.Contains(err.Error(), "sin anexos"):
				fmt.Println("   📝 No tiene establecimientos anexos registrados")
			default:
				fmt.Printf("   ❌ Error: %v\n", err)
			}
		} else {
			fmt.Printf("   ✅ Establecimientos anexos obtenidos\n")
			fmt.Printf("   🏭 Cantidad de anexos: %d\n", establecimientos.CantidadAnexos)

			if len(establecimientos.Establecimientos) > 0 {
				fmt.Printf("   📋 Detalles de establecimientos:\n")
				for i, est := range establecimientos.Establecimientos {
					fmt.Printf("      %d. [%s] %s\n", i+1, est.Codigo, est.TipoEstablecimiento)
					fmt.Printf("         📍 %s\n", est.Direccion)
					fmt.Printf("         💼 %s\n", est.ActividadEconomica)
					if i < len(establecimientos.Establecimientos)-1 {
						fmt.Printf("         %s\n", strings.Repeat("·", 30))
					}
				}
			}
		}

		// =================================================================
		// 6. OMISIONES TRIBUTARIAS
		// =================================================================
		fmt.Println("\n⚠️ 6. OMISIONES TRIBUTARIAS")
		fmt.Println(strings.Repeat("-", 40))

		omisiones, err := scraperExt.ScrapeOmisionesTributarias(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información de omisiones tributarias obtenida\n")
			if omisiones.TieneOmisiones {
				fmt.Printf("   🔴 Tiene omisiones: SÍ\n")
				fmt.Printf("   📄 Cantidad: %d\n", omisiones.CantidadOmisiones)
			} else {
				fmt.Printf("   🟢 Tiene omisiones: NO\n")
			}
		}

		// =================================================================
		// 7. ACTAS PROBATORIAS
		// =================================================================
		fmt.Println("\n📋 7. ACTAS PROBATORIAS")
		fmt.Println(strings.Repeat("-", 40))

		actas, err := scraperExt.ScrapeActasProbatorias(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información de actas probatorias obtenida\n")
			if actas.TieneActas {
				fmt.Printf("   📄 Cantidad de actas: %d\n", actas.CantidadActas)
				if len(actas.Actas) > 0 {
					fmt.Printf("   📋 Detalles de actas:\n")
					for i, acta := range actas.Actas {
						fmt.Printf("      %d. Acta N° %s\n", i+1, acta.NumeroActa)
						fmt.Printf("         📅 Fecha: %s\n", acta.FechaActa)
						fmt.Printf("         📍 Lugar: %s\n", acta.LugarIntervencion)
						if i < len(actas.Actas)-1 {
							fmt.Printf("         %s\n", strings.Repeat("·", 30))
						}
					}
				}
			} else {
				fmt.Printf("   🟢 No tiene actas probatorias\n")
			}
		}

		// =================================================================
		// 8. FACTURAS FÍSICAS
		// =================================================================
		fmt.Println("\n🧾 8. FACTURAS FÍSICAS")
		fmt.Println(strings.Repeat("-", 40))

		facturas, err := scraperExt.ScrapeFacturasFisicas(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información de facturas físicas obtenida\n")
			if facturas.TieneAutorizacion {
				fmt.Printf("   📄 Autorizaciones vigentes: %d\n", len(facturas.Autorizaciones))
				fmt.Printf("   🗑️ Canceladas/Bajas: %d\n", len(facturas.CanceladasOBajas))
			} else {
				fmt.Printf("   📝 No tiene autorización de facturas físicas\n")
			}
		}

		// =================================================================
		// 9. REACTIVA PERÚ
		// =================================================================
		fmt.Println("\n🇵🇪 9. REACTIVA PERÚ")
		fmt.Println(strings.Repeat("-", 40))

		reactiva, err := scraperExt.ScrapeReactivaPeru(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información Reactiva Perú obtenida\n")
			fmt.Printf("   🏢 Razón Social: %s\n", reactiva.RazonSocial)
			deudaStatus := map[bool]string{true: "SÍ", false: "NO"}[reactiva.TieneDeudaCoactiva]
			fmt.Printf("   💰 Tiene deuda coactiva: %s\n", deudaStatus)
			fmt.Printf("   📅 Última actualización: %s\n", reactiva.FechaActualizacion)
			fmt.Printf("   📖 Referencia legal: %s\n", reactiva.ReferenciaLegal)
		}

		// =================================================================
		// 10. PROGRAMA COVID-19
		// =================================================================
		fmt.Println("\n🦠 10. PROGRAMA COVID-19")
		fmt.Println(strings.Repeat("-", 40))

		covid, err := scraperExt.ScrapeProgramaCovid19(ruc)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ Información Programa COVID-19 obtenida\n")
			fmt.Printf("   🏢 Razón Social: %s\n", covid.RazonSocial)
			participaStatus := map[bool]string{true: "SÍ", false: "NO"}[covid.ParticipaPrograma]
			fmt.Printf("   🤝 Participa en programa: %s\n", participaStatus)
			deudaStatus := map[bool]string{true: "SÍ", false: "NO"}[covid.TieneDeudaCoactiva]
			fmt.Printf("   💰 Tiene deuda coactiva: %s\n", deudaStatus)
			fmt.Printf("   📅 Última actualización: %s\n", covid.FechaActualizacion)
			fmt.Printf("   📖 Base legal: %s\n", covid.BaseLegal)
		}

		// =================================================================
		// GUARDAR INFORMACIÓN COMPLETA
		// =================================================================
		fmt.Println("\n💾 GUARDANDO INFORMACIÓN COMPLETA...")
		fmt.Println(strings.Repeat("-", 40))

		// Crear estructura completa con toda la información
		rucCompleto := struct {
			RUC                    string                         `json:"ruc"`
			FechaConsultaCompleta  time.Time                      `json:"fecha_consulta_completa"`
			InformacionHistorica   *models.InformacionHistorica   `json:"informacion_historica,omitempty"`
			DeudaCoactiva          *models.DeudaCoactiva          `json:"deuda_coactiva,omitempty"`
			RepresentantesLegales  *models.RepresentantesLegales  `json:"representantes_legales,omitempty"`
			CantidadTrabajadores   *models.CantidadTrabajadores   `json:"cantidad_trabajadores,omitempty"`
			EstablecimientosAnexos *models.EstablecimientosAnexos `json:"establecimientos_anexos,omitempty"`
			OmisionesTributarias   *models.OmisionesTributarias   `json:"omisiones_tributarias,omitempty"`
			ActasProbatorias       *models.ActasProbatorias       `json:"actas_probatorias,omitempty"`
			FacturasFisicas        *models.FacturasFisicas        `json:"facturas_fisicas,omitempty"`
			ReactivaPeru           *models.ReactivaPeru           `json:"reactiva_peru,omitempty"`
			ProgramaCovid19        *models.ProgramaCovid19        `json:"programa_covid19,omitempty"`
		}{
			RUC:                   ruc,
			FechaConsultaCompleta: time.Now(),
		}

		// Asignar los datos obtenidos (solo los exitosos)
		if infoHistorica != nil {
			rucCompleto.InformacionHistorica = infoHistorica
		}
		if deudaCoactiva != nil {
			rucCompleto.DeudaCoactiva = deudaCoactiva
		}
		if representantes != nil {
			rucCompleto.RepresentantesLegales = representantes
		}
		if trabajadores != nil {
			rucCompleto.CantidadTrabajadores = trabajadores
		}
		if establecimientos != nil {
			rucCompleto.EstablecimientosAnexos = establecimientos
		}
		if omisiones != nil {
			rucCompleto.OmisionesTributarias = omisiones
		}
		if actas != nil {
			rucCompleto.ActasProbatorias = actas
		}
		if facturas != nil {
			rucCompleto.FacturasFisicas = facturas
		}
		if reactiva != nil {
			rucCompleto.ReactivaPeru = reactiva
		}
		if covid != nil {
			rucCompleto.ProgramaCovid19 = covid
		}

		// Guardar archivo JSON completo
		fileName := fmt.Sprintf("ruc_completo_%s_%s.json",
			ruc,
			time.Now().Format("20060102_150405"))

		jsonData, err := json.MarshalIndent(rucCompleto, "", "  ")
		if err != nil {
			log.Printf("❌ Error marshaling data for RUC %s: %v\n", ruc, err)
			continue
		}

		err = os.WriteFile(fileName, jsonData, 0644)
		if err != nil {
			log.Printf("❌ Error saving data for RUC %s: %v\n", ruc, err)
			continue
		}

		fmt.Printf("   ✅ Información completa guardada en: %s\n", fileName)

		// =================================================================
		// RESUMEN FINAL
		// =================================================================
		fmt.Println("\n📊 RESUMEN FINAL")
		fmt.Println(strings.Repeat("-", 40))

		consultasExitosas := 0
		consultasConError := 0

		funciones := []struct {
			nombre string
			exito  bool
		}{
			{"Información Histórica", infoHistorica != nil},
			{"Deuda Coactiva", deudaCoactiva != nil},
			{"Representantes Legales", representantes != nil},
			{"Cantidad Trabajadores", trabajadores != nil},
			{"Establecimientos Anexos", establecimientos != nil},
			{"Omisiones Tributarias", omisiones != nil},
			{"Actas Probatorias", actas != nil},
			{"Facturas Físicas", facturas != nil},
			{"Reactiva Perú", reactiva != nil},
			{"Programa COVID-19", covid != nil},
		}

		for _, f := range funciones {
			if f.exito {
				consultasExitosas++
				fmt.Printf("   ✅ %s\n", f.nombre)
			} else {
				consultasConError++
				fmt.Printf("   ❌ %s\n", f.nombre)
			}
		}

		fmt.Printf("\n   📈 Consultas exitosas: %d/%d\n", consultasExitosas, len(funciones))
		fmt.Printf("   📉 Consultas con error: %d/%d\n", consultasConError, len(funciones))
		fmt.Printf("   📁 Archivo generado: %s\n", fileName)

		fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	}

	fmt.Printf("\n🎉 PROCESO COMPLETADO - Navegador optimizado mantenido durante toda la ejecución\n")
}
