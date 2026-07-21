package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"myapp/internal/database"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiconfig struct {
	FileServerHits atomic.Int32
	DB             *database.Queries
}

type validation struct {
	Body string `json:"body"`
}

func (cfg *apiconfig) MiddleWareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return
	}
	dbQueries := database.New(db)
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
		DB:             dbQueries,
	}
	mux.Handle("/app/", apiCfg.MiddleWareMetricInc(http.StripPrefix("/app", fileserver))) //strip the url bcs fileServer search file in . directory not in /app/index.html
	mux.HandleFunc("GET /api/healthz", health)
	mux.HandleFunc("GET /admin/metrics", apiCfg.countRequests)
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetRequests)
	mux.HandleFunc("POST /api/validate_chirp", ValidateChirp)

	println("Server Listening on port 8080")
	log.Fatal(server.ListenAndServe())
}

func ValidateChirp(w http.ResponseWriter, r *http.Request) {

	valid := validation{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&valid)

	if err != nil {
		RespondWithErr(w, http.StatusBadRequest, "Something went wrong")
		return
	}

	if len(valid.Body) > 140 {
		RespondWithErr(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	ReplaceBadword(&valid.Body)

	RespondWithJson(w, http.StatusOK, map[string]string{"cleaned_body": valid.Body})
}

func ReplaceBadword(sentence *string) {
	words := []string{"kerfuffle", "sharbert", "fornax"}
	result := strings.Split(*sentence, " ")
	//found  := false
	for i, w := range result {
		for _, f := range words {
			if strings.ToLower(w) == f {
				result[i] = "****"
			}
		}
	}

	*sentence = strings.Join(result, " ")
}

func RespondWithJson(w http.ResponseWriter, code int, payload interface{}) error {
	resp, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	w.Write(resp)
	return nil
}

func RespondWithErr(w http.ResponseWriter, code int, msg string) error {
	return RespondWithJson(w, code, map[string]string{"error": msg})
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func (cfg *apiconfig) countRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset= utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
    <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
    </body>
</html>`, cfg.FileServerHits.Load())))
}

func (cfg *apiconfig) ResetRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset= utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.FileServerHits.Swap(0)
	w.Write([]byte(fmt.Sprintf("Hits: %d", cfg.FileServerHits.Load())))
}
