package controller

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
)

type ErrorHandler interface {
	ServeHTTP(w http.ResponseWriter, req *http.Request, err error, code int)
}

var DefaultHandler ErrorHandler = &StdHandler{}

type StdHandler struct {
}

type ErrorMessage struct {
	Message string `json:"message"`
}

func (e *StdHandler) ServeHTTP(w http.ResponseWriter, req *http.Request, err error, code int) {
	statusCode := code
	if e, ok := err.(net.Error); ok {
		if e.Timeout() {
			statusCode = http.StatusGatewayTimeout
		} else {
			statusCode = http.StatusBadGateway
		}
	} else if err == io.EOF {
		statusCode = http.StatusBadGateway
	}
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")

	var message string
	if err != nil {
		message = err.Error()
	} else {
		message = http.StatusText(code)
	}
	errMessage := ErrorMessage {
		Message : message,
	}
	if err := json.NewEncoder(w).Encode(errMessage); err != nil {
		log.Error("Encoder message fail.")
	}

}
