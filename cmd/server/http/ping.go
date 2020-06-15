package http

import (
	"fmt"
	"net/http"
)

func ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}
