package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	server := http.Server{Handler: mux, Addr: ":8080"}
	if err := server.ListenAndServe(); err != nil {
	  log.Fatal(err)
	}
}