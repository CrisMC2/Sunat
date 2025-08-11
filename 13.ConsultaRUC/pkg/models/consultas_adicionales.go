package models

import "time"

// InformacionHistorica representa los cambios históricos del RUC
type InformacionHistorica struct {
	RUC                   string                      `json:"ruc"`
	CambiosRazonSocial    []CambioHistorico          `json:"cambios_razon_social"`
	CambiosDomicilio      []CambioHistorico          `json:"cambios_domicilio"`
	CambiosEstado         []CambioEstado             `json:"cambios_estado"`
	CambiosCondicion      []CambioHistorico          `json:"cambios_condicion"`
	ActividadesHistoricas []ActividadEconomicaHistorica `json:"actividades_historicas"`
}

type CambioHistorico struct {
	Fecha        string `json:"fecha"`
	ValorAnterior string `json:"valor_anterior"`
	ValorNuevo   string `json:"valor_nuevo"`
	Motivo       string `json:"motivo,omitempty"`
}

type CambioEstado struct {
	Fecha        string `json:"fecha"`
	EstadoAnterior string `json:"estado_anterior"`
	EstadoNuevo    string `json:"estado_nuevo"`
	Motivo         string `json:"motivo"`
}

type ActividadEconomicaHistorica struct {
	Fecha            string `json:"fecha"`
	TipoMovimiento   string `json:"tipo_movimiento"` // ALTA, BAJA, MODIFICACION
	CodigoActividad  string `json:"codigo_actividad"`
	DescripcionActividad string `json:"descripcion_actividad"`
}

// DeudaCoactiva representa las deudas en cobranza coactiva
type DeudaCoactiva struct {
	RUC                string           `json:"ruc"`
	TotalDeuda         float64          `json:"total_deuda"`
	CantidadDocumentos int              `json:"cantidad_documentos"`
	Deudas             []DetalleDeuda   `json:"deudas"`
	FechaConsulta      time.Time        `json:"fecha_consulta"`
}

type DetalleDeuda struct {
	NumeroDocumento    string    `json:"numero_documento"`
	TipoDocumento      string    `json:"tipo_documento"`
	FechaEmision       string    `json:"fecha_emision"`
	Periodo            string    `json:"periodo"`
	Tributo            string    `json:"tributo"`
	MontoOriginal      float64   `json:"monto_original"`
	MontoActualizado   float64   `json:"monto_actualizado"`
	Estado             string    `json:"estado"`
}

// OmisionesTributarias representa las omisiones tributarias del contribuyente
type OmisionesTributarias struct {
	RUC                 string            `json:"ruc"`
	TieneOmisiones      bool              `json:"tiene_omisiones"`
	CantidadOmisiones   int               `json:"cantidad_omisiones"`
	Omisiones           []Omision         `json:"omisiones"`
	FechaConsulta       time.Time         `json:"fecha_consulta"`
}

type Omision struct {
	Periodo             string    `json:"periodo"`
	Tributo             string    `json:"tributo"`
	TipoDeclaracion     string    `json:"tipo_declaracion"`
	FechaVencimiento    string    `json:"fecha_vencimiento"`
	Estado              string    `json:"estado"`
}

// CantidadTrabajadores representa información de trabajadores y prestadores
type CantidadTrabajadores struct {
	RUC                        string                     `json:"ruc"`
	PeriodosDisponibles        []string                   `json:"periodos_disponibles"`
	DetallePorPeriodo          []DetalleTrabajadores      `json:"detalle_por_periodo"`
	FechaConsulta              time.Time                  `json:"fecha_consulta"`
}

type DetalleTrabajadores struct {
	Periodo                    string    `json:"periodo"`
	CantidadTrabajadores       int       `json:"cantidad_trabajadores"`
	CantidadPrestadoresServicio int      `json:"cantidad_prestadores_servicio"`
	CantidadPracticantes       int       `json:"cantidad_practicantes"`
	Total                      int       `json:"total"`
}

// ActasProbatorias representa las actas probatorias del contribuyente
type ActasProbatorias struct {
	RUC               string           `json:"ruc"`
	TieneActas        bool             `json:"tiene_actas"`
	CantidadActas     int              `json:"cantidad_actas"`
	Actas             []ActaProbatoria `json:"actas"`
	FechaConsulta     time.Time        `json:"fecha_consulta"`
}

type ActaProbatoria struct {
	NumeroActa        string    `json:"numero_acta"`
	FechaActa         string    `json:"fecha_acta"`
	TipoInfraccion    string    `json:"tipo_infraccion"`
	DescripcionHechos string    `json:"descripcion_hechos"`
	MontoMulta        float64   `json:"monto_multa,omitempty"`
	Estado            string    `json:"estado"`
}

