package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/consulta-ruc-scraper/pkg/models"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService(connectionString string) (*DatabaseService, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	service := &DatabaseService{db: db}
	return service, nil
}

func (ds *DatabaseService) Close() error {
	return ds.db.Close()
}

func (ds *DatabaseService) InsertRUCCompleto(ruc *models.RUCCompleto) error {
	tx, err := ds.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Insertar información básica
	rucID, err := ds.insertRUCInformacionBasica(tx, ruc)
	if err != nil {
		return fmt.Errorf("error inserting RUC info: %w", err)
	}

	// 2. Insertar actividades económicas
	if err := ds.insertActividadesEconomicas(tx, rucID, ruc.InformacionBasica.ActividadesEconomicas); err != nil {
		return fmt.Errorf("error inserting actividades economicas: %w", err)
	}

	// 3. Insertar comprobantes de pago
	if err := ds.insertComprobantesPago(tx, rucID, ruc.InformacionBasica.ComprobantesPago); err != nil {
		return fmt.Errorf("error inserting comprobantes pago: %w", err)
	}

	// 4. Insertar sistemas de emisión electrónica
	if err := ds.insertSistemasEmisionElectronica(tx, rucID, ruc.InformacionBasica.SistemaEmisionElectronica); err != nil {
		return fmt.Errorf("error inserting sistemas emision electronica: %w", err)
	}

	// 5. Insertar comprobantes electrónicos
	if err := ds.insertComprobantesElectronicos(tx, rucID, ruc.InformacionBasica.ComprobantesElectronicos); err != nil {
		return fmt.Errorf("error inserting comprobantes electronicos: %w", err)
	}

	// 6. Insertar padrones
	if err := ds.insertPadrones(tx, rucID, ruc.InformacionBasica.Padrones); err != nil {
		return fmt.Errorf("error inserting padrones: %w", err)
	}

	// 7. Insertar consulta
	if err := ds.insertConsulta(tx, rucID, ruc); err != nil {
		return fmt.Errorf("error inserting consulta: %w", err)
	}

	// 8. Insertar información histórica
	if ruc.InformacionHistorica != nil {
		if err := ds.insertInformacionHistorica(tx, rucID, ruc.InformacionHistorica); err != nil {
			return fmt.Errorf("error inserting informacion historica: %w", err)
		}
	}

	// 9. Insertar deuda coactiva
	if ruc.DeudaCoactiva != nil {
		if err := ds.insertDeudaCoactiva(tx, rucID, ruc.DeudaCoactiva); err != nil {
			return fmt.Errorf("error inserting deuda coactiva: %w", err)
		}
	}

	// 10. Insertar omisiones tributarias
	if ruc.OmisionesTributarias != nil {
		if err := ds.insertOmisionesTributarias(tx, rucID, ruc.OmisionesTributarias); err != nil {
			return fmt.Errorf("error inserting omisiones tributarias: %w", err)
		}
	}

	// 11. Insertar cantidad trabajadores
	if ruc.CantidadTrabajadores != nil {
		if err := ds.insertCantidadTrabajadores(tx, rucID, ruc.CantidadTrabajadores); err != nil {
			return fmt.Errorf("error inserting cantidad trabajadores: %w", err)
		}
	}

	// 12. Insertar actas probatorias
	if ruc.ActasProbatorias != nil {
		if err := ds.insertActasProbatorias(tx, rucID, ruc.ActasProbatorias); err != nil {
			return fmt.Errorf("error inserting actas probatorias: %w", err)
		}
	}

	// 13. Insertar facturas físicas
	if ruc.FacturasFisicas != nil {
		if err := ds.insertFacturasFisicas(tx, rucID, ruc.FacturasFisicas); err != nil {
			return fmt.Errorf("error inserting facturas fisicas: %w", err)
		}
	}

	// 14. Insertar reactiva peru
	if ruc.ReactivaPeru != nil {
		if err := ds.insertReactivaPeru(tx, rucID, ruc.ReactivaPeru); err != nil {
			return fmt.Errorf("error inserting reactiva peru: %w", err)
		}
	}

	// 15. Insertar programa covid19
	if ruc.ProgramaCovid19 != nil {
		if err := ds.insertProgramaCovid19(tx, rucID, ruc.ProgramaCovid19); err != nil {
			return fmt.Errorf("error inserting programa covid19: %w", err)
		}
	}

	// 16. Insertar representantes legales
	if ruc.RepresentantesLegales != nil {
		if err := ds.insertRepresentantesLegales(tx, rucID, ruc.RepresentantesLegales); err != nil {
			return fmt.Errorf("error inserting representantes legales: %w", err)
		}
	}

	// 17. Insertar establecimientos anexos
	if ruc.EstablecimientosAnexos != nil {
		if err := ds.insertEstablecimientosAnexos(tx, rucID, ruc.EstablecimientosAnexos); err != nil {
			return fmt.Errorf("error inserting establecimientos anexos: %w", err)
		}
	}

	return tx.Commit()
}

