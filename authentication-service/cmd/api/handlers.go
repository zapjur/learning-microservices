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
		err = app.writeError(w, err, http.StatusBadRequest)
		if err != nil {
			log.Println(err)
		}
		return
	}

	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		err = app.writeError(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		if err != nil {
			log.Println(err)
		}
		return
	}

	valid, err := user.PasswordMatches(requestPayload.Password)

	if err != nil || !valid {
		err = app.writeError(w, errors.New("invalid credentials"), http.StatusUnauthorized)
		if err != nil {
			log.Println(err)
		}
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		log.Println(err)
		return
	}
}
