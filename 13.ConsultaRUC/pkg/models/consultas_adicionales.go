package models

// InformacionHistorica representa los cambios históricos del RUC
type InformacionHistorica struct {
	FechaActualizada string                     `json:"fecha_actualizada"` // "07/08/2025"
	RazonesSociales  []RazonSocialHistorica     `json:"razones_sociales"`
	Condiciones      []CondicionHistorica       `json:"condiciones"`
	Domicilios       []DomicilioFiscalHistorico `json:"domicilios"`
}

type RazonSocialHistorica struct {
	Nombre      string `json:"nombre"`        // o "No hay Información"
	FechaDeBaja string `json:"fecha_de_baja"` // "-" si no hay
}

type CondicionHistorica struct {
	Condicion string `json:"condicion"` // Ej: "HABIDO" o "-"
	Desde     string `json:"desde"`     // Ej: "13/08/2020"
	Hasta     string `json:"hasta"`     // Ej: "01/07/2022"
}

type DomicilioFiscalHistorico struct {
	Direccion   string `json:"direccion"`
	FechaDeBaja string `json:"fecha_de_baja"` // Ej: "01/07/2022"
}

// DeudaCoactiva representa las deudas en cobranza coactiva
type DeudaCoactiva struct {
	TotalDeuda         float64        `json:"total_deuda"`
	CantidadDocumentos int            `json:"cantidad_documentos"`
	Deudas             []DetalleDeuda `json:"deudas"`
}

// DetalleDeuda representa una fila de deuda
type DetalleDeuda struct {
	Monto               float64 `json:"monto"`
	PeriodoTributario   string  `json:"periodo_tributario"`
	FechaInicioCobranza string  `json:"fecha_inicio_cobranza"`
	Entidad             string  `json:"entidad"`
}

// OmisionesTributarias representa las omisiones tributarias del contribuyente
type OmisionesTributarias struct {
	TieneOmisiones    bool      `json:"tiene_omisiones"`
	CantidadOmisiones int       `json:"cantidad_omisiones"`
	Omisiones         []Omision `json:"omisiones"`
}

type Omision struct {
	Periodo          string `json:"periodo"`
	Tributo          string `json:"tributo"`
	TipoDeclaracion  string `json:"tipo_declaracion"`
	FechaVencimiento string `json:"fecha_vencimiento"`
	Estado           string `json:"estado"`
}

// CantidadTrabajadores representa información de trabajadores y prestadores
type CantidadTrabajadores struct {
	PeriodosDisponibles []string              `json:"periodos_disponibles"`
	DetallePorPeriodo   []DetalleTrabajadores `json:"detalle_por_periodo"`
}

type DetalleTrabajadores struct {
	Periodo                     string `json:"periodo"`
	CantidadTrabajadores        int    `json:"cantidad_trabajadores"`
	CantidadPrestadoresServicio int    `json:"cantidad_prestadores_servicio"`
	CantidadPensionistas        int    `json:"cantidad_pensionistas"`
	Total                       int    `json:"total"`
}

// ActasProbatorias representa las actas probatorias del contribuyente
type ActasProbatorias struct {
	TieneActas    bool             `json:"tiene_actas"`
	CantidadActas int              `json:"cantidad_actas"`
	Actas         []ActaProbatoria `json:"actas"`
}

type ActaProbatoria struct {
	NumeroActa            string `json:"numero_acta"`            // Nº Acta Probatoria
	FechaActa             string `json:"fecha_acta"`             // Fecha de Acta Probatoria
	LugarIntervencion     string `json:"lugar_intervencion"`     // Lugar de Intervención
	ArticuloNumeral       string `json:"articulo_numeral"`       // Artículo y Numeral de la infracción
	DescripcionInfraccion string `json:"descripcion_infraccion"` // Descripción de la infracción
	NumeroRIROZ           string `json:"numero_ri_roz"`          // Nº de RI/ROZ
	TipoRIROZ             string `json:"tipo_ri_roz"`            // Tipo de RI/ROZ
	ActaReconocimiento    string `json:"acta_reconocimiento"`    // Acta de Reconocimiento
}

// FacturasFisicas representa información sobre facturas físicas autorizadas
type FacturasFisicas struct {
	TieneAutorizacion bool                    `json:"tiene_autorizacion"`
	Autorizaciones    []FacturaAutorizada     `json:"autorizaciones"`
	CanceladasOBajas  []FacturaBajaOCancelada `json:"canceladas_o_bajas"`
}

type FacturaAutorizada struct {
	NumeroAutorizacion string `json:"numero_autorizacion"`
	FechaAutorizacion  string `json:"fecha_autorizacion"`
	TipoComprobante    string `json:"tipo_comprobante"`
	Serie              string `json:"serie"`
	NumeroInicial      string `json:"numero_inicial"`
	NumeroFinal        string `json:"numero_final"`
}

type FacturaBajaOCancelada struct {
	NumeroAutorizacion string `json:"numero_autorizacion"`
	FechaAutorizacion  string `json:"fecha_autorizacion"`
	TipoComprobante    string `json:"tipo_comprobante"`
	Serie              string `json:"serie"`
	NumeroInicial      string `json:"numero_inicial"`
	NumeroFinal        string `json:"numero_final"`
}

// ReactivaPeru representa información del programa Reactiva Perú
type ReactivaPeru struct {
	RazonSocial        string `json:"razon_social"`
	TieneDeudaCoactiva bool   `json:"tiene_deuda_coactiva"` // true si dice "SÍ", false si dice "NO"
	FechaActualizacion string `json:"fecha_actualizacion"`  // "06/08/2025"
	ReferenciaLegal    string `json:"referencia_legal"`     // "Decreto Legislativo N° 1455"
}

// ProgramaCovid19 representa información del programa de garantías COVID-19
type ProgramaCovid19 struct {
	RazonSocial        string `json:"razon_social"`
	ParticipaPrograma  bool   `json:"participa_programa"`
	TieneDeudaCoactiva bool   `json:"tiene_deuda_coactiva"`
	FechaActualizacion string `json:"fecha_actualizacion"` // formato como "06/08/2025"
	BaseLegal          string `json:"base_legal"`          // ejemplo: "Ley N° 31050"
}

// RepresentantesLegales representa los representantes legales
type RepresentantesLegales struct {
	Representantes []RepresentanteLegal `json:"representantes"`
}

type RepresentanteLegal struct {
	TipoDocumento   string `json:"tipo_documento"`
	NumeroDocumento string `json:"numero_documento"`
	NombreCompleto  string `json:"nombre_completo"`
	Cargo           string `json:"cargo"`
	FechaDesde      string `json:"fecha_desde"`
	FechaHasta      string `json:"fecha_hasta,omitempty"`
	Vigente         bool   `json:"vigente"`
}

// EstablecimientosAnexos representa los establecimientos anexos
type EstablecimientosAnexos struct {
	CantidadAnexos   int                    `json:"cantidad_anexos"`
	Establecimientos []EstablecimientoAnexo `json:"establecimientos"`
}

type EstablecimientoAnexo struct {
	Codigo              string `json:"codigo"`
	TipoEstablecimiento string `json:"tipo_establecimiento"`
	Direccion           string `json:"direccion"`
	ActividadEconomica  string `json:"actividad_economica"`
}
