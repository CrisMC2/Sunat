-- Esquema de base de datos para información completa de RUC SUNAT

-- Tabla principal de RUC
CREATE TABLE IF NOT EXISTS ruc_info (
    id SERIAL PRIMARY KEY,
    ruc VARCHAR(11) UNIQUE NOT NULL,
    razon_social TEXT NOT NULL,
    tipo_contribuyente VARCHAR(100),
    nombre_comercial TEXT,
    fecha_inscripcion DATE,
    fecha_inicio_actividades DATE,
    estado VARCHAR(50),
    condicion VARCHAR(50),
    domicilio_fiscal TEXT,
    sistema_emision VARCHAR(100),
    actividad_comercio_exterior VARCHAR(100),
    sistema_contabilidad VARCHAR(100),
    sistema_emision_electronica TEXT,
    emisor_electronico_desde DATE,
    afiliado_ple VARCHAR(50),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para actividades económicas
CREATE TABLE IF NOT EXISTS actividades_economicas (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    tipo VARCHAR(50), -- Principal, Secundaria 1, etc.
    codigo VARCHAR(10),
    descripcion TEXT,
    fecha_alta DATE,
    fecha_baja DATE,
    vigente BOOLEAN DEFAULT TRUE
);

-- Tabla para comprobantes de pago
CREATE TABLE IF NOT EXISTS comprobantes_pago (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    tipo_comprobante VARCHAR(100),
    autorizacion VARCHAR(50),
    fecha_autorizacion DATE
);

-- Tabla para comprobantes electrónicos
CREATE TABLE IF NOT EXISTS comprobantes_electronicos (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    tipo_comprobante VARCHAR(100),
    sistema VARCHAR(50),
    fecha_desde DATE
);

-- Tabla para padrones
CREATE TABLE IF NOT EXISTS padrones (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    tipo_padron VARCHAR(100),
    fecha_incorporacion DATE,
    fecha_exclusion DATE
);

-- Tabla de información histórica
CREATE TABLE IF NOT EXISTS informacion_historica (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    tipo_cambio VARCHAR(50), -- RAZON_SOCIAL, DOMICILIO, ESTADO, CONDICION
    fecha_cambio DATE,
    valor_anterior TEXT,
    valor_nuevo TEXT,
    motivo TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de deuda coactiva
CREATE TABLE IF NOT EXISTS deuda_coactiva (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    numero_documento VARCHAR(50),
    tipo_documento VARCHAR(50),
    fecha_emision DATE,
    periodo VARCHAR(20),
    tributo VARCHAR(100),
    monto_original DECIMAL(15,2),
    monto_actualizado DECIMAL(15,2),
    estado VARCHAR(50),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de omisiones tributarias
CREATE TABLE IF NOT EXISTS omisiones_tributarias (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    periodo VARCHAR(20),
    tributo VARCHAR(100),
    tipo_declaracion VARCHAR(100),
    fecha_vencimiento DATE,
    estado VARCHAR(50),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de cantidad de trabajadores
CREATE TABLE IF NOT EXISTS cantidad_trabajadores (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    periodo VARCHAR(20),
    cantidad_trabajadores INTEGER DEFAULT 0,
    cantidad_prestadores_servicio INTEGER DEFAULT 0,
    cantidad_practicantes INTEGER DEFAULT 0,
    total INTEGER DEFAULT 0,
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de actas probatorias
CREATE TABLE IF NOT EXISTS actas_probatorias (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    numero_acta VARCHAR(50),
    fecha_acta DATE,
    tipo_infraccion VARCHAR(100),
    descripcion_hechos TEXT,
    monto_multa DECIMAL(15,2),
    estado VARCHAR(50),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de facturas físicas
CREATE TABLE IF NOT EXISTS facturas_fisicas (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    numero_autorizacion VARCHAR(50),
    fecha_autorizacion DATE,
    tipo_comprobante VARCHAR(50),
    serie VARCHAR(10),
    numero_inicial VARCHAR(20),
    numero_final VARCHAR(20),
    fecha_vencimiento DATE,
    estado VARCHAR(50),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Reactiva Perú
CREATE TABLE IF NOT EXISTS reactiva_peru (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    entidad_financiera VARCHAR(200),
    monto_credito DECIMAL(15,2),
    fecha_desembolso DATE,
    estado VARCHAR(50),
    saldo_pendiente DECIMAL(15,2),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Programa COVID-19
CREATE TABLE IF NOT EXISTS programa_covid19 (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    entidad_financiera VARCHAR(200),
    tipo_garantia VARCHAR(100),
    monto_garantia DECIMAL(15,2),
    fecha_otorgamiento DATE,
    estado VARCHAR(50),
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de representantes legales
CREATE TABLE IF NOT EXISTS representantes_legales (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    tipo_documento VARCHAR(20),
    numero_documento VARCHAR(20),
    apellido_paterno VARCHAR(100),
    apellido_materno VARCHAR(100),
    nombres VARCHAR(200),
    cargo VARCHAR(100),
    fecha_desde DATE,
    fecha_hasta DATE,
    vigente BOOLEAN DEFAULT TRUE,
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de establecimientos anexos
CREATE TABLE IF NOT EXISTS establecimientos_anexos (
    id SERIAL PRIMARY KEY,
    ruc_id INTEGER REFERENCES ruc_info(id) ON DELETE CASCADE,
    codigo_establecimiento VARCHAR(20),
    tipo_establecimiento VARCHAR(50),
    denominacion_comercial TEXT,
    direccion TEXT,
    departamento VARCHAR(100),
    provincia VARCHAR(100),
    distrito VARCHAR(100),
    estado VARCHAR(50),
    fecha_alta DATE,
    fecha_baja DATE,
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de actividades por establecimiento
CREATE TABLE IF NOT EXISTS actividades_establecimiento (
    id SERIAL PRIMARY KEY,
    establecimiento_id INTEGER REFERENCES establecimientos_anexos(id) ON DELETE CASCADE,
    codigo_actividad VARCHAR(10),
    descripcion_actividad TEXT
);

-- Tabla de resumen de consultas
CREATE TABLE IF NOT EXISTS resumen_consultas (
    id SERIAL PRIMARY KEY,
    ruc VARCHAR(11) NOT NULL,
    razon_social TEXT,
    estado VARCHAR(50),
    condicion VARCHAR(50),
    tiene_deuda_coactiva BOOLEAN DEFAULT FALSE,
    monto_deuda_coactiva DECIMAL(15,2),
    tiene_omisiones BOOLEAN DEFAULT FALSE,
    cantidad_omisiones INTEGER DEFAULT 0,
    tiene_actas_probatorias BOOLEAN DEFAULT FALSE,
    cantidad_trabajadores_actual INTEGER,
    cantidad_anexos INTEGER DEFAULT 0,
    participa_reactiva_peru BOOLEAN DEFAULT FALSE,
    participa_covid19 BOOLEAN DEFAULT FALSE,
    tiene_representantes_vigentes BOOLEAN DEFAULT FALSE,
    fecha_consulta TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices para mejorar el rendimiento
CREATE INDEX idx_ruc_info_ruc ON ruc_info(ruc);
CREATE INDEX idx_ruc_info_estado ON ruc_info(estado);
CREATE INDEX idx_ruc_info_condicion ON ruc_info(condicion);
CREATE INDEX idx_actividades_ruc_id ON actividades_economicas(ruc_id);
CREATE INDEX idx_deuda_coactiva_ruc_id ON deuda_coactiva(ruc_id);
CREATE INDEX idx_representantes_ruc_id ON representantes_legales(ruc_id);
CREATE INDEX idx_establecimientos_ruc_id ON establecimientos_anexos(ruc_id);

-- Vista para consulta rápida de información completa
CREATE OR REPLACE VIEW v_ruc_completo AS
SELECT 
    r.ruc,
    r.razon_social,
    r.tipo_contribuyente,
    r.estado,
    r.condicion,
    r.domicilio_fiscal,
    COUNT(DISTINCT dc.id) as cantidad_deudas,
    SUM(dc.monto_actualizado) as total_deuda_coactiva,
    COUNT(DISTINCT ot.id) as cantidad_omisiones,
    COUNT(DISTINCT ap.id) as cantidad_actas,
    COUNT(DISTINCT ea.id) as cantidad_anexos,
    COUNT(DISTINCT rl.id) FILTER (WHERE rl.vigente = TRUE) as representantes_vigentes,
    r.fecha_consulta
FROM ruc_info r
LEFT JOIN deuda_coactiva dc ON r.id = dc.ruc_id
LEFT JOIN omisiones_tributarias ot ON r.id = ot.ruc_id
LEFT JOIN actas_probatorias ap ON r.id = ap.ruc_id
LEFT JOIN establecimientos_anexos ea ON r.id = ea.ruc_id
LEFT JOIN representantes_legales rl ON r.id = rl.ruc_id
GROUP BY r.id, r.ruc, r.razon_social, r.tipo_contribuyente, r.estado, 
         r.condicion, r.domicilio_fiscal, r.fecha_consulta;