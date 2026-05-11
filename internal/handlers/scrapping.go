package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
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
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetPolizasCountByUser")

	var polizas_count int

	user_uuid := r.URL.Query().Get("uuid")

	err := db.Client.QueryRow(`SELECT COUNT(*) FROM polizas WHERE user_uuid = $1`, user_uuid).Scan(&polizas_count)
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
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	services.NewLogger().OriginAdvice("ApiPostPolizas")

	// Items recibidos desde el payload de la request
	var CobranzaItemsReceived internal.PostItems_Poliza
	// Items finales para subir
	var CobranzaItemsToUpload internal.PostItems_Poliza
	// Contratante enlazado para filtrar los registros
	contratante_uuid, _ := r.Context().Value(middlewares.UserIDKey).(string)

	// fmt.Println(contratante_uuid)

	err := json.NewDecoder(r.Body).Decode(&CobranzaItemsReceived)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	// /*
	// 	 *
	// 		*
	// 		* Validacion para verificar que vengan datos en la peticion
	// 		*
	// */
	if len(CobranzaItemsReceived.Payload) == 0 {
		services.HandleResponseError(http.StatusBadRequest, "No se ha recibido información", w)
		return
	}

	// // Se obtienen todos los registros del contratante
	rows, err := db.Client.Query(`
		SELECT numpoliza
		FROM polizas p
		WHERE p.user_uuid = $1`, contratante_uuid)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Ha ocurrido un error", w)
		return
	}

	// /*
	// 	 *
	// 		*
	// 		* Validacion en caso de que los registros que se quieren insertar ya existen en la DB
	// 		*
	// */
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

	for _, receivedItem := range CobranzaItemsReceived.Payload {
		if !existing_rows[strings.ToLower(receivedItem.NumPoliza)] {
			CobranzaItemsToUpload.Payload = append(CobranzaItemsToUpload.Payload, receivedItem)
			fmt.Println(receivedItem)
		}
	}
	defer rows.Close()

	if len(CobranzaItemsToUpload.Payload) == 0 {
		services.NewLogger().ErrorMessage("Todos los registros a ingresar ya existen, verifique la petición")
		services.HandleResponseError(http.StatusConflict, "Todos los registros ya se encuentran sincronizados", w)
		return
	}

	// /*
	// 	 *
	// 		*
	// 		* Armado de INSERT dinámico
	// 		*
	// */
	const colsPerItem = 16
	var placeholders []string
	var args []interface{}

	for i, item := range CobranzaItemsToUpload.Payload {
		base := i * colsPerItem

		placeholders = append(placeholders, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8,
			base+9, base+10, base+11, base+12, base+13, base+14, base+15, base+16,
		))

		// 	// Agregamos los valores al slice de interfaces (sin comillas manuales)
		args = append(args,
			item.Asegurado, item.Contratante, item.DiaCobro, item.Estatus,
			item.FechaEmision, item.FormaPago, item.MedioCobro, item.NumPoliza,
			item.Plan, item.TipoSeguro,
			item.Direccion.Calle, item.Direccion.CodigoPostal,
			item.Direccion.Ciudad, item.Direccion.Estado, item.Direccion.Colonia, contratante_uuid,
		)
	}

	// // 2. Construimos el query final
	queryStart := `INSERT INTO polizas (
	    asegurado, contratante, dia_cobro, estatus, fecha_emision,
	    forma_pago, medio_cobro, numpoliza, plan, tipo_seguro,
	    addr_calle, addr_codigopostal, addr_ciudad,
	    addr_estado, addr_colonia, user_uuid
	   ) VALUES `

	finalQuery := queryStart + strings.Join(placeholders, ",")

	res, err := db.Client.Exec(finalQuery, args...)
	if err != nil {
		fmt.Println("Error en bulk insert:", err)
		services.HandleResponseError(http.StatusConflict, "No se ha podido realizar la inserción", w)
		return
	}

	// // 2. Extraemos la cantidad de filas afectadas
	count, err := res.RowsAffected()
	if err != nil {
		// Es raro que falle aquí, pero es buena práctica validarlo
		fmt.Println("Error al obtener filas afectadas:", err)
		services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
	}

	message_obj := make(map[string]interface{})
	// // 3. Validamos o logueamos el resultado
	if count == 0 {
		fmt.Println("Cuidado: El insert terminó sin errores pero no se insertó ninguna fila.")
	} else {
		message_obj["message"] = fmt.Sprintf("¡Éxito! Se insertaron %d registros correctamente.\n", count)
		fmt.Printf("¡Éxito! Se insertaron %d registros correctamente.\n", count)
	}

	services.HandleResponseSuccessWithData(message_obj, w)
}

