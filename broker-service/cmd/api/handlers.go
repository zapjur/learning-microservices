package main

import (
	"net/http"
)

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	err := app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		err := app.writeError(w, err)
		if err != nil {
			return
		}
		return
	}
}
