package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	if err := app.writeJSON(w, http.StatusOK, payload); err != nil {
		app.logAndWriteError(w, err)
	}
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	if err := app.readJSON(w, r, &requestPayload); err != nil {
		app.logAndWriteError(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	default:
		app.logAndWriteError(w, errors.New("invalid action"))
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	jsonData, err := json.MarshalIndent(a, "", "\t")
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Println(err)
		}
	}()

	if response.StatusCode == http.StatusUnauthorized {
		app.logAndWriteError(w, errors.New("unauthorized"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.logAndWriteError(w, errors.New("authentication service error"))
		return
	}

	var jsonFromService jsonResponse
	if err = json.NewDecoder(response.Body).Decode(&jsonFromService); err != nil {
		app.logAndWriteError(w, err)
		return
	}

	if jsonFromService.Error {
		app.logAndWriteError(w, errors.New("authentication failed"), http.StatusUnauthorized)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Authenticated",
		Data:    jsonFromService.Data,
	}

	if err = app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		app.logAndWriteError(w, err, http.StatusInternalServerError)
	}
}

func (app *Config) logAndWriteError(w http.ResponseWriter, err error, statusCodes ...int) {
	log.Println(err)
	statusCode := http.StatusBadRequest
	if len(statusCodes) > 0 {
		statusCode = statusCodes[0]
	}
	err = app.writeError(w, err, statusCode)
	if err != nil {
		log.Println(err)
	}
}
