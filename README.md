-- ====================================
-- CONSULTA COMPLETA DE RUC CON DETALLES ESPECÍFICOS
-- Muestra 50 registros aleatorios con datos detallados de tablas anidadas
-- ====================================

SELECT 
    -- INFORMACIÓN BÁSICA
    rib.ruc AS "RUC",
    rib.razon_social AS "Razón Social",
    rib.tipo_contribuyente AS "Tipo Contribuyente",
    rib.tipo_documento AS "Tipo Documento",
    rib.nombre_comercial AS "Nombre Comercial",
    rib.fecha_inscripcion AS "Fecha Inscripción",
    rib.fecha_inicio_actividades AS "Fecha Inicio Actividades",
    rib.estado AS "Estado",
    rib.condicion AS "Condición",
    rib.domicilio_fiscal AS "Domicilio Fiscal",
    rib.sistema_emision AS "Sistema Emisión",
    rib.actividad_comercio_exterior AS "Actividad Comercio Exterior",
    rib.sistema_contabilidad AS "Sistema Contabilidad",
    rib.emisor_electronico_desde AS "Emisor Electrónico Desde",
    rib.afiliado_ple AS "Afiliado PLE",
    
    -- ACTIVIDADES ECONÓMICAS (cada una separada)
    STRING_AGG(DISTINCT '- ' || rae.actividad_economica, E'\n') AS "Actividades Económicas Detalle",
    
    -- COMPROBANTES DE PAGO (cada uno separado)
    STRING_AGG(DISTINCT '- ' || rcp.comprobante_pago, E'\n') AS "Comprobantes de Pago Detalle",
    
    -- SISTEMAS DE EMISIÓN ELECTRÓNICA (cada uno separado)
    STRING_AGG(DISTINCT '- ' || rsee.sistema_emision, E'\n') AS "Sistemas Emisión Electrónica Detalle",
    
    -- COMPROBANTES ELECTRÓNICOS (cada uno separado)
    STRING_AGG(DISTINCT '- ' || rce.comprobante_electronico, E'\n') AS "Comprobantes Electrónicos Detalle",
    
    -- PADRONES (cada uno separado)
    STRING_AGG(DISTINCT '- ' || LEFT(rp.padron, 100), E'\n') AS "Padrones Detalle",
    
    -- DEUDA COACTIVA RESUMEN
    rdc.total_deuda AS "Total Deuda Coactiva",
    rdc.cantidad_documentos AS "Cantidad Documentos Deuda",
    
    -- DETALLES DE DEUDA (cada deuda específica)
    STRING_AGG(DISTINCT 
        CASE WHEN rdd.id IS NOT NULL THEN
            '- Monto: ' || COALESCE(rdd.monto::text, 'N/A') || 
            ' | Período: ' || COALESCE(rdd.periodo_tributario, 'N/A') || 
            ' | Inicio Cobranza: ' || COALESCE(rdd.fecha_inicio_cobranza::text, 'N/A') ||
            ' | Entidad: ' || COALESCE(rdd.entidad, 'N/A')
        END, E'\n'
    ) AS "Detalles Deuda Específica",
    
    -- OMISIONES TRIBUTARIAS RESUMEN
    rot.tiene_omisiones AS "Tiene Omisiones",
    rot.cantidad_omisiones AS "Cantidad Omisiones",
    
    -- OMISIONES ESPECÍFICAS (cada omisión detallada)
    STRING_AGG(DISTINCT 
        CASE WHEN ro.id IS NOT NULL THEN
            '- Período: ' || COALESCE(ro.periodo, 'N/A') ||
            ' | Tributo: ' || COALESCE(ro.tributo, 'N/A') ||
            ' | Tipo Declaración: ' || COALESCE(ro.tipo_declaracion, 'N/A') ||
            ' | Vencimiento: ' || COALESCE(ro.fecha_vencimiento::text, 'N/A') ||
            ' | Estado: ' || COALESCE(ro.estado, 'N/A')
        END, E'\n'
    ) AS "Omisiones Específicas Detalle",
    
    -- TRABAJADORES (todos los períodos disponibles)
    STRING_AGG(DISTINCT 
        CASE WHEN rdt.id IS NOT NULL THEN
            '- Período: ' || COALESCE(rdt.periodo, 'N/A') ||
            ' | Trabajadores: ' || COALESCE(rdt.cantidad_trabajadores::text, '0') ||
            ' | Prestadores: ' || COALESCE(rdt.cantidad_prestadores_servicio::text, '0') ||
            ' | Pensionistas: ' || COALESCE(rdt.cantidad_pensionistas::text, '0') ||
            ' | Total: ' || COALESCE(rdt.total::text, '0')
        END, E'\n'
    ) AS "Trabajadores por Período Detalle",
    
    -- PERÍODOS DISPONIBLES TRABAJADORES
    STRING_AGG(DISTINCT 
        CASE WHEN rpdt.periodo IS NOT NULL THEN
            '- ' || rpdt.periodo
        END, E'\n'
    ) AS "Períodos Disponibles Trabajadores",
    
    -- ACTAS PROBATORIAS RESUMEN
    rap.tiene_actas AS "Tiene Actas",
    rap.cantidad_actas AS "Cantidad Actas",
    
    -- ACTAS ESPECÍFICAS (cada acta detallada)
    STRING_AGG(DISTINCT 
        CASE WHEN ra.id IS NOT NULL THEN
            '- Acta: ' || COALESCE(ra.numero_acta, 'N/A') ||
            ' | Fecha: ' || COALESCE(ra.fecha_acta::text, 'N/A') ||
            ' | Lugar: ' || COALESCE(LEFT(ra.lugar_intervencion, 50), 'N/A') ||
            ' | Artículo: ' || COALESCE(ra.articulo_numeral, 'N/A') ||
            ' | Infracción: ' || COALESCE(LEFT(ra.descripcion_infraccion, 50), 'N/A')
        END, E'\n'
    ) AS "Actas Probatorias Detalle",
    
    -- FACTURAS FÍSICAS RESUMEN
    rff.tiene_autorizacion AS "Tiene Autorización Facturas",
    
    -- FACTURAS AUTORIZADAS (cada autorización detallada)
    STRING_AGG(DISTINCT 
        CASE WHEN rfa.id IS NOT NULL THEN
            '- Autorización: ' || COALESCE(rfa.numero_autorizacion, 'N/A') ||
            ' | Fecha: ' || COALESCE(rfa.fecha_autorizacion::text, 'N/A') ||
            ' | Tipo: ' || COALESCE(rfa.tipo_comprobante, 'N/A') ||
            ' | Serie: ' || COALESCE(rfa.serie, 'N/A') ||
            ' | Rango: ' || COALESCE(rfa.numero_inicial, 'N/A') || '-' || COALESCE(rfa.numero_final, 'N/A')
        END, E'\n'
    ) AS "Facturas Autorizadas Detalle",
    
    -- FACTURAS CANCELADAS/BAJAS (cada cancelación detallada)
    STRING_AGG(DISTINCT 
        CASE WHEN rfcb.id IS NOT NULL THEN
            '- Autorización Cancelada: ' || COALESCE(rfcb.numero_autorizacion, 'N/A') ||
            ' | Fecha: ' || COALESCE(rfcb.fecha_autorizacion::text, 'N/A') ||
            ' | Tipo: ' || COALESCE(rfcb.tipo_comprobante, 'N/A') ||
            ' | Serie: ' || COALESCE(rfcb.serie, 'N/A') ||
            ' | Rango: ' || COALESCE(rfcb.numero_inicial, 'N/A') || '-' || COALESCE(rfcb.numero_final, 'N/A')
        END, E'\n'
    ) AS "Facturas Canceladas/Bajas Detalle",
    
    -- REACTIVA PERÚ
    CASE WHEN rrp.id IS NOT NULL THEN
        '- Razón Social: ' || COALESCE(rrp.razon_social, 'N/A') ||
        ' | Tiene Deuda: ' || COALESCE(rrp.tiene_deuda_coactiva::text, 'N/A') ||
        ' | Actualización: ' || COALESCE(rrp.fecha_actualizacion::text, 'N/A') ||
        ' | Referencia Legal: ' || COALESCE(LEFT(rrp.referencia_legal, 100), 'N/A')
    END AS "Reactiva Perú Detalle",
    
    -- PROGRAMA COVID-19
    CASE WHEN rpc.id IS NOT NULL THEN
        '- Razón Social: ' || COALESCE(rpc.razon_social, 'N/A') ||
        ' | Participa: ' || COALESCE(rpc.participa_programa::text, 'N/A') ||
        ' | Tiene Deuda: ' || COALESCE(rpc.tiene_deuda_coactiva::text, 'N/A') ||
        ' | Actualización: ' || COALESCE(rpc.fecha_actualizacion::text, 'N/A') ||
        ' | Base Legal: ' || COALESCE(LEFT(rpc.base_legal, 100), 'N/A')
    END AS "Programa COVID-19 Detalle",
    
    -- REPRESENTANTES LEGALES (cada representante detallado)
    STRING_AGG(DISTINCT 
        CASE WHEN rr.id IS NOT NULL THEN
            '- ' || COALESCE(rr.tipo_documento, 'N/A') || ': ' || COALESCE(rr.numero_documento, 'N/A') ||
            ' | Nombre: ' || COALESCE(rr.nombre_completo, 'N/A') ||
            ' | Cargo: ' || COALESCE(rr.cargo, 'N/A') ||
            ' | Desde: ' || COALESCE(rr.fecha_desde::text, 'N/A') ||
            ' | Hasta: ' || COALESCE(rr.fecha_hasta::text, 'Vigente') ||
            ' | Vigente: ' || COALESCE(rr.vigente::text, 'N/A')
        END, E'\n'
    ) AS "Representantes Legales Detalle",
    
    -- ESTABLECIMIENTOS ANEXOS RESUMEN
    rea.cantidad_anexos AS "Cantidad Establecimientos Anexos",
    
    -- ESTABLECIMIENTOS ESPECÍFICOS (cada establecimiento detallado)
    STRING_AGG(DISTINCT 
        CASE WHEN re.id IS NOT NULL THEN
            '- Código: ' || COALESCE(re.codigo, 'N/A') ||
            ' | Tipo: ' || COALESCE(re.tipo_establecimiento, 'N/A') ||
            ' | Dirección: ' || COALESCE(LEFT(re.direccion, 100), 'N/A') ||
            ' | Actividad: ' || COALESCE(LEFT(re.actividad_economica, 100), 'N/A')
        END, E'\n'
    ) AS "Establecimientos Anexos Detalle",
    
    -- INFORMACIÓN HISTÓRICA - RAZONES SOCIALES
    STRING_AGG(DISTINCT 
        CASE WHEN rrsh.id IS NOT NULL THEN
            '- Razón Social Histórica: ' || COALESCE(rrsh.nombre, 'N/A') ||
            ' | Fecha Baja: ' || COALESCE(rrsh.fecha_de_baja::text, 'N/A')
        END, E'\n'
    ) AS "Razones Sociales Históricas Detalle",
    
    -- INFORMACIÓN HISTÓRICA - CONDICIONES
    STRING_AGG(DISTINCT 
        CASE WHEN rch.id IS NOT NULL THEN
            '- Condición: ' || COALESCE(rch.condicion, 'N/A') ||
            ' | Desde: ' || COALESCE(rch.desde::text, 'N/A') ||
            ' | Hasta: ' || COALESCE(rch.hasta::text, 'N/A')
        END, E'\n'
    ) AS "Condiciones Históricas Detalle",
    
    -- INFORMACIÓN HISTÓRICA - DOMICILIOS FISCALES
    STRING_AGG(DISTINCT 
        CASE WHEN rdfh.id IS NOT NULL THEN
            '- Dirección: ' || COALESCE(LEFT(rdfh.direccion, 100), 'N/A') ||
            ' | Fecha Baja: ' || COALESCE(rdfh.fecha_de_baja::text, 'N/A')
        END, E'\n'
    ) AS "Domicilios Fiscales Históricos Detalle",
    
    -- FECHAS DE REGISTRO Y ACTUALIZACIÓN
    rib.created_at AS "Fecha Registro Sistema",
    rib.updated_at AS "Última Actualización"

