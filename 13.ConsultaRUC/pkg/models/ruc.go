package models

// Estructura actualizada RUCInfo con el nuevo campo TipoDocumento
type RUCInfo struct {
	RUC                       string   `json:"ruc"`
	RazonSocial               string   `json:"razon_social"`
	TipoContribuyente         string   `json:"tipo_contribuyente"`
	TipoDocumento             string   `json:"tipo_documento"` // NUEVO CAMPO
	NombreComercial           string   `json:"nombre_comercial"`
	FechaInscripcion          string   `json:"fecha_inscripcion"`
	FechaInicioActividades    string   `json:"fecha_inicio_actividades"`
	Estado                    string   `json:"estado"`
	Condicion                 string   `json:"condicion"`
	DomicilioFiscal           string   `json:"domicilio_fiscal"`
	SistemaEmision            string   `json:"sistema_emision"`
	ActividadComercioExterior string   `json:"actividad_comercio_exterior"`
	SistemaContabilidad       string   `json:"sistema_contabilidad"`
	ActividadesEconomicas     []string `json:"actividades_economicas"`
	ComprobantesPago          []string `json:"comprobantes_pago"`
	SistemaEmisionElectronica []string `json:"sistema_emision_electronica"`
	EmisorElectronicoDesde    string   `json:"emisor_electronico_desde"`
	ComprobantesElectronicos  []string `json:"comprobantes_electronicos"`
	AfiliadoPLE               string   `json:"afiliado_ple"`
	Padrones                  []string `json:"padrones"`
}
