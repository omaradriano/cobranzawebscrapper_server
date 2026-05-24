package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

/*
Obtiene la cantidad de las polizas que tiene registrado el usuario en base a su user_uuid
*/
func ApiGetPolizasCountByUser(w http.ResponseWriter, r *http.Request) {
	/*
	 * [] Implementar el uso de JWT para acceso a la ruta
	 */
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetPolizasCountByUser")

	var polizas_count int

	user_uuid := r.URL.Query().Get("uuid")

	err := db.Client.QueryRow(`SELECT COUNT(*) FROM polizas WHERE agente_uuid = $1`, user_uuid).Scan(&polizas_count)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, fmt.Sprintf(`No se ha podido realizar la inserción del usuario: %s`, err.Error()), w)
		return
	}

	services.HandleResponseSuccessWithData(polizas_count, w)
}

/*
*
Carga masiva de los registros
*/
func ApiPostPolizas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	services.NewLogger().OriginAdvice("ApiPostPolizas")

	var CobranzaItemsReceived internal.PostItems_Poliza
	var CobranzaItemsToUpload internal.PostItems_Poliza
	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var agente_id int
	err := db.Client.QueryRow(`SELECT agente_id FROM agentes WHERE agente_uuid = $1`, agente_uuid).Scan(&agente_id)
	if err != nil {
		fmt.Println("Error buscando agente:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error obteniendo información del agente", w)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&CobranzaItemsReceived)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	if len(CobranzaItemsReceived.Payload) == 0 {
		services.HandleResponseError(http.StatusBadRequest, "No se ha recibido información", w)
		return
	}

	// Se obtienen todos los registros del contratante para evitar duplicados
	rows, err := db.Client.Query(`
		SELECT p.numpoliza
		FROM polizas p
		JOIN agentes a ON p.agente_id = a.agente_id
		WHERE a.agente_uuid = $1`, agente_uuid)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Ha ocurrido un error", w)
		return
	}

	existing_rows := make(map[string]bool)
	for rows.Next() {
		var numPoliza string
		if err := rows.Scan(&numPoliza); err != nil {
			fmt.Println("Error escaneando:", err)
			services.HandleResponseError(500, "Error de base de datos", w)
			return
		}
		existing_rows[strings.ToLower(numPoliza)] = true
	}
	rows.Close() // Cerramos explícitamente aquí para liberar la conexión

	for _, receivedItem := range CobranzaItemsReceived.Payload {
		if !existing_rows[strings.ToLower(receivedItem.NumPoliza)] {
			CobranzaItemsToUpload.Payload = append(CobranzaItemsToUpload.Payload, receivedItem)
		}
	}

	if len(CobranzaItemsToUpload.Payload) == 0 {
		services.NewLogger().ErrorMessage("Todos los registros a ingresar ya existen, verifique la petición")
		services.HandleResponseError(http.StatusConflict, "Todos los registros ya se encuentran sincronizados", w)
		return
	}

	// =========================================================================
	// INICIO DE LA TRANSACCIÓN (Para asegurar consistencia entre ambas tablas)
	// =========================================================================
	tx, err := db.Client.Begin()
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Error al iniciar transacción", w)
		return
	}
	defer tx.Rollback() // Red de seguridad: cancela todo si la función sale antes del Commit

	/*
	 * Armado de INSERT dinámico para polizas añadiendo RETURNING
	 */
	const colsPerItem = 19
	var placeholders []string
	var args []interface{}

	for i, item := range CobranzaItemsToUpload.Payload {
		base := i * colsPerItem

		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8,
			base+9, base+10, base+11, base+12, base+13, base+14, base+15,
			base+16, base+17, base+18, base+19,
		))

		args = append(args,
			item.DiaCobro, item.Estatus, item.FechaEmision, item.FormaPago, item.MedioCobro,
			item.NumPoliza, item.Plan, item.TipoSeguro, item.Direccion.Calle, item.Direccion.CodigoPostal,
			item.Direccion.Ciudad, item.Direccion.Colonia, item.Direccion.Estado, item.Moneda,
			item.Telefono, item.SumaAsegurada, item.Email, item.Pais, agente_id,
		)
	}

	// Agregamos RETURNING al query de pólizas para obtener los IDs que cree la base de datos
	queryStart := `INSERT INTO polizas (
		dia_cobro, estatus, fecha_emision, forma_pago, medio_cobro, numpoliza, plan,
		tipo_seguro, addr_calle, addr_codigopostal, addr_ciudad, addr_colonia,
		addr_estado, moneda, telefono, suma_asegurada, email, pais, agente_id
	) VALUES `
	finalQuery := queryStart + strings.Join(placeholders, ",") + " RETURNING poliza_id, numpoliza"

	// Cambiamos Exec por Query para leer lo que arroja el RETURNING
	polizasRows, err := tx.Query(finalQuery, args...)
	if err != nil {
		fmt.Println("Error en bulk insert de polizas:", err)
		services.HandleResponseError(http.StatusConflict, "No se ha podido realizar la inserción de pólizas", w)
		return
	}

	// Mapeamos los IDs autogenerados indexándolos por el número de póliza texto
	polizasMap := make(map[string]int64)
	var totalPolizasInsertadas int64

	for polizasRows.Next() {
		var id int64
		var num string
		if err := polizasRows.Scan(&id, &num); err != nil {
			polizasRows.Close()
			services.HandleResponseError(http.StatusInternalServerError, "Error al procesar identificadores de pólizas", w)
			return
		}
		polizasMap[strings.ToLower(num)] = id
		totalPolizasInsertadas++
	}
	polizasRows.Close() // Cerramos este set de resultados para proceder con el siguiente

	// =========================================================================
	// PASO 2: BULK INSERT DE ASEGURADOS UTILIZANDO EL MAPA DE IDs
	// =========================================================================
	var nombresAsegurados []string
	var cumpleanosAsegurados []string
	var principalesAsegurados []bool
	var polizasIdsAsegurados []int64

	// Recorremos los datos que sabemos que se intentaron subir
	for _, item := range CobranzaItemsToUpload.Payload {
		polizaID, existe := polizasMap[strings.ToLower(item.NumPoliza)]
		if !existe {
			continue // Protección por si no se generó el ID (no debería pasar)
		}

		for _, asegurado := range item.Asegurados {
			nombresAsegurados = append(nombresAsegurados, asegurado.Nombre)
			cumpleanosAsegurados = append(cumpleanosAsegurados, asegurado.Cumpleanos)
			principalesAsegurados = append(principalesAsegurados, asegurado.IsPrincipal)
			polizasIdsAsegurados = append(polizasIdsAsegurados, polizaID) // Inyectamos el ID relacional real
		}
	}

	// Si las pólizas tenían asegurados en su payload, los metemos todos de un solo golpe masivo
	if len(nombresAsegurados) > 0 {
		// CORRECCIÓN: Casteamos explícitamente cada parámetro individual ($1::text[], $2::text[], etc.)
		// antes de pasárselo a UNNEST para que el driver nativo no se confunda con literales vacíos.
		queryAsegurados := `
				INSERT INTO asegurados (nombre_completo, birthday, is_principal, poliza_id)
				SELECT
					temp.nombre,
					NULLIF(temp.cumple, '')::timestamptz,
					temp.principal,
					temp.pol_id
				FROM UNNEST($1::text[], $2::text[], $3::boolean[], $4::bigint[])
				AS temp(nombre, cumple, principal, pol_id);`

		_, err = tx.Exec(queryAsegurados,
			pq.Array(nombresAsegurados),
			pq.Array(cumpleanosAsegurados),
			pq.Array(principalesAsegurados),
			pq.Array(polizasIdsAsegurados),
		)
		if err != nil {
			fmt.Println("Error en bulk insert de asegurados:", err)
			services.HandleResponseError(http.StatusConflict, "No se ha podido realizar la inserción de asegurados", w)
			return
		}
	}

	// =========================================================================
	// COMMIT: Si todo salió bien hasta aquí, guardamos permanentemente en la DB
	// =========================================================================
	if err := tx.Commit(); err != nil {
		fmt.Println("Error al confirmar transacción:", err)
		services.HandleResponseError(http.StatusInternalServerError, "Error al guardar los datos", w)
		return
	}

	// Respuesta al cliente
	message_obj := make(map[string]interface{})
	message_obj["message"] = fmt.Sprintf("¡Éxito! Se insertaron %d pólizas con sus respectivos asegurados correctamente.\n", totalPolizasInsertadas)
	fmt.Printf("¡Éxito! Se insertaron %d registros correctamente.\n", totalPolizasInsertadas)

	services.HandleResponseSuccessWithData(message_obj, w)
}