func (ds *DatabaseService) insertRUCInformacionBasica(tx *sql.Tx, ruc *models.RUCCompleto) (int64, error) {
	query := `
	INSERT INTO ruc_informacion_basica (
		ruc, razon_social, tipo_contribuyente, tipo_documento, nombre_comercial,
		fecha_inscripcion, fecha_inicio_actividades, estado, condicion,
		domicilio_fiscal, sistema_emision, actividad_comercio_exterior,
		sistema_contabilidad, emisor_electronico_desde, afiliado_ple
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
	)
	ON CONFLICT (ruc) DO UPDATE SET
		razon_social = EXCLUDED.razon_social,
		tipo_contribuyente = EXCLUDED.tipo_contribuyente,
		tipo_documento = EXCLUDED.tipo_documento,
		nombre_comercial = EXCLUDED.nombre_comercial,
		fecha_inscripcion = EXCLUDED.fecha_inscripcion,
		fecha_inicio_actividades = EXCLUDED.fecha_inicio_actividades,
		estado = EXCLUDED.estado,
		condicion = EXCLUDED.condicion,
		domicilio_fiscal = EXCLUDED.domicilio_fiscal,
		sistema_emision = EXCLUDED.sistema_emision,
		actividad_comercio_exterior = EXCLUDED.actividad_comercio_exterior,
		sistema_contabilidad = EXCLUDED.sistema_contabilidad,
		emisor_electronico_desde = EXCLUDED.emisor_electronico_desde,
		afiliado_ple = EXCLUDED.afiliado_ple,
		updated_at = CURRENT_TIMESTAMP
	RETURNING id`

	var rucID int64
	err := tx.QueryRow(query,
		ruc.InformacionBasica.RUC,
		ds.nullString(ruc.InformacionBasica.RazonSocial),
		ds.nullString(ruc.InformacionBasica.TipoContribuyente),
		ds.nullString(ruc.InformacionBasica.TipoDocumento),
		ds.nullString(ruc.InformacionBasica.NombreComercial),
		ds.parseDate(ruc.InformacionBasica.FechaInscripcion),
		ds.parseDate(ruc.InformacionBasica.FechaInicioActividades),
		ds.nullString(ruc.InformacionBasica.Estado),
		ds.nullString(ruc.InformacionBasica.Condicion),
		ds.nullString(ruc.InformacionBasica.DomicilioFiscal),
		ds.nullString(ruc.InformacionBasica.SistemaEmision),
		ds.nullString(ruc.InformacionBasica.ActividadComercioExterior),
		ds.nullString(ruc.InformacionBasica.SistemaContabilidad),
		ds.parseDate(ruc.InformacionBasica.EmisorElectronicoDesde),
		ds.nullString(ruc.InformacionBasica.AfiliadoPLE),
	).Scan(&rucID)

	return rucID, err
}

