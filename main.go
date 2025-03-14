package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	"github.com/noueii/go-http-server/internal/app"

	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	godotenv.Load()

	app, err := app.New()

	if err != nil {
		fmt.Println("Could not initialize app. Exiting.")
		os.Exit(1)
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: app.Api.ServeMux,
	}

	server.ListenAndServe()

}

func handlerHealth(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("OK"))
}

func (cfg *apiConfig) handlerHits(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	body := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())

	rw.Write([]byte(body))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		cfg.fileserverHits.Add(int32(1))
		next.ServeHTTP(rw, req)
	})

}

func (cfg *apiConfig) handlerResetHits(rw http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	rw.WriteHeader(http.StatusOK)

}

func handlerChirp(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	type validJson struct {
		Valid bool `json:"valid"`
	}

	rw.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(req.Body)
	params := parameters{}

	err := decoder.Decode(&params)

	if err != nil {
		data, err := marshalError("Something went wrong")
		if err != nil {
			log.Printf("Could not marshall error")
			rw.WriteHeader(500)
			return
		}
		rw.WriteHeader(400)
		rw.Write([]byte(data))
		return
	}
	respBody := validJson{
		Valid: isValidChirp(params.Body),
	}

	if respBody.Valid {
		rw.WriteHeader(http.StatusOK)
	} else {
		data, err := marshalError("Chirp is too long")
		if err != nil {
			rw.WriteHeader(500)
			log.Printf("Error marshalling error 'chirp is too long'")
			return
		}
		rw.WriteHeader(400)
		rw.Write(data)
		return
	}

	data, err := json.Marshal(respBody)

	if err != nil {
		rw.WriteHeader(500)
		log.Printf("Error marshalling valid JSON")
		return
	}

	rw.Write([]byte(data))

}

func marshalError(message string) ([]byte, error) {
	type customJSON struct {
		Error string `json:"error"`
	}

	body := customJSON{
		Error: message,
	}

	data, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	return data, nil

}

func isValidChirp(text string) bool {
	if len(text) > 140 {
		return false
	}

	return true
}