FROM ruc_informacion_basica rib

-- JOINS PRINCIPALES (LEFT JOIN para incluir RUCs aunque no tengan datos en algunas tablas)
LEFT JOIN ruc_actividades_economicas rae ON rib.id = rae.ruc_id
LEFT JOIN ruc_comprobantes_pago rcp ON rib.id = rcp.ruc_id
LEFT JOIN ruc_sistemas_emision_electronica rsee ON rib.id = rsee.ruc_id
LEFT JOIN ruc_comprobantes_electronicos rce ON rib.id = rce.ruc_id
LEFT JOIN ruc_padrones rp ON rib.id = rp.ruc_id

-- DEUDA COACTIVA
LEFT JOIN ruc_deuda_coactiva rdc ON rib.id = rdc.ruc_id
LEFT JOIN ruc_detalle_deudas rdd ON rdc.id = rdd.deuda_coactiva_id

-- OMISIONES TRIBUTARIAS
LEFT JOIN ruc_omisiones_tributarias rot ON rib.id = rot.ruc_id
LEFT JOIN ruc_omisiones ro ON rot.id = ro.omisiones_tributarias_id

-- TRABAJADORES
LEFT JOIN ruc_cantidad_trabajadores rct ON rib.id = rct.ruc_id
LEFT JOIN ruc_periodos_disponibles_trabajadores rpdt ON rct.id = rpdt.cantidad_trabajadores_id
LEFT JOIN ruc_detalle_trabajadores rdt ON rct.id = rdt.cantidad_trabajadores_id

