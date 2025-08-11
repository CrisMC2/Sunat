package models

import "time"

// RUCCompleto representa toda la información disponible de un RUC
type RUCCompleto struct {
	// Información básica
	InformacionBasica      RUCInfo                 `json:"informacion_basica"`
	
	// Consultas adicionales
	InformacionHistorica   *InformacionHistorica   `json:"informacion_historica,omitempty"`
	DeudaCoactiva          *DeudaCoactiva          `json:"deuda_coactiva,omitempty"`
	OmisionesTributarias   *OmisionesTributarias   `json:"omisiones_tributarias,omitempty"`
	CantidadTrabajadores   *CantidadTrabajadores   `json:"cantidad_trabajadores,omitempty"`
	ActasProbatorias       *ActasProbatorias       `json:"actas_probatorias,omitempty"`
	FacturasFisicas        *FacturasFisicas        `json:"facturas_fisicas,omitempty"`
	ReactivaPeru           *ReactivaPeru           `json:"reactiva_peru,omitempty"`
	ProgramaCovid19        *ProgramaCovid19        `json:"programa_covid19,omitempty"`
	RepresentantesLegales  *RepresentantesLegales  `json:"representantes_legales,omitempty"`
	EstablecimientosAnexos *EstablecimientosAnexos `json:"establecimientos_anexos,omitempty"`
	
	// Metadata
	FechaConsulta          time.Time               `json:"fecha_consulta"`
	VersionAPI             string                  `json:"version_api"`
}

// ResumenConsulta representa un resumen de todas las consultas realizadas
type ResumenConsulta struct {
	RUC                        string    `json:"ruc"`
	RazonSocial                string    `json:"razon_social"`
	Estado                     string    `json:"estado"`
	Condicion                  string    `json:"condicion"`
	
	// Resumen de consultas adicionales
	TieneDeudaCoactiva         bool      `json:"tiene_deuda_coactiva"`
	MontoDeudaCoactiva         float64   `json:"monto_deuda_coactiva,omitempty"`
	TieneOmisiones             bool      `json:"tiene_omisiones"`
	CantidadOmisiones          int       `json:"cantidad_omisiones,omitempty"`
	TieneActasProbatorias      bool      `json:"tiene_actas_probatorias"`
	CantidadTrabajadoresActual int       `json:"cantidad_trabajadores_actual,omitempty"`
	CantidadAnexos             int       `json:"cantidad_anexos"`
	ParticipaReactivaPeru      bool      `json:"participa_reactiva_peru"`
	ParticipaCovid19           bool      `json:"participa_covid19"`
	TieneRepresentantesVigentes bool     `json:"tiene_representantes_vigentes"`
	
	FechaConsulta              time.Time `json:"fecha_consulta"`
}