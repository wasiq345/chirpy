package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiconfig struct {
	FileServerHits atomic.Int32
}

func (cfg *apiconfig) MiddleWareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	const port = "8080"
	const filePath = "."
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	fileserver := http.FileServer(http.Dir(filePath))
	apiCfg := apiconfig{
		FileServerHits: atomic.Int32{},
	}
	mux.Handle("/app/", apiCfg.MiddleWareMetricInc(http.StripPrefix("/app", fileserver))) //strip the url bcs fileServer search file in . directory not in /app/index.html
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/metrics", apiCfg.countRequests)
	mux.HandleFunc("/reset", apiCfg.ResetRequests)

	println("Server Listening on port 8080")
	log.Fatal(server.ListenAndServe())
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiconfig) countRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset= utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.FileServerHits.Load())))
}

func (cfg *apiconfig) ResetRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset= utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.FileServerHits.Swap(0)
	w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.FileServerHits.Load())))
}
