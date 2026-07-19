package main

import (
	"log"
	"net/http"
)

func main() {
	const port = "8080"
	const filePath = "."
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fileserver := http.FileServer(http.Dir(filePath))
	mux.Handle("/app/", http.StripPrefix("/app", fileserver))
	mux.HandleFunc("/healthz", health)

	println("Server Listening on port 8080")
	log.Fatal(server.ListenAndServe())
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}
