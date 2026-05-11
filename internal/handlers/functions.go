package handlers

import (
	"database/sql"
	"fmt"
)

/*
 *
 * Querie de prueba:
 * select * from (
 * 	SELECT row_number() over (order by last_modified desc) rownum, * FROM clients WHERE 1 = 1 AND dueindicator = 'paid'
 * )
 * WHERE rownum >= 0 and rownum <= 20;
 *
 * asegurado, contratante, dia_cobro, estatus, fecha_emision, forma_pago, medio_cobro, numpoliza, plan, tipo_seguro, allownotifications, addr_calle, addr_codigopostal, addr_ciudad, addr_estado, addr_colonia, dueIndicator
 *
 */

func GetPolizasDinamicas(db *sql.DB, filtros map[string]interface{}, user_uuid string) (*sql.Rows, error) {
	// 1. Iniciamos con el argumento del usuario
	args := []interface{}{user_uuid}
	argCount := 2 // Empezamos en 2 porque el $1 es el user_uuid

	// Base de la consulta usando $1 para el UUID
	sqlQuery := `
		SELECT row_number() over (order by p.last_modified desc) rownum,
		p.poliza_uuid, p.asegurado, p.contratante, p.dia_cobro, p.estatus, p.fecha_emision,
		p.forma_pago, p.medio_cobro, p.numpoliza, p.plan, p.tipo_seguro, p.addr_calle,
		p.addr_codigopostal, p.addr_ciudad, p.addr_colonia, p.addr_estado,
		ppc.next_payment, p.user_uuid as owner_id
		FROM polizas p
		JOIN polizas_payments_conf ppc ON ppc.poliza_uuid = p.poliza_uuid
		WHERE p.user_uuid = $1
	`

	// 2. Construcción dinámica de filtros
	for columna, valor := range filtros {
		// Validamos que el valor sea útil
		if valor == "" || valor == nil {
			continue
		}

		if columna == "next_payment" {
			// Opción A: Filtro estático de 5 días
			// No usamos placeholder porque el intervalo está "hardcodeado"
			sqlQuery += ` AND (ppc.next_payment - CURRENT_DATE) <= INTERVAL '5 days' `

			// IMPORTANTE: No incrementamos argCount ni añadimos a args
			// porque no pusimos ningún "$n" en este bloque.
		} else {
			// Filtros dinámicos normales
			sqlQuery += fmt.Sprintf(` AND p.%s = $%d`, columna, argCount)
			args = append(args, valor)
			argCount++
		}
	}

	// 3. Query final (simplificada)
	// No necesitas envolverla en otro SELECT a menos que vayas a filtrar el rownum
	finalQuery := fmt.Sprintf(`
		SELECT poliza_uuid, asegurado, contratante,
		dia_cobro, estatus, fecha_emision,
		forma_pago, medio_cobro, numpoliza, plan, tipo_seguro, addr_calle, addr_codigopostal,
		addr_ciudad, addr_colonia, addr_estado, next_payment, owner_id
		FROM ( %s ) sub;`, sqlQuery)

	fmt.Println("Ejecutando Query con args:", args)
	fmt.Println("Final query:", finalQuery)

	return db.Query(finalQuery, args...)
}
