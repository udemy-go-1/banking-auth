package app

import (
	"encoding/json"
	"github.com/udemy-go-1/banking-auth/dto"
	"github.com/udemy-go-1/banking-auth/service"
	"github.com/udemy-go-1/banking-lib/errs"
	"github.com/udemy-go-1/banking-lib/logger"
	"net/http"
)

type RegistrationHandler struct { //REST handler (adapter)
	service service.RegistrationService
}

func (h RegistrationHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	var registrationRequest dto.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&registrationRequest); err != nil {
		logger.Error("Error while decoding json body of registration request: " + err.Error())
		writeJsonResponse(w, http.StatusBadRequest, errs.NewMessageObject(err.Error()))
		return
	}

	//if appErr := registrationRequest.Validate(); appErr != nil { //TODO: parse fields + sanitize
	//	writeJsonResponse(w, appErr.Code, appErr.AsMessage())
	//	return
	//}

	response, appErr := h.service.Register(registrationRequest)
	if appErr != nil {
		writeJsonResponse(w, appErr.Code, appErr.AsMessage())
		return
	}

	writeJsonResponse(w, http.StatusCreated, response)
}

func (h RegistrationHandler) CheckRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	tokenString := r.URL.Query().Get("ott")
	if tokenString == "" {
		writeJsonResponse(w, http.StatusBadRequest, errs.NewMessageObject(errs.MessageMissingToken))
		return
	}

	isConfirmed, appErr := h.service.CheckRegistration(tokenString)
	if appErr != nil {
		writeJsonResponse(w, appErr.Code, appErr.AsMessage())
		return
	}
	if isConfirmed {
		writeJsonResponse(w, http.StatusOK, errs.NewMessageObject("Registration already confirmed"))
		return
	}

	writeJsonResponse(w, http.StatusOK, errs.NewMessageObject(""))
}

func (h RegistrationHandler) FinishRegistrationHandler(w http.ResponseWriter, r *http.Request) {

}