func (ds *DatabaseService) insertActividadesEconomicas(tx *sql.Tx, rucID int64, actividades []string) error {
	_, err := tx.Exec("DELETE FROM ruc_actividades_economicas WHERE ruc_id = $1", rucID)
	if err != nil {
		return err
	}

	for _, actividad := range actividades {
		_, err := tx.Exec(`
			INSERT INTO ruc_actividades_economicas (ruc_id, actividad_economica)
			VALUES ($1, $2)`,
			rucID, actividad)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *DatabaseService) insertComprobantesPago(tx *sql.Tx, rucID int64, comprobantes []string) error {
	_, err := tx.Exec("DELETE FROM ruc_comprobantes_pago WHERE ruc_id = $1", rucID)
	if err != nil {
		return err
	}

	for _, comprobante := range comprobantes {
		_, err := tx.Exec(`
			INSERT INTO ruc_comprobantes_pago (ruc_id, comprobante_pago)
			VALUES ($1, $2)`,
			rucID, comprobante)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *DatabaseService) insertSistemasEmisionElectronica(tx *sql.Tx, rucID int64, sistemas []string) error {
	_, err := tx.Exec("DELETE FROM ruc_sistemas_emision_electronica WHERE ruc_id = $1", rucID)
	if err != nil {
		return err
	}

	for _, sistema := range sistemas {
		_, err := tx.Exec(`
			INSERT INTO ruc_sistemas_emision_electronica (ruc_id, sistema_emision)
			VALUES ($1, $2)`,
			rucID, sistema)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *DatabaseService) insertComprobantesElectronicos(tx *sql.Tx, rucID int64, comprobantes []string) error {
	_, err := tx.Exec("DELETE FROM ruc_comprobantes_electronicos WHERE ruc_id = $1", rucID)
	if err != nil {
		return err
	}

	for _, comprobante := range comprobantes {
		_, err := tx.Exec(`
			INSERT INTO ruc_comprobantes_electronicos (ruc_id, comprobante_electronico)
			VALUES ($1, $2)`,
			rucID, comprobante)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *DatabaseService) insertPadrones(tx *sql.Tx, rucID int64, padrones []string) error {
	_, err := tx.Exec("DELETE FROM ruc_padrones WHERE ruc_id = $1", rucID)
	if err != nil {
		return err
	}

	for _, padron := range padrones {
		_, err := tx.Exec(`
			INSERT INTO ruc_padrones (ruc_id, padron)
			VALUES ($1, $2)`,
			rucID, padron)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ds *DatabaseService) insertConsulta(tx *sql.Tx, rucID int64, ruc *models.RUCCompleto) error {
	_, err := tx.Exec(`
		INSERT INTO ruc_consultas (ruc_id, fecha_consulta, version_api)
		VALUES ($1, $2, $3)`,
		rucID, ruc.FechaConsulta, ruc.VersionAPI)
	return err
}

func (ds *DatabaseService) insertInformacionHistorica(tx *sql.Tx, rucID int64, info *models.InformacionHistorica) error {
	// Insertar registro principal
	var histID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_informacion_historica (ruc_id)
		VALUES ($1) RETURNING id`, rucID).Scan(&histID)
	if err != nil {
		return err
	}

	// Insertar razones sociales históricas
	for _, razon := range info.RazonesSociales {
		_, err := tx.Exec(`
			INSERT INTO ruc_razones_sociales_historicas (informacion_historica_id, nombre, fecha_de_baja)
			VALUES ($1, $2, $3)`,
			histID, ds.nullString(razon.Nombre), ds.parseDate(razon.FechaDeBaja))
		if err != nil {
			return err
		}
	}

	// Insertar condiciones históricas
	for _, condicion := range info.Condiciones {
		_, err := tx.Exec(`
			INSERT INTO ruc_condiciones_historicas (informacion_historica_id, condicion, desde, hasta)
			VALUES ($1, $2, $3, $4)`,
			histID, ds.nullString(condicion.Condicion),
			ds.parseDate(condicion.Desde), ds.parseDate(condicion.Hasta))
		if err != nil {
			return err
		}
	}

	// Insertar domicilios históricos
	for _, domicilio := range info.Domicilios {
		_, err := tx.Exec(`
			INSERT INTO ruc_domicilios_fiscales_historicos (informacion_historica_id, direccion, fecha_de_baja)
			VALUES ($1, $2, $3)`,
			histID, ds.nullString(domicilio.Direccion), ds.parseDate(domicilio.FechaDeBaja))
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertDeudaCoactiva(tx *sql.Tx, rucID int64, deuda *models.DeudaCoactiva) error {
	// Insertar registro principal
	var deudaID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_deuda_coactiva (ruc_id, total_deuda, cantidad_documentos)
		VALUES ($1, $2, $3) RETURNING id`,
		rucID, deuda.TotalDeuda, deuda.CantidadDocumentos).Scan(&deudaID)
	if err != nil {
		return err
	}

	// Insertar detalle de deudas
	for _, detalle := range deuda.Deudas {
		_, err := tx.Exec(`
			INSERT INTO ruc_detalle_deudas (deuda_coactiva_id, monto, periodo_tributario, fecha_inicio_cobranza, entidad)
			VALUES ($1, $2, $3, $4, $5)`,
			deudaID, detalle.Monto, detalle.PeriodoTributario,
			ds.parseDate(detalle.FechaInicioCobranza), detalle.Entidad)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertOmisionesTributarias(tx *sql.Tx, rucID int64, omisiones *models.OmisionesTributarias) error {
	// Insertar registro principal
	var omisionesID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_omisiones_tributarias (ruc_id, tiene_omisiones, cantidad_omisiones)
		VALUES ($1, $2, $3) RETURNING id`,
		rucID, omisiones.TieneOmisiones, omisiones.CantidadOmisiones).Scan(&omisionesID)
	if err != nil {
		return err
	}

	// Insertar detalle de omisiones
	for _, omision := range omisiones.Omisiones {
		_, err := tx.Exec(`
			INSERT INTO ruc_omisiones (omisiones_tributarias_id, periodo, tributo, tipo_declaracion, fecha_vencimiento, estado)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			omisionesID, omision.Periodo, omision.Tributo, omision.TipoDeclaracion,
			ds.parseDate(omision.FechaVencimiento), omision.Estado)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertCantidadTrabajadores(tx *sql.Tx, rucID int64, trabajadores *models.CantidadTrabajadores) error {
	// Insertar registro principal
	var trabajadoresID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_cantidad_trabajadores (ruc_id)
		VALUES ($1) RETURNING id`, rucID).Scan(&trabajadoresID)
	if err != nil {
		return err
	}

	// Insertar períodos disponibles
	for _, periodo := range trabajadores.PeriodosDisponibles {
		_, err := tx.Exec(`
			INSERT INTO ruc_periodos_disponibles_trabajadores (cantidad_trabajadores_id, periodo)
			VALUES ($1, $2)`,
			trabajadoresID, periodo)
		if err != nil {
			return err
		}
	}

	// Insertar detalle por período
	for _, detalle := range trabajadores.DetallePorPeriodo {
		_, err := tx.Exec(`
			INSERT INTO ruc_detalle_trabajadores (
				cantidad_trabajadores_id, periodo, cantidad_trabajadores, 
				cantidad_prestadores_servicio, cantidad_pensionistas, total
			) VALUES ($1, $2, $3, $4, $5, $6)`,
			trabajadoresID, detalle.Periodo, detalle.CantidadTrabajadores,
			detalle.CantidadPrestadoresServicio, detalle.CantidadPensionistas, detalle.Total)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertActasProbatorias(tx *sql.Tx, rucID int64, actas *models.ActasProbatorias) error {
	// Insertar registro principal
	var actasID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_actas_probatorias (ruc_id, tiene_actas, cantidad_actas)
		VALUES ($1, $2, $3) RETURNING id`,
		rucID, actas.TieneActas, actas.CantidadActas).Scan(&actasID)
	if err != nil {
		return err
	}

	// Insertar detalle de actas
	for _, acta := range actas.Actas {
		_, err := tx.Exec(`
			INSERT INTO ruc_actas (
				actas_probatorias_id, numero_acta, fecha_acta, lugar_intervencion,
				articulo_numeral, descripcion_infraccion, numero_ri_roz, tipo_ri_roz, acta_reconocimiento
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			actasID, acta.NumeroActa, ds.parseDate(acta.FechaActa), acta.LugarIntervencion,
			acta.ArticuloNumeral, acta.DescripcionInfraccion, acta.NumeroRIROZ,
			acta.TipoRIROZ, acta.ActaReconocimiento)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertFacturasFisicas(tx *sql.Tx, rucID int64, facturas *models.FacturasFisicas) error {
	// Insertar registro principal
	var facturasID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_facturas_fisicas (ruc_id, tiene_autorizacion)
		VALUES ($1, $2) RETURNING id`,
		rucID, facturas.TieneAutorizacion).Scan(&facturasID)
	if err != nil {
		return err
	}

	// Insertar facturas autorizadas
	for _, autorizada := range facturas.Autorizaciones {
		_, err := tx.Exec(`
			INSERT INTO ruc_facturas_autorizadas (
				facturas_fisicas_id, numero_autorizacion, fecha_autorizacion, 
				tipo_comprobante, serie, numero_inicial, numero_final
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			facturasID, autorizada.NumeroAutorizacion, ds.parseDate(autorizada.FechaAutorizacion),
			autorizada.TipoComprobante, autorizada.Serie, autorizada.NumeroInicial, autorizada.NumeroFinal)
		if err != nil {
			return err
		}
	}

	// Insertar facturas canceladas o de baja
	for _, cancelada := range facturas.CanceladasOBajas {
		_, err := tx.Exec(`
			INSERT INTO ruc_facturas_canceladas_bajas (
				facturas_fisicas_id, numero_autorizacion, fecha_autorizacion,
				tipo_comprobante, serie, numero_inicial, numero_final
			) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			facturasID, cancelada.NumeroAutorizacion, ds.parseDate(cancelada.FechaAutorizacion),
			cancelada.TipoComprobante, cancelada.Serie, cancelada.NumeroInicial, cancelada.NumeroFinal)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertReactivaPeru(tx *sql.Tx, rucID int64, reactiva *models.ReactivaPeru) error {
	_, err := tx.Exec(`
		INSERT INTO ruc_reactiva_peru (ruc_id, razon_social, tiene_deuda_coactiva, fecha_actualizacion, referencia_legal)
		VALUES ($1, $2, $3, $4, $5)`,
		rucID, ds.nullString(reactiva.RazonSocial), reactiva.TieneDeudaCoactiva,
		ds.parseDate(reactiva.FechaActualizacion), ds.nullString(reactiva.ReferenciaLegal))
	return err
}

func (ds *DatabaseService) insertProgramaCovid19(tx *sql.Tx, rucID int64, covid *models.ProgramaCovid19) error {
	_, err := tx.Exec(`
		INSERT INTO ruc_programa_covid19 (ruc_id, razon_social, participa_programa, tiene_deuda_coactiva, fecha_actualizacion, base_legal)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		rucID, ds.nullString(covid.RazonSocial), covid.ParticipaPrograma, covid.TieneDeudaCoactiva,
		ds.parseDate(covid.FechaActualizacion), ds.nullString(covid.BaseLegal))
	return err
}

func (ds *DatabaseService) insertRepresentantesLegales(tx *sql.Tx, rucID int64, representantes *models.RepresentantesLegales) error {
	// Insertar registro principal
	var representantesID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_representantes_legales (ruc_id)
		VALUES ($1) RETURNING id`, rucID).Scan(&representantesID)
	if err != nil {
		return err
	}

	// Insertar representantes
	for _, rep := range representantes.Representantes {
		_, err := tx.Exec(`
			INSERT INTO ruc_representantes (
				representantes_legales_id, tipo_documento, numero_documento,
				nombre_completo, cargo, fecha_desde, fecha_hasta, vigente
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			representantesID, rep.TipoDocumento, rep.NumeroDocumento, rep.NombreCompleto,
			rep.Cargo, ds.parseDate(rep.FechaDesde), ds.parseDate(rep.FechaHasta), rep.Vigente)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ds *DatabaseService) insertEstablecimientosAnexos(tx *sql.Tx, rucID int64, establecimientos *models.EstablecimientosAnexos) error {
	// Insertar registro principal
	var establecimientosID int64
	err := tx.QueryRow(`
		INSERT INTO ruc_establecimientos_anexos (ruc_id, cantidad_anexos)
		VALUES ($1, $2) RETURNING id`,
		rucID, establecimientos.CantidadAnexos).Scan(&establecimientosID)
	if err != nil {
		return err
	}

	// Insertar establecimientos
	for _, est := range establecimientos.Establecimientos {
		_, err := tx.Exec(`
			INSERT INTO ruc_establecimientos (
				establecimientos_anexos_id, codigo, tipo_establecimiento, direccion, actividad_economica
			) VALUES ($1, $2, $3, $4, $5)`,
			establecimientosID, est.Codigo, est.TipoEstablecimiento,
			ds.nullString(est.Direccion), ds.nullString(est.ActividadEconomica))
		if err != nil {
			return err
		}
	}

	return nil
}

// Helper functions
func (ds *DatabaseService) nullString(s string) interface{} {
	if s == "" || s == "-" || s == "No hay información" {
		return nil
	}
	return s
}

func (ds *DatabaseService) parseDate(dateStr string) interface{} {
	if dateStr == "" || dateStr == "-" || dateStr == "No hay información" {
		return nil
	}

	formats := []string{
		"02/01/2006",
		"2006-01-02",
		"01/02/2006",
	}

	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date
		}
	}

	return nil
}

func (ds *DatabaseService) GetRUCByNumber(rucNumber string) (*models.RUCCompleto, error) {
	return nil, fmt.Errorf("GetRUCByNumber not implemented for normalized schema")
}
