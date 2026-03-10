package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/omaradriano/cobranzawebscrapper_server/auxiliar"
)

func GetField(r *http.Request, index int) string {
	fields := r.Context().Value(CtxKey{}).([]string)
	return fields[index]
}

type CtxKey struct{}

func ApiGetCobranzasItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

	// fmt.Fprintf(w, "Se ha escrito sobre apiGetCobranzasItems\n")

	data, _ := json.Marshal(auxiliar.DbItemsTest)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func ApiPostCobranzasItems(w http.ResponseWriter, r *http.Request) {
	slug := GetField(r, 0)
	id, _ := strconv.Atoi(GetField(r, 1))
	fmt.Fprintf(w, "apiUpdateWidgetPart %s %d\n", slug, id)
}
