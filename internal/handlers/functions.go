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

func GetPolizasDinamicas(db *sql.DB, filtros map[string]interface{}) (*sql.Rows, error) {
	// Base de la consulta
	sqlQuery := `
		SELECT row_number() over (order by last_modified desc) rownum, p.poliza_uuid, p.asegurado, p.contratante,
		p.dia_cobro, p.estatus, p.fecha_emision,
		p.forma_pago, p.medio_cobro, p.numpoliza, p.plan, p.tipo_seguro, p.addr_calle, addr_codigopostal,
		p.addr_ciudad, p.addr_colonia, p.addr_estado, ppc.next_payment, ua.user_uuid,
		COALESCE(
            (SELECT 1 FROM polizas_payments_log ppl
             WHERE ppl.poliza_id = p.poliza_uuid LIMIT 1),
            0
        ) AS hasLog
		FROM polizas p
		JOIN polizas_payments_conf ppc ON ppc.poliza_uuid = p.poliza_uuid
		JOIN users_aseguradores ua ON ua.user_id = p.user_id
		WHERE 1 = 1
	`
	var args []interface{}
	argCount := 1

	for columna, valor := range filtros {
		if valor != "" && valor != nil {
			sqlQuery += fmt.Sprintf(" AND %s = $%d", columna, argCount)
			args = append(args, valor)
			argCount++
		}
	}

	finalQuery := fmt.Sprintf(`
		SELECT poliza_uuid, asegurado, contratante,
		dia_cobro, estatus, fecha_emision,
		forma_pago, medio_cobro, numpoliza, plan, tipo_seguro, addr_calle, addr_codigopostal,
		addr_ciudad, addr_colonia, addr_estado, next_payment, user_uuid, hasLog
		FROM ( %s ) WHERE rownum >= 0 and rownum <= 20;`, sqlQuery)

	fmt.Println(finalQuery)

	rows, err := db.Query(finalQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error en query: %v", err)
	}

	return rows, nil
}