// FacturasFisicas representa información sobre facturas físicas autorizadas
type FacturasFisicas struct {
	RUC                    string                `json:"ruc"`
	TieneAutorizacion      bool                  `json:"tiene_autorizacion"`
	Autorizaciones         []AutorizacionFactura `json:"autorizaciones"`
	FechaConsulta          time.Time             `json:"fecha_consulta"`
}

type AutorizacionFactura struct {
	NumeroAutorizacion     string    `json:"numero_autorizacion"`
	FechaAutorizacion      string    `json:"fecha_autorizacion"`
	TipoComprobante        string    `json:"tipo_comprobante"`
	Serie                  string    `json:"serie"`
	NumeroInicial          string    `json:"numero_inicial"`
	NumeroFinal            string    `json:"numero_final"`
	FechaVencimiento       string    `json:"fecha_vencimiento"`
	Estado                 string    `json:"estado"`
}

// ReactivaPeru representa información del programa Reactiva Perú
type ReactivaPeru struct {
	RUC                    string            `json:"ruc"`
	ParticipaProgramma     bool              `json:"participa_programa"`
	TieneDeudaCoactiva     bool              `json:"tiene_deuda_coactiva"`
	MontoDeudaPrograma     float64           `json:"monto_deuda_programa"`
	DetalleCreditos        []CreditoReactiva `json:"detalle_creditos"`
	FechaConsulta          time.Time         `json:"fecha_consulta"`
}

type CreditoReactiva struct {
	EntidadFinanciera      string    `json:"entidad_financiera"`
	MontoCredito           float64   `json:"monto_credito"`
	FechaDesembolso        string    `json:"fecha_desembolso"`
	Estado                 string    `json:"estado"`
	SaldoPendiente         float64   `json:"saldo_pendiente"`
}

// ProgramaCovid19 representa información del programa de garantías COVID-19
type ProgramaCovid19 struct {
	RUC                    string          `json:"ruc"`
	ParticipaProgramma     bool            `json:"participa_programa"`
	TieneDeudaCoactiva     bool            `json:"tiene_deuda_coactiva"`
	MontoTotalGarantizado  float64         `json:"monto_total_garantizado"`
	DetalleGarantias       []GarantiaCovid `json:"detalle_garantias"`
	FechaConsulta          time.Time       `json:"fecha_consulta"`
}

type GarantiaCovid struct {
	EntidadFinanciera      string    `json:"entidad_financiera"`
	TipoGarantia           string    `json:"tipo_garantia"`
	MontoGarantia          float64   `json:"monto_garantia"`
	FechaOtorgamiento      string    `json:"fecha_otorgamiento"`
	Estado                 string    `json:"estado"`
}

// RepresentantesLegales representa los representantes legales
type RepresentantesLegales struct {
	RUC                    string                 `json:"ruc"`
	Representantes         []RepresentanteLegal   `json:"representantes"`
	FechaConsulta          time.Time              `json:"fecha_consulta"`
}

type RepresentanteLegal struct {
	TipoDocumento          string    `json:"tipo_documento"`
	NumeroDocumento        string    `json:"numero_documento"`
	ApellidoPaterno        string    `json:"apellido_paterno"`
	ApellidoMaterno        string    `json:"apellido_materno"`
	Nombres                string    `json:"nombres"`
	Cargo                  string    `json:"cargo"`
	FechaDesde             string    `json:"fecha_desde"`
	FechaHasta             string    `json:"fecha_hasta,omitempty"`
	Vigente                bool      `json:"vigente"`
}

// EstablecimientosAnexos representa los establecimientos anexos
type EstablecimientosAnexos struct {
	RUC                    string                `json:"ruc"`
	CantidadAnexos         int                   `json:"cantidad_anexos"`
	Establecimientos       []EstablecimientoAnexo `json:"establecimientos"`
	FechaConsulta          time.Time              `json:"fecha_consulta"`
}

type EstablecimientoAnexo struct {
	CodigoEstablecimiento  string    `json:"codigo_establecimiento"`
	TipoEstablecimiento    string    `json:"tipo_establecimiento"`
	DenominacionComercial  string    `json:"denominacion_comercial,omitempty"`
	Direccion              string    `json:"direccion"`
	Departamento           string    `json:"departamento"`
	Provincia              string    `json:"provincia"`
	Distrito               string    `json:"distrito"`
	Estado                 string    `json:"estado"`
	FechaAlta              string    `json:"fecha_alta"`
	FechaBaja              string    `json:"fecha_baja,omitempty"`
	ActividadesEconomicas  []string  `json:"actividades_economicas"`
}