/*
*
Carga de una sola poliza
*/
func ApiPostPoliza(w http.ResponseWriter, r *http.Request) {
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	services.NewLogger().OriginAdvice("ApiPostPoliza")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var CobranzaItem internal.PostItem_Poliza
	var PolizaItemReturn internal.GetItem_Poliza

	err := json.NewDecoder(r.Body).Decode(&CobranzaItem)
	if err != nil {
		fmt.Println(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}

	_, err = db.Client.Exec(`
		INSERT INTO polizas (asegurado, contratante, dia_cobro, estatus, fecha_emision,
							forma_pago, medio_cobro, numpoliza, plan, tipo_seguro, addr_calle, addr_codigoPostal,
							addr_ciudad, addr_estado, addr_colonia, user_uuid) VALUES ($1, $2, $3,$4, $5, $6,$7, $8, $9,$10, $11, $12, $13, $14, $15, $16)`,
		CobranzaItem.Asegurado, CobranzaItem.Contratante, CobranzaItem.DiaCobro, CobranzaItem.Estatus, CobranzaItem.FechaEmision,
		CobranzaItem.FormaPago, CobranzaItem.MedioCobro, CobranzaItem.NumPoliza, CobranzaItem.Plan, CobranzaItem.TipoSeguro,
		CobranzaItem.Direccion.Calle, CobranzaItem.Direccion.CodigoPostal, CobranzaItem.Direccion.Ciudad, CobranzaItem.Direccion.Estado,
		CobranzaItem.Direccion.Colonia, userUUID)
	if err != nil {
		// Comprobación segura de tipo de error
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23505": // Unique Violation
				fmt.Println("Registro duplicado detectado:", pqErr.Detail)
				services.HandleResponseError(http.StatusConflict, "El número de póliza ya está registrado en el sistema", w)
				return
			}
		}

		// Si no es un error de duplicado o no es un pq.Error
		fmt.Printf("Error de base de datos: %v\n", err)
		services.HandleResponseError(http.StatusInternalServerError, "Error interno al procesar la póliza", w)
		return
	}

	rows, err := db.Client.Query(`
		SELECT  p.poliza_uuid, p.asegurado, p.contratante, p.dia_cobro, p.estatus, p.fecha_emision, p.forma_pago, p.medio_cobro,
			 	p.numpoliza, p.plan, p.tipo_seguro, p.addr_calle, p.addr_codigopostal, p.addr_ciudad, p.addr_colonia, p.addr_estado,
				ppc.next_payment, p.user_uuid
		FROM polizas p
		JOIN polizas_payments_conf ppc
		ON p.poliza_uuid = ppc.poliza_uuid
		WHERE p.numpoliza = $1`, CobranzaItem.NumPoliza)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}

	for rows.Next() {
		err := rows.Scan(&PolizaItemReturn.PolizaUUID, &PolizaItemReturn.Asegurado, &PolizaItemReturn.Contratante, &PolizaItemReturn.DiaCobro,
			&PolizaItemReturn.Estatus, &PolizaItemReturn.FechaEmision, &PolizaItemReturn.FormaPago, &PolizaItemReturn.MedioCobro,
			&PolizaItemReturn.NumPoliza, &PolizaItemReturn.Plan, &PolizaItemReturn.TipoSeguro,
			&PolizaItemReturn.Direccion.Calle, &PolizaItemReturn.Direccion.CodigoPostal, &PolizaItemReturn.Direccion.Ciudad,
			&PolizaItemReturn.Direccion.Colonia, &PolizaItemReturn.Direccion.Estado, &PolizaItemReturn.SiguientePago, &PolizaItemReturn.UserUUID)
		if err != nil {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
			return
		}
	}

	services.HandleResponseSuccessWithData(PolizaItemReturn, w)
}

