#!/bin/bash

# Variables de conexión
DB_NAME="sunat"
DB_USER="postgres"
DB_HOST="localhost"
DB_PORT="5433"
DB_PASS="admin123"

# Exportar la contraseña para psql
export PGPASSWORD="$DB_PASS"

# Crear la tabla si no existe
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<'EOF'
CREATE TABLE IF NOT EXISTS log_consultas (
    id SERIAL PRIMARY KEY,
    ruc VARCHAR(11) NOT NULL,
    estado VARCHAR(50) NOT NULL,
    mensaje TEXT,
    fecha_registro TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
ALTER TABLE log_consultas ADD COLUMN especificacion TEXT;
EOF

# Insertar datos iniciales desde empresas_sunat
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<'EOF'
INSERT INTO log_consultas (ruc, estado, mensaje, especificacion)
SELECT ruc, 'pendiente', 'Registro inicial', 'Consulta RUC inicial desde empresas_sunat'
FROM empresas_sunat
ON CONFLICT DO NOTHING;
EOF

echo "✅ Datos insertados en log_consultas con estado inicial 'pendiente'."
