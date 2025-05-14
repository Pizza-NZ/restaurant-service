package api

import "net/http"

func BadRequest(w http.ResponseWriter, message string) {
	http.Error(w, message, http.StatusBadRequest)
}

func InternalServerError(w http.ResponseWriter, err error) {
	// Log err
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func NotFound(w http.ResponseWriter, err error) {
	// Log err
	http.Error(w, err.Error(), http.StatusNotFound)
}