func ApiGetDetails(w http.ResponseWriter, r *http.Request) {
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetDetails")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var details internal.PolizasUserDetails

	query := `
		SELECT
	    	COUNT(*) as total,
		    COUNT(CASE WHEN p.estatus = 'En Vigor' THEN 1 END) as activas,
			COUNT(CASE WHEN p.estatus != 'En Vigor' THEN 1 END) as inactivas,
			COUNT(CASE WHEN (ppc.next_payment - CURRENT_DATE) <= INTERVAL '5 days' THEN 1 END) as por_vencer,
			COUNT(CASE WHEN (ppc.next_payment - CURRENT_DATE) >= INTERVAL '5 days' AND ppl.paid_period != NULL THEN 1 END) as cobertura_activa,
			COUNT(CASE WHEN ppl.paid_period IS NULL THEN 1 END) as sin_pago_registrado
		FROM polizas p
		JOIN polizas_payments_conf ppc ON p.poliza_uuid = ppc.poliza_uuid
		LEFT JOIN polizas_payments_log ppl ON p.poliza_uuid = ppl.poliza_id
		WHERE user_uuid = $1`

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
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetPolizasIds")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)
	var polizasids []string
	polizasidspayload := make(map[string]interface{})

	rows, err := db.Client.Query(`SELECT numpoliza FROM polizas WHERE user_uuid = $1`, userUUID)
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
	AllowOrigins(w, r)
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	services.NewLogger().OriginAdvice("ApiGetPoliza")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	polizanum := GetField(r, 0)

	rows, err := db.Client.Query(`
		SELECT  p.poliza_uuid, p.asegurado, p.contratante, p.dia_cobro, p.estatus, p.fecha_emision, p.forma_pago, p.medio_cobro,
			 	p.numpoliza, p.plan, p.tipo_seguro, p.addr_calle, p.addr_codigopostal, p.addr_ciudad, p.addr_colonia, p.addr_estado,
				ppc.next_payment, p.user_uuid
		FROM polizas p
		JOIN polizas_payments_conf ppc
		ON p.poliza_uuid = ppc.poliza_uuid
		WHERE p.numpoliza = $1
		AND user_uuid = $2`, polizanum, userUUID)
	if err != nil {
		services.NewLogger().ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusInternalServerError, err.Error(), w)
		return
	}

	var cobranza internal.GetItem_Poliza
	for rows.Next() {
		err := rows.Scan(&cobranza.PolizaUUID, &cobranza.Asegurado, &cobranza.Contratante, &cobranza.DiaCobro,
			&cobranza.Estatus, &cobranza.FechaEmision, &cobranza.FormaPago, &cobranza.MedioCobro,
			&cobranza.NumPoliza, &cobranza.Plan, &cobranza.TipoSeguro,
			&cobranza.Direccion.Calle, &cobranza.Direccion.CodigoPostal, &cobranza.Direccion.Ciudad,
			&cobranza.Direccion.Colonia, &cobranza.Direccion.Estado, &cobranza.SiguientePago, &cobranza.UserUUID)
		if err != nil {
			services.NewLogger().ErrorMessage(err.Error())
			services.HandleResponseError(http.StatusBadRequest, err.Error(), w)
			return
		}
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
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	fmt.Println("Request from ApiGetPolizas")
	fmt.Printf("----------------------------------------\n")

	slug := GetField(r, 0)

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	fmt.Println(`Hola`)
	fmt.Println(userUUID)

	filtros := map[string]interface{}{}

	switch slug {
	case "all":
		filtros = map[string]interface{}{}
		break
	case "active":
		filtros["estatus"] = "En Vigor"
		break
	case "inactive":
		filtros["estatus"] = "Anulada"
		break
	case "almost_due":
		filtros["next_payment"] = "hello"
		break
	}

	rows, err := GetPolizasDinamicas(db.Client, filtros, userUUID)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var cobranzas []internal.GetItem_Poliza
	for rows.Next() {
		var cobranza internal.GetItem_Poliza
		err := rows.Scan(&cobranza.PolizaUUID, &cobranza.Asegurado, &cobranza.Contratante, &cobranza.DiaCobro,
			&cobranza.Estatus, &cobranza.FechaEmision, &cobranza.FormaPago, &cobranza.MedioCobro,
			&cobranza.NumPoliza, &cobranza.Plan, &cobranza.TipoSeguro,
			&cobranza.Direccion.Calle, &cobranza.Direccion.CodigoPostal, &cobranza.Direccion.Ciudad,
			&cobranza.Direccion.Colonia, &cobranza.Direccion.Estado, &cobranza.SiguientePago, &cobranza.UserUUID)
		if err != nil {
			fmt.Printf("Error con: %s\n", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cobranzas = append(cobranzas, cobranza)
	}

	services.HandleResponseSuccessWithData(cobranzas, w)
}
