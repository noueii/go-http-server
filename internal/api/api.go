package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/noueii/go-http-server/internal/auth"
	"github.com/noueii/go-http-server/internal/database"
)

type config struct {
	fileserverHits atomic.Int32
	Db             *database.Queries
	Platform       string
	Secret         string
	PolkaKey       string
}

type API struct {
	Config     *config
	ServeMux   *http.ServeMux
	FileServer *http.Handler
}

func Load() (*API, error) {
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	dbConn, err := sql.Open("postgres", dbURL)

	if err != nil {
		return nil, err
	}

	secret := os.Getenv("SECRET")
	polkaKey := os.Getenv("POLKA_KEY")
	dbQueries := database.New(dbConn)

	cfg := &config{
		Db:       dbQueries,
		Platform: platform,
		Secret:   secret,
		PolkaKey: polkaKey,
	}

	fs, err := initFileServer()

	if err != nil {
		return nil, err
	}

	sm, err := initServeMux(cfg, fs)

	if err != nil {
		return nil, err
	}

	return &API{
		Config:     cfg,
		ServeMux:   sm,
		FileServer: fs,
	}, nil
}

func initFileServer() (*http.Handler, error) {

	fileServer := http.FileServer(http.Dir("."))
	return &fileServer, nil

}

func initServeMux(c *config, fs *http.Handler) (*http.ServeMux, error) {
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("GET /admin/metrics", c.handlerHits)
	serveMux.HandleFunc("POST /admin/reset", c.handlerReset)
	serveMux.Handle("/api/", http.StripPrefix("/api", *fs))
	serveMux.Handle("/app/", c.middlewareMetricsInc(http.StripPrefix("/app", *fs)))
	serveMux.HandleFunc("POST /api/chirps", c.handlerCreateChirp)
	serveMux.HandleFunc("GET /api/chirps", c.handlerGetAllChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpId}", c.handlerGetChirp)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpId}", c.handlerDeleteChirp)
	serveMux.HandleFunc("GET /api/healthz", handlerHealth)
	serveMux.HandleFunc("POST /api/users", c.handlerNewUser)
	serveMux.HandleFunc("POST /api/login", c.handlerLogin)
	serveMux.HandleFunc("POST /api/refresh", c.handlerRefreshToken)
	serveMux.HandleFunc("POST /api/revoke", c.handlerRevokeToken)
	serveMux.HandleFunc("PUT /api/users", c.handlerUpdateUser)
	serveMux.HandleFunc("POST /api/polka/webhooks", c.handlerUserUpgradeWebhook)
	return serveMux, nil

}

func handlerHealth(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("OK"))
}

func (cfg *config) handlerHits(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "text/html; charset=utf-8")
	rw.WriteHeader(http.StatusOK)
	body := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load())

	rw.Write([]byte(body))
}

func (cfg *config) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(int32(1))
		next.ServeHTTP(rw, req)
	})

}

func (cfg *config) handlerReset(rw http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Store(0)
	if cfg.Platform != "dev" {
		respondWithError(rw, 403, "Forbidden")
		return
	}

	err := cfg.Db.DeleteAllUsers(context.Background())
	if err != nil {
		respondWithError(rw, 500, "Could not delete users")
		return
	}

	rw.WriteHeader(200)
}

func (cfg *config) handlerNewUser(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}

	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(rw, 500, "Failed to decode request")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	user, err := cfg.Db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		respondWithError(rw, 500, "Failed to create user")
		return
	}

	type userJson struct {
		Id          string `json:"id"`
		CreatedAt   string `json:"created_at"`
		UpdatedAt   string `json:"updated_at"`
		Email       string `json:"email"`
		IsChirpyRed bool   `json:"is_chirpy_red"`
	}

	userBody := userJson{
		Id:          user.ID.String(),
		CreatedAt:   user.CreatedAt.String(),
		UpdatedAt:   user.UpdatedAt.String(),
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}

	respondWithJSON(rw, 201, userBody)
}

func (cfg *config) handlerLogin(rw http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}

	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(rw, 500, "Failed to decode request")
		return
	}

	user, err := cfg.Db.GetUserByEmail(context.Background(), params.Email)

	if err != nil {
		respondWithError(rw, 401, "Incorrect email or password")
		return
	}

	err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		respondWithError(rw, 401, "Incorrect email or password")
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.Secret)

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	refreshToken, err := auth.MakeRefreshToken()

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	_, err = cfg.Db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
	})

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	type userJson struct {
		Id           string `json:"id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		Email        string `json:"email"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
		IsChirpyRed  bool   `json:"is_chirpy_red"`
	}

	userBody := userJson{
		Id:           user.ID.String(),
		CreatedAt:    user.CreatedAt.String(),
		UpdatedAt:    user.UpdatedAt.String(),
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken,
		IsChirpyRed:  user.IsChirpyRed.Bool,
	}

	respondWithJSON(rw, 200, userBody)

}

