#!/bin/bash

# Activar modo estricto para detectar errores
set -e

echo "=== Iniciando ejecución de scripts de configuracion de BD==="


# Dar permisos de ejecución a todos los scripts de la carpeta
chmod +x scripts/*.sh

# Llamar a cada script
./scripts/eliminar_tablas.sh
./scripts/crear_tablas.sh
./scripts/run-schema.sh
./scripts/datos-iniciales.sh

echo "=== Todos los scripts se ejecutaron correctamente ==="