/*
*
Carga de una sola poliza
*/
func ApiPostPoliza(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	services.NewLogger().OriginAdvice("ApiPostPoliza")

	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var CobranzaItem internal.PostItem_Poliza
	var agente_id int
	var poliza_id int

	err := json.NewDecoder(r.Body).Decode(&CobranzaItem)
	if err != nil {
		fmt.Println(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}

	err = db.Client.QueryRow(`SELECT agente_id FROM agentes WHERE agente_uuid = $1`, agente_uuid).Scan(&agente_id)
	if err != nil {
		if err == sql.ErrNoRows {
			// CASO: No hay registros
			fmt.Println("El agente no existe en la base de datos")
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		}

		// CASO: Otro tipo de error (conexión, sintaxis, etc.)
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
	}

	_, err = db.Client.Exec(`
		INSERT INTO polizas (dia_cobro, estatus, fecha_emision, forma_pago, medio_cobro, numpoliza, plan,
			tipo_seguro, addr_calle, addr_codigopostal, addr_ciudad, addr_colonia,
			addr_estado, moneda, telefono, suma_asegurada, email, pais, agente_id) VALUES ($1, $2, $3,$4, $5, $6,$7, $8, $9,$10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		CobranzaItem.DiaCobro, CobranzaItem.Estatus, CobranzaItem.FechaEmision, CobranzaItem.FormaPago, CobranzaItem.MedioCobro,
		CobranzaItem.NumPoliza, CobranzaItem.Plan, CobranzaItem.TipoSeguro, CobranzaItem.Direccion.Calle, CobranzaItem.Direccion.CodigoPostal,
		CobranzaItem.Direccion.Ciudad, CobranzaItem.Direccion.Colonia, CobranzaItem.Direccion.Estado, CobranzaItem.Moneda,
		CobranzaItem.Telefono, CobranzaItem.SumaAsegurada, CobranzaItem.Email, CobranzaItem.Pais, agente_id)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // Unique Violation
				services.NewLogger().ErrorMessage(err.Error())
				services.HandleResponseError(http.StatusConflict, "El número de póliza ya está registrado en el sistema", w)
				return
			}
		}
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}

	err = db.Client.QueryRow(`SELECT poliza_id FROM polizas WHERE numpoliza = $1`, CobranzaItem.NumPoliza).Scan(&poliza_id)
	if err != nil {
		if err == sql.ErrNoRows {
			// CASO: No hay registros
			fmt.Println("El agente no existe en la base de datos")
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		}

		// CASO: Otro tipo de error (conexión, sintaxis, etc.)
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
	}

	for _, asegurado_item := range CobranzaItem.Asegurados {
		_, err = db.Client.Exec(`
			INSERT INTO asegurados (birthday, nombre_completo, is_principal, poliza_id)
			VALUES ($1, $2, $3,$4)`,
			asegurado_item.Cumpleanos, asegurado_item.Nombre, asegurado_item.IsPrincipal, poliza_id)
		if err != nil {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
			return
		}
	}

	services.HandleResponseSuccess(w)
}

func ApiGetDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetDetails")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var details internal.PolizasUserDetails

	query := `
		SELECT
				COUNT(*) as total,
				COUNT(CASE WHEN p.estatus = 'En Vigor' THEN 1 END) as activas,
			COUNT(CASE WHEN p.estatus != 'En Vigor' THEN 1 END) as inactivas,
			COUNT(CASE WHEN ppc.next_payment <= CURRENT_DATE + INTERVAL '5 days' THEN 1 END) as por_vencer,
			COUNT(CASE WHEN (ppc.next_payment - CURRENT_DATE) >= INTERVAL '5 days' AND ppl.paid_period != NULL THEN 1 END) as cobertura_activa,
			COUNT(CASE WHEN ppl.paid_period IS NULL THEN 1 END) as sin_pago_registrado
		FROM polizas p
		JOIN agentes a ON p.agente_id = a.agente_id
		JOIN polizas_payments_conf ppc ON p.poliza_id = ppc.poliza_id
		LEFT JOIN polizas_payments_log ppl ON p.poliza_id = ppl.poliza_id
		WHERE a.agente_uuid = $1`

	err := db.Client.QueryRow(query, userUUID).
		Scan(&details.Total, &details.Activas, &details.Inactivas,
			&details.PorVencer, &details.CoberturaActiva, &details.SinPagoRegistrado)
	if err != nil {
		if err == sql.ErrNoRows {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		} else {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		}
		return
	}
	// PENDIENTE DE PROBAR <---------------------
	services.HandleResponseSuccessWithData(details, w)
}

func ApiGetPolizasIds(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetPolizasIds")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)
	var polizasids []string
	polizasidspayload := make(map[string]interface{})

	rows, err := db.Client.Query(`
		SELECT numpoliza
		FROM polizas p
		JOIN agentes a ON p.agente_id = a.agente_id
		WHERE a.agente_uuid = $1`, userUUID)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
	}
	defer rows.Close()

	for rows.Next() {
		var numpoliza string

		if err := rows.Scan(&numpoliza); err != nil {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		}

		polizasids = append(polizasids, numpoliza)
	}

	if err = rows.Err(); err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
	}

	polizasidspayload["polizas"] = polizasids

	services.HandleResponseSuccessWithData(polizasidspayload, w)
}

func ApiGetPoliza(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetPoliza")

	agente_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)

	polizanum := GetField(r, 0)
	var agente_id int
	var poliza_id int

	err := db.Client.QueryRow(`SELECT agente_id FROM agentes WHERE agente_uuid = $1`, agente_uuid).Scan(&agente_id)

	rows, err := db.Client.Query(`
		SELECT  p.poliza_uuid, a.agente_uuid, p.dia_cobro, p.estatus, p.fecha_emision, p.forma_pago, p.medio_cobro,
				p.numpoliza, p.plan, p.tipo_seguro, p.addr_calle, p.addr_codigopostal, p.addr_ciudad, p.addr_colonia,
				p.addr_estado, p.moneda, p.pais, p.email, p.telefono, ppc.next_payment, p.poliza_id
		FROM polizas p
		JOIN polizas_payments_conf ppc
		ON p.poliza_id = ppc.poliza_id
		JOIN agentes a
		ON p.agente_id = a.agente_id
		WHERE p.numpoliza = $1
		AND a.agente_id = $2`, polizanum, agente_id)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.NewLogger().ErrorMessage("ola")
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}

	defer rows.Close()

	var cobranza internal.GetItem_Poliza
	for rows.Next() {
		err := rows.Scan(&cobranza.PolizaUUID, &cobranza.AgenteUUID, &cobranza.DiaCobro,
			&cobranza.Estatus, &cobranza.FechaEmision, &cobranza.FormaPago, &cobranza.MedioCobro,
			&cobranza.NumPoliza, &cobranza.Plan, &cobranza.TipoSeguro,
			&cobranza.Direccion.Calle, &cobranza.Direccion.CodigoPostal, &cobranza.Direccion.Ciudad,
			&cobranza.Direccion.Colonia, &cobranza.Direccion.Estado, &cobranza.Moneda, &cobranza.Pais,
			&cobranza.Email, &cobranza.Telefono, &cobranza.SiguientePago, &poliza_id)
		if err != nil {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
			return
		}
	}

	rows, err = db.Client.Query(`
		SELECT nombre_completo, birthday, is_principal FROM asegurados WHERE poliza_id = $1`, poliza_id)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var asegurado internal.Asegurado
		if err := rows.Scan(&asegurado.Nombre, &asegurado.Cumpleanos, &asegurado.IsPrincipal); err != nil {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		}

		cobranza.Asegurados = append(cobranza.Asegurados, asegurado)
	}

	if cobranza.NumPoliza == "" {
		services.HandleResponseError(http.StatusNotFound, "La poliza no está registrada", w)
		return
	}

	services.HandleResponseSuccessWithData(cobranza, w)
}

/*
*
Obtencion de todas las polizas
*/
func ApiGetPolizas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("Request from ApiGetPolizas")
	fmt.Printf("----------------------------------------\n")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var filters internal.GetItem_Poliza_Filters
	filters.Filters = make(map[string]string)

	queryParams := r.URL.Query()

	pageSize, _ := strconv.Atoi(queryParams.Get("pageSize"))
	currentPage, _ := strconv.Atoi(queryParams.Get("currentPage"))

	if pageSize <= 0 {
		pageSize = 10
	}

	if currentPage <= 0 {
		currentPage = 1
	}

	filters.PageSize = pageSize
	filters.CurentPage = currentPage

	// ==========================================================
	// FILTROS
	// ==========================================================

	if status := queryParams.Get("estatus"); status != "" {
		filters.Filters["estatus"] = status
	}

	if nextDue := queryParams.Get("next_due"); nextDue != "" {
		filters.Filters["next_due"] = nextDue
	}

	if numPoliza := queryParams.Get("numpoliza"); numPoliza != "" {
		filters.Filters["numpoliza"] = numPoliza
	}

	// ==========================================================
	// OBTENER AGENTE
	// ==========================================================

	err := db.Client.QueryRow(`
		SELECT agente_id
		FROM agentes
		WHERE agente_uuid=$1
	`, userUUID).Scan(&filters.Agente_id)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(
			http.StatusInternalServerError,
			err.Error(),
			w,
		)
		return
	}

	// ==========================================================
	// BASE QUERY
	// ==========================================================

	baseQuery := `
		FROM polizas p
		JOIN agentes a
			ON p.agente_id = a.agente_id
		JOIN polizas_payments_conf ppc
			ON ppc.poliza_id=p.poliza_id
		WHERE a.agente_id=$1
	`

	args := []interface{}{filters.Agente_id}
	argCount := 1

	// ==========================================================
	// FILTROS DINÁMICOS
	// ==========================================================

	for columna, valor := range filters.Filters {

		if valor == "" {
			continue
		}

		if columna == "next_due" && valor == "true" {
			baseQuery += `
				AND ppc.next_payment <= NOW() + INTERVAL '5 days'
			`
		} else if columna == "numpoliza" {

			argCount++

			baseQuery += fmt.Sprintf(
				" AND p.numpoliza ILIKE $%d",
				argCount,
			)

			args = append(
				args,
				fmt.Sprintf("%%%s%%", valor),
			)

		} else {

			argCount++

			baseQuery += fmt.Sprintf(
				" AND p.%s=$%d",
				columna,
				argCount,
			)

			args = append(args, valor)
		}
	}

	// ==========================================================
	// QUERY TOTAL
	// ==========================================================

	countQuery := "SELECT COUNT(*) " + baseQuery

	var totalRecords int

	err = db.Client.QueryRow(
		countQuery,
		args...,
	).Scan(&totalRecords)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())

		services.HandleResponseError(
			http.StatusInternalServerError,
			err.Error(),
			w,
		)

		return
	}

	// ==========================================================
	// PAGINACIÓN
	// ==========================================================

	offset := filters.PageSize *
		(filters.CurentPage - 1)

	totalPages := int(
		math.Ceil(
			float64(totalRecords) /
				float64(filters.PageSize),
		),
	)

	if totalPages <= 0 {
		totalPages = 1
	}

	// ==========================================================
	// QUERY DE DATOS
	// ==========================================================

	selectQuery := `
		SELECT
			p.dia_cobro, p.estatus, p.fecha_emision, p.forma_pago, p.medio_cobro, p.numpoliza,
			p.plan, p.tipo_seguro, p.addr_calle, p.addr_codigopostal, p.addr_ciudad, p.addr_colonia, p.addr_estado,
			ppc.next_payment, p.moneda, p.pais, p.telefono, p.email, p.suma_asegurada, p.last_modified, p.poliza_uuid
	` + baseQuery

	argCount++

	selectQuery += fmt.Sprintf(
		`
		ORDER BY ppc.next_payment ASC
		LIMIT $%d
		OFFSET $%d
	`,
		argCount,
		argCount+1,
	)

	args = append(
		args,
		filters.PageSize,
		offset,
	)

	rows, err := db.Client.Query(
		selectQuery,
		args...,
	)
	if err != nil {

		services.NewLogger().ErrorMessage(
			err.Error(),
		)

		services.HandleResponseError(
			http.StatusInternalServerError,
			err.Error(),
			w,
		)

		return
	}

	defer rows.Close()

	var polizas []internal.GetItem_Poliza

	for rows.Next() {

		var poliza internal.GetItem_Poliza

		err := rows.Scan(
			&poliza.DiaCobro, &poliza.Estatus, &poliza.FechaEmision,
			&poliza.FormaPago, &poliza.MedioCobro, &poliza.NumPoliza, &poliza.Plan, &poliza.TipoSeguro,
			&poliza.Direccion.Calle, &poliza.Direccion.CodigoPostal, &poliza.Direccion.Ciudad, &poliza.Direccion.Colonia,
			&poliza.Direccion.Estado, &poliza.SiguientePago, &poliza.Moneda, &poliza.Pais, &poliza.Telefono, &poliza.Email,
			&poliza.SumaAsegurada, &poliza.UltimaModificacion, &poliza.PolizaUUID,
		)
		if err != nil {

			rows.Close()

			services.HandleResponseError(
				http.StatusInternalServerError,
				err.Error(),
				w,
			)

			return
		}

		polizas = append(
			polizas,
			poliza,
		)
	}

	if polizas == nil {
		polizas = []internal.GetItem_Poliza{}
	}

	response := map[string]interface{}{
		"items": polizas,
		"total": totalRecords,
		"pages": totalPages,
	}

	services.HandleResponseSuccessWithData(
		response,
		w,
	)
}

/*
*
Obtencion de todas las polizas
*/
func ApiGetBirthdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("Request from ApiGetBirthdates")
	fmt.Printf("----------------------------------------\n")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var agente_id int

	err := db.Client.QueryRow(`SELECT agente_id FROM agentes WHERE agente_uuid = $1`, userUUID).Scan(&agente_id)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
	}

	rows, err := db.Client.Query(`
		WITH birthday_calc AS (
		    SELECT
		        a.nombre_completo,
		        a.birthday,
								p.numpoliza,
		        MAKE_DATE(
		            EXTRACT(YEAR FROM NOW())::int,
		            EXTRACT(MONTH FROM birthday)::int,
		            EXTRACT(DAY FROM birthday)::int
		        )::timestamp AS has_current_year_birthday
		    FROM asegurados a
						JOIN polizas p ON p.poliza_id = a.poliza_id
						JOIN agentes ag ON ag.agente_id = p.agente_id
						WHERE aG.agente_id = $1
		)
		SELECT
		    nombre_completo,
		    CASE
		        WHEN has_current_year_birthday < NOW() THEN has_current_year_birthday + INTERVAL '1 year'
		        ELSE has_current_year_birthday
		    END AS next_birthday,
						numpoliza
		FROM birthday_calc
		ORDER BY next_birthday ASC;
		`, agente_id)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
	}

	var asegurados_birthdates []internal.AseguradoBirthdate

	for rows.Next() {

		var asegurado_item internal.AseguradoBirthdate

		err := rows.Scan(&asegurado_item.NombreCompleto, &asegurado_item.Birthdate, &asegurado_item.Numpoliza)
		if err != nil {

			services.HandleResponseError(
				http.StatusInternalServerError,
				err.Error(),
				w,
			)

			return
		}

		asegurados_birthdates = append(
			asegurados_birthdates,
			asegurado_item,
		)
	}

	services.HandleResponseSuccessWithData(asegurados_birthdates, w)
}