func (cfg *config) handlerUpdateUser(rw http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)

	if err != nil {
		respondWithError(rw, 401, "Invalid token")
		return
	}

	userUUID, err := auth.ValidateJWT(token, cfg.Secret)

	if err != nil {
		respondWithError(rw, 401, "Invalid token")
		return
	}

	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	params := parameters{}

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&params)

	if err != nil {
		respondWithError(rw, 500, "Could not decode request body.")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)

	if err != nil {
		respondWithError(rw, 500, "Password could not be hashed")
		return
	}

	user, err := cfg.Db.UpdateUserEmailAndPasswordById(context.Background(), database.UpdateUserEmailAndPasswordByIdParams{
		ID:             userUUID,
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	type responseBody struct {
		Id          uuid.UUID `json:"id"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	response := responseBody{
		Id:          user.ID,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}

	respondWithJSON(rw, 200, response)
}

func (cfg *config) handlerUserUpgradeWebhook(rw http.ResponseWriter, req *http.Request) {
	key, err := auth.GetApiKey(req.Header)

	if err != nil {
		respondWithError(rw, 401, err.Error())
		return
	}

	if key != cfg.PolkaKey {
		respondWithError(rw, 401, "Unauthorized")
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserId string `json:"user_id"`
		} `json:"data"`
	}

	params := parameters{}

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&params)

	if err != nil {
		respondWithError(rw, 500, "Could not decode request body")
		return
	}

	if params.Event != "user.upgraded" {
		respondWithError(rw, 204, "Bad event")
		return
	}

	parsedId, err := uuid.Parse(params.Data.UserId)

	if err != nil {
		respondWithError(rw, 500, "Could not parse user id")
		return
	}

	_, err = cfg.Db.UpgradeUserById(context.Background(), parsedId)

	if err != nil {
		respondWithError(rw, 404, err.Error())
		return
	}

	rw.WriteHeader(204)

}

func (cfg *config) handlerRefreshToken(rw http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)

	if err != nil {
		respondWithError(rw, 401, "Missing token")
		return
	}

	dbRefreshToken, err := cfg.Db.GetRefreshTokenByToken(context.Background(), refreshToken)

	if err != nil || dbRefreshToken.RevokedAt.Valid || time.Until(dbRefreshToken.ExpiresAt) <= 0 {
		respondWithError(rw, 401, "Missing token")
		return
	}

	jwtToken, err := auth.MakeJWT(dbRefreshToken.UserID, cfg.Secret)

	if err != nil {
		respondWithError(rw, 500, "Could not generate JWT token")
		return
	}

	err = cfg.Db.RefreshTokenByToken(context.Background(), refreshToken)

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	type jsonBody struct {
		Token string `json:"token"`
	}

	respondWithJSON(rw, 200, jsonBody{
		Token: jwtToken,
	})
}

func (cfg *config) handlerRevokeToken(rw http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)

	if err != nil {
		respondWithError(rw, 401, "Token not found")
		return
	}

	err = cfg.Db.RevokeRefreshTokenByToken(context.Background(), token)
	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	rw.WriteHeader(204)
}

func cleanChirp(text string) (string, error) {
	profane := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	words := strings.Split(text, " ")

	for idx, word := range words {
		if slices.Contains(profane, strings.ToLower(word)) {
			words[idx] = "****"
		}
	}

	return strings.Join(words, " "), nil
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
		fmt.Println("Error marshalling error")
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

func respondWithError(rw http.ResponseWriter, code int, msg string) {
	rw.WriteHeader(code)
	rw.Header().Set("Content-Type", "application/json")
	data, err := marshalError(msg)
	if err != nil {
		return
	}

	rw.Write([]byte(data))
}

func marshallJSON(payload interface{}) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload")
		return nil, err
	}

	return data, nil
}

func respondWithJSON(rw http.ResponseWriter, code int, payload interface{}) {
	data, err := marshallJSON(payload)
	rw.Header().Set("Content-Type", "application/json")

	if err != nil {
		rw.WriteHeader(500)
		return
	}

	rw.WriteHeader(code)
	rw.Write([]byte(data))
}
