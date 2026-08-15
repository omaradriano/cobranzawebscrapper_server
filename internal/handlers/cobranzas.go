package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/omaradriano/cobranzawebscrapper_server/internal/dto"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/middlewares"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/models"
	"github.com/omaradriano/cobranzawebscrapper_server/internal/services"
)

func ApiSetPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Methods", "PATCH")

	fmt.Println("Request from ApiPatchPayment")
	fmt.Printf("----------------------------------------\n")

	userUUID, _ := r.Context().Value(middlewares.UserIDKey).(string)

	var item dto.CobranzaItemPayment

	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, "Error decoding JSON", w)
		return
	}

	agenteID, err := deps.AgenteRepo.FindIDByUUID(r.Context(), userUUID)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}

	polizaID, err := findPolizaIDByUUID(r, item.Poliza)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}

	paidPeriod, err := time.Parse(time.RFC3339, item.PaidPeriod)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusBadRequest, "Formato de fecha inválido", w)
		return
	}

	aID := int64(agenteID)
	pID := int64(polizaID)
	paymentLog := &models.PaymentLog{
		PolizaID:   &pID,
		AgenteID:   &aID,
		PaidPeriod: paidPeriod,
	}

	err = deps.PaymentRepo.CreateLog(r.Context(), paymentLog)
	if err != nil {
		services.Log.ErrorMessage(err.Error())
		services.HandleResponseError(http.StatusConflict, err.Error(), w)
		return
	}

	services.HandleResponseSuccess(w)
}

func findPolizaIDByUUID(r *http.Request, polizaUUID string) (int, error) {
	var poliza models.Poliza
	err := deps.DB.WithContext(r.Context()).
		Select("poliza_id").
		Where("poliza_uuid = ?", polizaUUID).
		First(&poliza).Error
	if err != nil {
		return 0, err
	}
	id, _ := strconv.Atoi(fmt.Sprintf("%d", poliza.PolizaID))
	return id, nil
}
