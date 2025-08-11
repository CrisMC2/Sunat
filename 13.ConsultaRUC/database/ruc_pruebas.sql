CREATE TABLE empresas_sunat (
    ruc BIGINT PRIMARY KEY,
    estado VARCHAR(20),
    condicion VARCHAR(20),
    tipo VARCHAR(50),
    actividad_economica_ciiu_rev3_principal VARCHAR(255),
    actividad_economica_ciiu_rev3_secundaria VARCHAR(255),
    actividad_economica_ciiu_rev4_principal VARCHAR(255),
    nro_trabajadores VARCHAR(20),
    tipo_facturacion VARCHAR(20),
    tipo_contabilidad VARCHAR(20),
    comercio_exterior VARCHAR(50),
    ubigeo CHAR(6),
    departamento VARCHAR(50),
    provincia VARCHAR(50),
    distrito VARCHAR(50),
    periodo_publicacion CHAR(6)
);
