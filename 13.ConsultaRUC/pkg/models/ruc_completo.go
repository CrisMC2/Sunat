package models

import "time"

// RUCCompleto representa toda la información disponible de un RUC
type RUCCompleto struct {
	// Información básica
	InformacionBasica RUCInfo `json:"informacion_basica"`

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
	FechaConsulta       time.Time       `json:"fecha_consulta"`
	VersionAPI          string          `json:"version_api"`
	DeteccionPaginacion map[string]bool `json:"deteccion_paginacion,omitempty"`
}
