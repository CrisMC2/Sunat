package utils

import (
	"strconv"
	"strings"
)

// ParseMonto convierte un string de monto a float64
// Ejemplo: "S/ 1,234.56" -> 1234.56
func ParseMonto(montoStr string) float64 {
	// Remover símbolo de moneda
	cleaned := strings.ReplaceAll(montoStr, "S/", "")
	cleaned = strings.ReplaceAll(cleaned, "S/.", "")
	
	// Remover espacios
	cleaned = strings.TrimSpace(cleaned)
	
	// Remover comas
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	
	// Convertir a float
	monto, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0
	}
	
	return monto
}

// ParseCantidad convierte un string a entero
// Ejemplo: "1,234" -> 1234
func ParseCantidad(cantidadStr string) int {
	// Remover comas
	cleaned := strings.ReplaceAll(cantidadStr, ",", "")
	
	// Remover espacios
	cleaned = strings.TrimSpace(cleaned)
	
	// Convertir a int
	cantidad, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0
	}
	
	return cantidad
}

// ParseFecha convierte diferentes formatos de fecha a un formato estándar
// Ejemplo: "01/02/2023" -> "2023-02-01"
func ParseFecha(fechaStr string) string {
	// Formato DD/MM/YYYY
	parts := strings.Split(fechaStr, "/")
	if len(parts) == 3 && len(parts[2]) == 4 {
		return parts[2] + "-" + parts[1] + "-" + parts[0]
	}
	
	return fechaStr
}

// ExtractPeriodo extrae el periodo en formato YYYYMM
// Ejemplo: "ENERO 2023" -> "202301"
func ExtractPeriodo(periodoStr string) string {
	meses := map[string]string{
		"ENERO": "01", "FEBRERO": "02", "MARZO": "03", "ABRIL": "04",
		"MAYO": "05", "JUNIO": "06", "JULIO": "07", "AGOSTO": "08",
		"SEPTIEMBRE": "09", "OCTUBRE": "10", "NOVIEMBRE": "11", "DICIEMBRE": "12",
	}
	
	parts := strings.Fields(strings.ToUpper(periodoStr))
	if len(parts) == 2 {
		mes, ok := meses[parts[0]]
		if ok {
			return parts[1] + mes
		}
	}
	
	return periodoStr
}