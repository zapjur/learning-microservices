package main

import (
	"broker-service/event"
	"broker-service/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"net/rpc"
	"time"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
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
	case "log":
		//app.logItem(w, requestPayload.Log) communicating via api
		//app.logEventViaRabbit(w, requestPayload.Log) communicating via rabbitmq
		app.logItemViaRPC(w, requestPayload.Log) // communicating via rpc
	case "mail":
		app.sendMail(w, requestPayload.Mail)
	default:
		app.logAndWriteError(w, errors.New("invalid action"))
	}
}

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) logItemViaRPC(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}
	rpcPayload := RPCPayload{
		Name: l.Name,
		Data: l.Data,
	}

	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	if err = app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		app.logAndWriteError(w, err)
		return
	}
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	err := app.pushToQueue(l.Name, l.Data)
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "logged via RabbitMQ",
	}

	if err = app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		app.logAndWriteError(w, err)
	}
}

func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	j, err := json.MarshalIndent(payload, "", "\t")
	if err != nil {
		return err
	}
	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}

func (app *Config) logItem(w http.ResponseWriter, entry LogPayload) {
	jsonData, err := json.MarshalIndent(entry, "", "\t")
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
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

	if response.StatusCode != http.StatusAccepted {
		app.logAndWriteError(w, errors.New("log service error"))
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	app.writeJSON(w, http.StatusAccepted, payload)
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

func (app *Config) sendMail(w http.ResponseWriter, msg MailPayload) {
	jsonData, err := json.MarshalIndent(msg, "", "\t")
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	mailServiceURL := "http://mailer-service/send"

	request, err := http.NewRequest("POST", mailServiceURL, bytes.NewBuffer(jsonData))
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

	if response.StatusCode != http.StatusAccepted {
		app.logAndWriteError(w, errors.New("mail service error"))
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + msg.To

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

func (app *Config) LogViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	if err := app.readJSON(w, r, &requestPayload); err != nil {
		app.logAndWriteError(w, err)
		return
	}

	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	defer func(conn *grpc.ClientConn) {
		err = conn.Close()
		if err != nil {
			log.Printf("error closing connection: %v", err)
		}
	}(conn)

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})

	if err != nil {
		app.logAndWriteError(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via gRPC"

	if err = app.writeJSON(w, http.StatusAccepted, payload); err != nil {
		app.logAndWriteError(w, err)
	}
}
