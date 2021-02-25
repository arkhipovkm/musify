package server

import (
	"encoding/base64"
	"log"
	"net/http"
)

func decodeBase64URI(base64EncodedURI string) (string, error) {
	decodedURI, err := base64.URLEncoding.DecodeString(base64EncodedURI)
	if err != nil {
		return "", err
	}
	return string(decodedURI), nil
}

func handleError(w *http.ResponseWriter, err error) {
	log.Println(err.Error())
	(*w).WriteHeader(http.StatusInternalServerError)
	(*w).Write([]byte(err.Error()))
}
