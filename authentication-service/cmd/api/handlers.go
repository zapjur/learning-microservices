package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (app *Config) authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.logAndWriteError(w, err, http.StatusBadRequest)
		return
	}

	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.logAndWriteError(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.logAndWriteError(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}

	if err = app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		log.Println(err)
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
