# SUNAT RUC Scraper - Completo

Web scraper avanzado para consultar información completa de RUC en el portal de SUNAT, incluyendo todas las consultas adicionales disponibles. Implementado en Go con Rod.

## Opción 1: Usar versión Python (Recomendado si no tienes Go)

### Instalación
```bash
pip3 install -r requirements.txt
```

### Uso
```bash
# Con RUC por defecto
./run-python.sh

# Con RUCs específicos
./run-python.sh 20606316977 20100000001
```

## Opción 2: Usar versión Go

### Instalación
```bash
go mod download
```

### Uso
```bash
# Con RUC por defecto
./run.sh

# Con RUCs específicos
./run.sh 20606316977 20100000001
```

## Opción 3: Usar Docker (No requiere Go ni Python)

```bash
# Con RUC por defecto
./run-docker.sh

# Con RUCs específicos
./run-docker.sh 20606316977 20100000001
```

## Funcionalidades

### Consulta Básica
- Información general del RUC
- Actividades económicas
- Comprobantes autorizados
- Estado y condición

### Consultas Adicionales (Nuevo)
1. **Información Histórica** - Cambios en razón social, domicilio, estado
2. **Deuda Coactiva** - Deudas pendientes en cobranza
3. **Omisiones Tributarias** - Declaraciones pendientes
4. **Cantidad de Trabajadores** - Histórico por periodos
5. **Actas Probatorias** - Infracciones registradas
6. **Facturas Físicas** - Autorizaciones de impresión
7. **Reactiva Perú** - Participación en el programa
8. **Programa COVID-19** - Garantías estatales
9. **Representantes Legales** - Directivos y apoderados
10. **Establecimientos Anexos** - Locales adicionales

## Ejecutar Consulta Completa

```bash
# Consulta completa con todas las opciones
./run-completo.sh

# O directamente con un RUC
./run-completo.sh 20606316977
```

## Estructura del Proyecto

```
.
├── cmd/
│   ├── scraper/            # Scraper básico
│   └── scraper-completo/   # Scraper con todas las consultas
├── pkg/
│   ├── models/
│   │   ├── ruc.go          # Modelo básico
│   │   ├── consultas_adicionales.go  # Modelos extendidos
│   │   └── ruc_completo.go # Modelo completo
│   ├── scraper/
│   │   ├── scraper.go      # Scraper básico
│   │   ├── scraper_extendido.go # Implementación completa
│   │   └── scraper_mejorado.go  # Con reintentos y mejoras
│   └── utils/
│       └── parser.go       # Utilidades de parseo
├── database/
│   └── schema.sql          # Esquema PostgreSQL
├── ejemplos/
│   └── ruc_completo_ejemplo.json # Ejemplo de salida
├── go.mod                  # Dependencias
└── README.md               # Este archivo
```

## Salida de Datos

Los datos se guardan en formato JSON con la estructura completa de información. Ver `ejemplos/ruc_completo_ejemplo.json` para un ejemplo detallado.

## Base de Datos

El proyecto incluye un esquema completo de PostgreSQL para almacenar toda la información de manera estructurada. Ver `database/schema.sql`.

## Nota

- El scraper puede tomar varios minutos para obtener toda la información
- Algunas consultas pueden no estar disponibles para todos los RUCs
- Se recomienda usar con moderación para no sobrecargar los servidores de SUNAT