-- ACTAS PROBATORIAS
LEFT JOIN ruc_actas_probatorias rap ON rib.id = rap.ruc_id
LEFT JOIN ruc_actas ra ON rap.id = ra.actas_probatorias_id

-- FACTURAS FÍSICAS
LEFT JOIN ruc_facturas_fisicas rff ON rib.id = rff.ruc_id
LEFT JOIN ruc_facturas_autorizadas rfa ON rff.id = rfa.facturas_fisicas_id
LEFT JOIN ruc_facturas_canceladas_bajas rfcb ON rff.id = rfcb.facturas_fisicas_id

-- PROGRAMAS ESPECIALES
LEFT JOIN ruc_reactiva_peru rrp ON rib.id = rrp.ruc_id
LEFT JOIN ruc_programa_covid19 rpc ON rib.id = rpc.ruc_id

-- REPRESENTANTES LEGALES
LEFT JOIN ruc_representantes_legales rrl ON rib.id = rrl.ruc_id
LEFT JOIN ruc_representantes rr ON rrl.id = rr.representantes_legales_id

-- ESTABLECIMIENTOS ANEXOS
LEFT JOIN ruc_establecimientos_anexos rea ON rib.id = rea.ruc_id
LEFT JOIN ruc_establecimientos re ON rea.id = re.establecimientos_anexos_id

