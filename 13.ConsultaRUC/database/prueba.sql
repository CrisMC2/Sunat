-- ====================================
-- TABLAS PARA SISTEMA RUC - PostgreSQL
-- ====================================

-- Tabla principal con información básica del RUC
CREATE TABLE ruc_informacion_basica (
    id BIGSERIAL PRIMARY KEY,
    ruc VARCHAR(11) NOT NULL UNIQUE,
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
    emisor_electronico_desde DATE,
    afiliado_ple VARCHAR(10),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para actividades económicas (relación muchos a muchos)
CREATE TABLE ruc_actividades_economicas (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    actividad_economica TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para comprobantes de pago (relación muchos a muchos)
CREATE TABLE ruc_comprobantes_pago (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    comprobante_pago VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para sistemas de emisión electrónica (relación muchos a muchos)
CREATE TABLE ruc_sistemas_emision_electronica (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    sistema_emision VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para comprobantes electrónicos (relación muchos a muchos)
CREATE TABLE ruc_comprobantes_electronicos (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    comprobante_electronico VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla para padrones (relación muchos a muchos)
CREATE TABLE ruc_padrones (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    padron VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabla principal para RUC completo (metadata)
CREATE TABLE ruc_consultas (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    fecha_consulta TIMESTAMP NOT NULL,
    version_api VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- INFORMACIÓN HISTÓRICA
-- ====================================

CREATE TABLE ruc_informacion_historica (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_razones_sociales_historicas (
    id BIGSERIAL PRIMARY KEY,
    informacion_historica_id BIGINT NOT NULL REFERENCES ruc_informacion_historica(id) ON DELETE CASCADE,
    nombre TEXT,
    fecha_de_baja DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_condiciones_historicas (
    id BIGSERIAL PRIMARY KEY,
    informacion_historica_id BIGINT NOT NULL REFERENCES ruc_informacion_historica(id) ON DELETE CASCADE,
    condicion VARCHAR(50),
    desde DATE,
    hasta DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_domicilios_fiscales_historicos (
    id BIGSERIAL PRIMARY KEY,
    informacion_historica_id BIGINT NOT NULL REFERENCES ruc_informacion_historica(id) ON DELETE CASCADE,
    direccion TEXT,
    fecha_de_baja DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- DEUDA COACTIVA
-- ====================================

CREATE TABLE ruc_deuda_coactiva (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    total_deuda DECIMAL(15,2),
    cantidad_documentos INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_detalle_deudas (
    id BIGSERIAL PRIMARY KEY,
    deuda_coactiva_id BIGINT NOT NULL REFERENCES ruc_deuda_coactiva(id) ON DELETE CASCADE,
    monto DECIMAL(15,2),
    periodo_tributario VARCHAR(20),
    fecha_inicio_cobranza DATE,
    entidad VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- OMISIONES TRIBUTARIAS
-- ====================================

CREATE TABLE ruc_omisiones_tributarias (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    tiene_omisiones BOOLEAN,
    cantidad_omisiones INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_omisiones (
    id BIGSERIAL PRIMARY KEY,
    omisiones_tributarias_id BIGINT NOT NULL REFERENCES ruc_omisiones_tributarias(id) ON DELETE CASCADE,
    periodo VARCHAR(20),
    tributo VARCHAR(100),
    tipo_declaracion VARCHAR(100),
    fecha_vencimiento DATE,
    estado VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- CANTIDAD DE TRABAJADORES
-- ====================================

CREATE TABLE ruc_cantidad_trabajadores (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_periodos_disponibles_trabajadores (
    id BIGSERIAL PRIMARY KEY,
    cantidad_trabajadores_id BIGINT NOT NULL REFERENCES ruc_cantidad_trabajadores(id) ON DELETE CASCADE,
    periodo VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_detalle_trabajadores (
    id BIGSERIAL PRIMARY KEY,
    cantidad_trabajadores_id BIGINT NOT NULL REFERENCES ruc_cantidad_trabajadores(id) ON DELETE CASCADE,
    periodo VARCHAR(20),
    cantidad_trabajadores INTEGER,
    cantidad_prestadores_servicio INTEGER,
    cantidad_pensionistas INTEGER,
    total INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- ACTAS PROBATORIAS
-- ====================================

CREATE TABLE ruc_actas_probatorias (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    tiene_actas BOOLEAN,
    cantidad_actas INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_actas (
    id BIGSERIAL PRIMARY KEY,
    actas_probatorias_id BIGINT NOT NULL REFERENCES ruc_actas_probatorias(id) ON DELETE CASCADE,
    numero_acta VARCHAR(50),
    fecha_acta DATE,
    lugar_intervencion TEXT,
    articulo_numeral VARCHAR(100),
    descripcion_infraccion TEXT,
    numero_ri_roz VARCHAR(50),
    tipo_ri_roz VARCHAR(50),
    acta_reconocimiento VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- FACTURAS FÍSICAS
-- ====================================

CREATE TABLE ruc_facturas_fisicas (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    tiene_autorizacion BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_facturas_autorizadas (
    id BIGSERIAL PRIMARY KEY,
    facturas_fisicas_id BIGINT NOT NULL REFERENCES ruc_facturas_fisicas(id) ON DELETE CASCADE,
    numero_autorizacion VARCHAR(50),
    fecha_autorizacion DATE,
    tipo_comprobante VARCHAR(100),
    serie VARCHAR(20),
    numero_inicial VARCHAR(20),
    numero_final VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_facturas_canceladas_bajas (
    id BIGSERIAL PRIMARY KEY,
    facturas_fisicas_id BIGINT NOT NULL REFERENCES ruc_facturas_fisicas(id) ON DELETE CASCADE,
    numero_autorizacion VARCHAR(50),
    fecha_autorizacion DATE,
    tipo_comprobante VARCHAR(100),
    serie VARCHAR(20),
    numero_inicial VARCHAR(20),
    numero_final VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- REACTIVA PERÚ
-- ====================================

CREATE TABLE ruc_reactiva_peru (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    razon_social TEXT,
    tiene_deuda_coactiva BOOLEAN,
    fecha_actualizacion DATE,
    referencia_legal TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- PROGRAMA COVID-19
-- ====================================

CREATE TABLE ruc_programa_covid19 (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    razon_social TEXT,
    participa_programa BOOLEAN,
    tiene_deuda_coactiva BOOLEAN,
    fecha_actualizacion DATE,
    base_legal TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- REPRESENTANTES LEGALES
-- ====================================

CREATE TABLE ruc_representantes_legales (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_representantes (
    id BIGSERIAL PRIMARY KEY,
    representantes_legales_id BIGINT NOT NULL REFERENCES ruc_representantes_legales(id) ON DELETE CASCADE,
    tipo_documento VARCHAR(20),
    numero_documento VARCHAR(20),
    nombre_completo TEXT,
    cargo VARCHAR(100),
    fecha_desde DATE,
    fecha_hasta DATE,
    vigente BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- ESTABLECIMIENTOS ANEXOS
-- ====================================

CREATE TABLE ruc_establecimientos_anexos (
    id BIGSERIAL PRIMARY KEY,
    ruc_id BIGINT NOT NULL REFERENCES ruc_informacion_basica(id) ON DELETE CASCADE,
    cantidad_anexos INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE ruc_establecimientos (
    id BIGSERIAL PRIMARY KEY,
    establecimientos_anexos_id BIGINT NOT NULL REFERENCES ruc_establecimientos_anexos(id) ON DELETE CASCADE,
    codigo VARCHAR(20),
    tipo_establecimiento VARCHAR(100),
    direccion TEXT,
    actividad_economica TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ====================================
-- ÍNDICES PARA OPTIMIZACIÓN
-- ====================================

-- Índices principales
CREATE INDEX idx_ruc_informacion_basica_ruc ON ruc_informacion_basica(ruc);
CREATE INDEX idx_ruc_informacion_basica_estado ON ruc_informacion_basica(estado);
CREATE INDEX idx_ruc_informacion_basica_condicion ON ruc_informacion_basica(condicion);
CREATE INDEX idx_ruc_informacion_basica_razon_social ON ruc_informacion_basica USING gin(to_tsvector('spanish', razon_social));

-- Índices de relaciones
CREATE INDEX idx_ruc_actividades_economicas_ruc_id ON ruc_actividades_economicas(ruc_id);
CREATE INDEX idx_ruc_comprobantes_pago_ruc_id ON ruc_comprobantes_pago(ruc_id);
CREATE INDEX idx_ruc_sistemas_emision_electronica_ruc_id ON ruc_sistemas_emision_electronica(ruc_id);
CREATE INDEX idx_ruc_comprobantes_electronicos_ruc_id ON ruc_comprobantes_electronicos(ruc_id);
CREATE INDEX idx_ruc_padrones_ruc_id ON ruc_padrones(ruc_id);

-- Índices de fechas
CREATE INDEX idx_ruc_consultas_fecha_consulta ON ruc_consultas(fecha_consulta);
CREATE INDEX idx_ruc_informacion_historica_fecha_actualizada ON ruc_informacion_historica(fecha_actualizada);

-- Índices de deuda
CREATE INDEX idx_ruc_deuda_coactiva_ruc_id ON ruc_deuda_coactiva(ruc_id);
CREATE INDEX idx_ruc_deuda_coactiva_total_deuda ON ruc_deuda_coactiva(total_deuda);

-- Índices de trabajadores
CREATE INDEX idx_ruc_detalle_trabajadores_periodo ON ruc_detalle_trabajadores(periodo);
CREATE INDEX idx_ruc_detalle_trabajadores_total ON ruc_detalle_trabajadores(total);

-- Índices de representantes
CREATE INDEX idx_ruc_representantes_numero_documento ON ruc_representantes(numero_documento);
CREATE INDEX idx_ruc_representantes_vigente ON ruc_representantes(vigente);

-- ====================================
-- FUNCIONES DE UTILIDAD
-- ====================================

-- Función para actualizar timestamps automáticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Aplicar trigger a la tabla principal
CREATE TRIGGER update_ruc_informacion_basica_updated_at 
    BEFORE UPDATE ON ruc_informacion_basica 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ====================================
-- COMENTARIOS EN TABLAS
-- ====================================

COMMENT ON TABLE ruc_informacion_basica IS 'Información básica del RUC';
COMMENT ON TABLE ruc_consultas IS 'Registro de consultas realizadas al RUC';
COMMENT ON TABLE ruc_informacion_historica IS 'Información histórica de cambios del RUC';
COMMENT ON TABLE ruc_deuda_coactiva IS 'Información de deuda en cobranza coactiva';
COMMENT ON TABLE ruc_omisiones_tributarias IS 'Información de omisiones tributarias';
COMMENT ON TABLE ruc_cantidad_trabajadores IS 'Información de cantidad de trabajadores por período';
COMMENT ON TABLE ruc_actas_probatorias IS 'Información de actas probatorias del contribuyente';
COMMENT ON TABLE ruc_facturas_fisicas IS 'Información de facturas físicas autorizadas';
COMMENT ON TABLE ruc_reactiva_peru IS 'Información del programa Reactiva Perú';
COMMENT ON TABLE ruc_programa_covid19 IS 'Información del programa de garantías COVID-19';
COMMENT ON TABLE ruc_representantes_legales IS 'Información de representantes legales';
COMMENT ON TABLE ruc_establecimientos_anexos IS 'Información de establecimientos anexos';