CREATE TABLE empresas_sunat (
    ruc BIGINT PRIMARY KEY,
    estado TEXT,
    condicion TEXT,
    tipo TEXT,
    actividad_economica_ciiu_rev3_principal TEXT,
    actividad_economica_ciiu_rev3_secundaria TEXT,
    actividad_economica_ciiu_rev4_principal TEXT,
    nro_trabajadores TEXT,
    tipo_facturacion TEXT,
    tipo_contabilidad TEXT,
    comercio_exterior TEXT,
    ubigeo TEXT,
    departamento TEXT,
    provincia TEXT,
    distrito TEXT,
    periodo_publicacion CHAR(6)
);

-- Crear la tabla ruc_pruebas
CREATE TABLE IF NOT EXISTS ruc_pruebas (
    id SERIAL PRIMARY KEY,
    ruc VARCHAR(11) NOT NULL UNIQUE  -- UNIQUE constraint agregada
);

CREATE TABLE IF NOT EXISTS log_consultas (
    id SERIAL PRIMARY KEY,
    ruc VARCHAR(11) NOT NULL,
    estado VARCHAR(50) NOT NULL,
    mensaje TEXT,
    fecha_registro TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);