-- INFORMACIÓN HISTÓRICA
LEFT JOIN ruc_informacion_historica rih ON rib.id = rih.ruc_id
LEFT JOIN ruc_razones_sociales_historicas rrsh ON rih.id = rrsh.informacion_historica_id
LEFT JOIN ruc_condiciones_historicas rch ON rih.id = rch.informacion_historica_id
LEFT JOIN ruc_domicilios_fiscales_historicos rdfh ON rih.id = rdfh.informacion_historica_id

-- FILTRO SOLO RUCs QUE EMPIECEN CON 20
WHERE rib.ruc LIKE '20%'

-- AGRUPACIÓN POR TODAS LAS COLUMNAS DE LA TABLA PRINCIPAL
GROUP BY 
    rib.id, rib.ruc, rib.razon_social, rib.tipo_contribuyente, rib.tipo_documento,
    rib.nombre_comercial, rib.fecha_inscripcion, rib.fecha_inicio_actividades,
    rib.estado, rib.condicion, rib.domicilio_fiscal, rib.sistema_emision,
    rib.actividad_comercio_exterior, rib.sistema_contabilidad, rib.emisor_electronico_desde,
    rib.afiliado_ple, rib.created_at, rib.updated_at,
    rdc.total_deuda, rdc.cantidad_documentos,
    rot.tiene_omisiones, rot.cantidad_omisiones,
    rap.tiene_actas, rap.cantidad_actas,
    rff.tiene_autorizacion,
    rrp.id, rrp.razon_social, rrp.tiene_deuda_coactiva, rrp.fecha_actualizacion, rrp.referencia_legal,
    rpc.id, rpc.razon_social, rpc.participa_programa, rpc.tiene_deuda_coactiva, rpc.fecha_actualizacion, rpc.base_legal,
    rea.cantidad_anexos

-- ORDENAMIENTO ALEATORIO Y LÍMITE
ORDER BY RANDOM()
LIMIT 50;
