package main

import (
	"log"
	"net/http"
)

func (app *Config) sendMail(w http.ResponseWriter, r *http.Request) {
	type mailMessage struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}

	var requestPayload mailMessage

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	msg := Message{
		From:    requestPayload.From,
		To:      requestPayload.To,
		Subject: requestPayload.Subject,
		Data:    requestPayload.Message,
	}

	err = app.Mailer.SendSMTPMessage(msg)
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}
	payload := jsonResponse{
		Error:   false,
		Message: "Mail sent successfully to " + requestPayload.To,
	}

	if err = app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		app.logAndWriteError(w, err)
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
