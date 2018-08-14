package main

import (
	"net/http"
)

func CorsOptionsHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w).WriteHeader(http.StatusOK)
}
