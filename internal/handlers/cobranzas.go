package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/omaradriano/cobranzawebscrapper_server/db"
	"github.com/omaradriano/cobranzawebscrapper_server/internal"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

func GetField(r *http.Request, index int) string {
	fields := r.Context().Value(internal.CtxKey{}).([]string)
	return fields[index]
}

func ApiPatchItemPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	fmt.Println("Request from ApiPatchPayment")
	fmt.Printf("----------------------------------------\n")

	var item internal.CobranzaItemPayment

	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	vQuery := `	INSERT INTO polizas_payments_log (poliza_id, asegurador, paid_period)
				VALUES ($1, $2, $3)`
	fmt.Printf(`INSERT INTO polizas_payments_log (poliza_id, asegurador, paid_period)
				VALUES (%s, %s, %s)`, item.Poliza, item.Asegurador, item.PaidPeriod)
	_, err = db.Client.Exec(vQuery, item.Poliza, item.Asegurador, item.PaidPeriod)
	if err != nil {
		// fmt.Println("Error executing query:", err)
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}
	services.HandleResponseSuccess(w)
}

func ApiPostCobranzaAllItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	fmt.Println("Request from ApiPostCobranzaAllItems")
	fmt.Printf("----------------------------------------\n")

	// Items recibidos desde el payload de la request
	var CobranzaItemsReceived internal.CobranzaItems
	// Items finales para subir
	var CobranzaItemsToUpload internal.CobranzaItems
	// Contratante enlazado para filtrar los registros
	var contratante_uuid, contratante_id string

	err := json.NewDecoder(r.Body).Decode(&CobranzaItemsReceived)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	/*
		 *
			*
			* Validacion para verificar que vengan datos en la peticion
			*
	*/
	if len(CobranzaItemsReceived.Payload) == 0 {
		services.HandleResponseError(http.StatusBadRequest, "No se ha recibido información", w)
		return
	}

	// Se obtiene el ID del contratante para filtrar los registros
	contratante_uuid = "94e4e62c-723f-48f4-b81f-f0329fea0b82"

	// Se obtienen todos los registros del contratante
	rows, err := db.Client.Query("SELECT ua.user_uuid, ua.user_id FROM polizas p JOIN agentes ua ON ua.user_id = p.user_id WHERE ua.user_uuid = $1", contratante_uuid)
	if err != nil {
		services.HandleResponseError(http.StatusInternalServerError, "Ha ocurrido un error", w)
		return
	}

	/*
		 *
			*
			* Validacion en caso de que los registros que se quieren insertar ya existen en la DB
			*
	*/
	cobranzasAlreadyExist := make(map[string]bool)
	for rows.Next() {
		var numPoliza string
		if err := rows.Scan(&numPoliza, &contratante_id); err != nil {
			fmt.Println("Error escaneando:", err)
			services.HandleResponseError(500, "Error de base de datos", w)
			return
		}
		cobranzasAlreadyExist[strings.ToLower(numPoliza)] = true
	}

	for _, receivedItem := range CobranzaItemsReceived.Payload {
		if !cobranzasAlreadyExist[strings.ToLower(receivedItem.NumPoliza)] {
			CobranzaItemsToUpload.Payload = append(CobranzaItemsToUpload.Payload, receivedItem)

			cobranzasAlreadyExist[strings.ToLower(receivedItem.NumPoliza)] = true
		}
	}
	defer rows.Close()

	if len(CobranzaItemsToUpload.Payload) == 0 {
		services.HandleResponseError(409, "Todos los registros a ingresar ya existen, verifique la petición.", w)
		return
	}

	/*
		 *
			*
			* Armado de INSERT dinámico
			*
	*/
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

		// Agregamos los valores al slice de interfaces (sin comillas manuales)
		args = append(args,
			item.DiaCobro, item.Estatus,
			item.FechaEmision, item.FormaPago, item.MedioCobro, item.NumPoliza,
			item.Plan, item.TipoSeguro, contratante_id,
			item.Direccion.Calle, item.Direccion.CodigoPostal,
			item.Direccion.Ciudad, item.Direccion.Estado, item.Direccion.Colonia,
		)
	}

	// 2. Construimos el query final
	queryStart := `INSERT INTO polizas (
	    asegurado, contratante, dia_cobro, estatus, fecha_emision,
	    forma_pago, medio_cobro, numpoliza, plan, tipo_seguro,
	    user_id, addr_calle, addr_codigoPostal, addr_ciudad,
	    addr_estado, addr_colonia
    ) VALUES `

	finalQuery := queryStart + strings.Join(placeholders, ",")

	res, err := db.Client.Exec(finalQuery, args...)
	if err != nil {
		fmt.Println("Error en bulk insert:", err)
		services.HandleResponseError(409, "No se ha podido realizar la inserción", w)
		return
	}

	// 2. Extraemos la cantidad de filas afectadas
	count, err := res.RowsAffected()
	if err != nil {
		// Es raro que falle aquí, pero es buena práctica validarlo
		fmt.Println("Error al obtener filas afectadas:", err)
	}

	// 3. Validamos o logueamos el resultado
	if count == 0 {
		fmt.Println("Cuidado: El insert terminó sin errores pero no se insertó ninguna fila.")
	} else {
		fmt.Printf("¡Éxito! Se insertaron %d registros correctamente.\n", count)
	}

	services.HandleResponseSuccess(w)
}

// func ApiPatchFewCobranzas(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("Request from ApiPatchFewCobranzas")
// 	fmt.Printf("----------------------------------------\n")

// 	var CobranzaPatchItems internal.CobranzaPatchItems

// 	if err := json.NewDecoder(r.Body).Decode(&CobranzaPatchItems); err != nil {
// 		services.HandleResponseError(400, "Invalid JSON", w)
// 		return
// 	}

// 	if len(CobranzaPatchItems.Payload) == 0 {
// 		services.HandleResponseError(400, "No items to patch", w)
// 		return
// 	}

// 	contratanteID := CobranzaPatchItems.Payload[0].ContratanteID

// 	for _, item := range CobranzaPatchItems.Payload {
// 		query := fmt.Sprintf("UPDATE clients SET allownotifications = %t WHERE numpoliza = '%s' AND user_id = %d;",
// 			item.Notificaciones, item.NumPoliza, contratanteID)
// 		_, err := db.Client.Exec(query)
// 		if err != nil {
// 			// fmt.Println(err.Error())
// 			services.HandleResponseError(409, "No se ha podido realizar la actualizacion de registros", w)
// 			return
// 		}
// 	}

// 	services.HandleResponseSuccess(w)
// }

func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
