package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"myapp/internal/database"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiconfig struct {
	FileServerHits atomic.Int32
	DB             *database.Queries
	Platform       string
}

type Chirp struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	User_id    uuid.UUID `json:"user_id"`
}

type user struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Email      string    `json:"email"`
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
		log.Fatal(err)
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
		Platform:       os.Getenv("PLATFORM"),
	}
	mux.Handle("/app/", apiCfg.MiddleWareMetricInc(http.StripPrefix("/app", fileserver))) //strip the url bcs fileServer search file in . directory not in /app/index.html
	mux.HandleFunc("POST /api/users", apiCfg.createUsers)
	mux.HandleFunc("GET /api/healthz", health)
	mux.HandleFunc("GET /admin/metrics", apiCfg.countRequests)
	mux.HandleFunc("POST /admin/reset", apiCfg.ResetRequests)
	mux.HandleFunc("POST /api/chirps", apiCfg.CreateChirps)

	println("Server Listening on port 8080")
	log.Fatal(server.ListenAndServe())
}

func (apicfg *apiconfig) createUsers(w http.ResponseWriter, r *http.Request) {

	User := user{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&User)

	defer r.Body.Close()

	if err != nil {
		RespondWithErr(w, http.StatusBadRequest, "Couldn't Create User")
		return
	}

	U, err := apicfg.DB.CreateUser(r.Context(), User.Email)
	if err != nil {
		RespondWithErr(w, http.StatusInternalServerError, "Couldn't Create User")
		return
	}

	User.Created_at = U.CreatedAt
	User.Updated_at = U.UpdatedAt
	User.Id = U.ID
	User.Email = U.Email

	RespondWithJson(w, http.StatusCreated, User)
}

func (cfg *apiconfig) CreateChirps(w http.ResponseWriter, r *http.Request) {
	chirp := Chirp{}
	err := ValidateChirp(w, r, &chirp)
	if err != nil {
		return
	}

	C, err := cfg.DB.CreateChirp(r.Context(), database.CreateChirpParams{Body: chirp.Body, Userid: chirp.User_id})

	if err != nil {
		RespondWithErr(w, http.StatusInternalServerError, "Couldn't Create Chirp")
		return
	}

	chirp.Body = C.Body
	chirp.Created_at = C.CreatedAt
	chirp.Updated_at = C.UpdatedAt
	chirp.User_id = C.Userid
	chirp.Id = C.ID

	RespondWithJson(w, http.StatusCreated, chirp)
}

func ValidateChirp(w http.ResponseWriter, r *http.Request, chirp *Chirp) error {

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(chirp)

	defer r.Body.Close()

	if err != nil {
		RespondWithErr(w, http.StatusBadRequest, "Something went wrong")
		return err
	}

	if len(chirp.Body) > 140 {
		RespondWithErr(w, http.StatusBadRequest, "Chirp is too long")
		return errors.New("chirp too long")
	}

	ReplaceBadword(&chirp.Body)
	return nil
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
	if cfg.Platform != "dev" {
		RespondWithErr(w, http.StatusForbidden, "you dont have the permissoin to access")
		return
	}
	w.Header().Add("Content-Type", "text/plain; charset= utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.FileServerHits.Swap(0)
	err := cfg.DB.DeleteAllUser(r.Context())
	if err != nil {
		RespondWithJson(w, http.StatusInternalServerError, "unable to delete all users")
		return
	}
	w.Write([]byte(fmt.Sprintf("Deleted ALl Users\nHits: %d", cfg.FileServerHits.Load())))
}
