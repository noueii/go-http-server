package api

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/noueii/go-http-server/internal/auth"
	"github.com/noueii/go-http-server/internal/database"
)

type ChirpJSON struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

func (cfg *config) handlerCreateChirp(rw http.ResponseWriter, req *http.Request) {
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
		respondWithError(rw, 400, "Something went wrong")
		return
	}

	validBody := validJson{
		Valid: isValidChirp(params.Body),
	}

	if !validBody.Valid {
		respondWithError(rw, 400, "Chirp is too long")
		return
	}

	cleanedBody, _ := cleanChirp(params.Body)

	token, err := auth.GetBearerToken(req.Header)

	if err != nil {
		respondWithError(rw, 401, "Unauthorized")
		return
	}

	jwtUUID, err := auth.ValidateJWT(token, cfg.Secret)

	if err != nil {
		respondWithError(rw, 401, "Unauthorized")
		return
	}

	chirp, err := cfg.Db.CreateChirp(context.Background(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: jwtUUID,
	})

	if err != nil {
		respondWithError(rw, 500, err.Error())
	}

	respondWithJSON(rw, 201, ChirpJSON{
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID,
	})

}

func (cfg *config) handlerGetAllChirps(rw http.ResponseWriter, req *http.Request) {
	authorId := req.URL.Query().Get("author_id")

	chirps := make([]database.Chirp, 0)

	if authorId == "" {
		chirpsDB, err := cfg.Db.GetAllChirps(context.Background())

		if err != nil {
			respondWithError(rw, 500, err.Error())
			return
		}

		chirps = chirpsDB

	} else {
		authorUUID, err := uuid.Parse(authorId)

		if err != nil {
			respondWithError(rw, 500, err.Error())
			return
		}

		chirpsDB, err := cfg.Db.GetAllChirpsByAuthorId(context.Background(), authorUUID)

		if err != nil {
			respondWithError(rw, 500, err.Error())
			return
		}

		chirps = chirpsDB

	}

	response := make([]ChirpJSON, 0)

	for _, chirp := range chirps {
		response = append(response, ChirpJSON{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		})
	}

	sorting := req.URL.Query().Get("sort")

	if sorting == "desc" {
		slices.Reverse(response)
	}

	respondWithJSON(rw, 200, response)
}

func (cfg *config) handlerGetChirp(rw http.ResponseWriter, req *http.Request) {
	chirpId := req.PathValue("chirpId")
	if chirpId == "" {
		respondWithError(rw, 404, "Missing chirp id")
		return
	}

	id, err := uuid.Parse(chirpId)

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	chirp, err := cfg.Db.GetChirpById(context.Background(), id)

	if err != nil {
		respondWithError(rw, 404, err.Error())
		return
	}

	respondWithJSON(rw, 200, ChirpJSON{
		Id:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID,
	})

}

func (cfg *config) handlerDeleteChirp(rw http.ResponseWriter, req *http.Request) {
	jwtToken, err := auth.GetBearerToken(req.Header)

	if err != nil {
		respondWithError(rw, 401, "Unauthorized")
		return
	}

	chirpId := req.PathValue("chirpId")

	if chirpId == "" {
		respondWithError(rw, 404, "Chirp not found")
		return
	}

	chirpUUID, err := uuid.Parse(chirpId)

	if err != nil {
		respondWithError(rw, 500, "Invalid chirp")
		return
	}

	chirp, err := cfg.Db.GetChirpById(context.Background(), chirpUUID)

	if err != nil {
		respondWithError(rw, 404, "Chirp not found")
		return
	}

	userId, err := auth.ValidateJWT(jwtToken, cfg.Secret)

	if err != nil {
		respondWithError(rw, 403, "Unauthorized")
		return
	}

	if userId != chirp.UserID {
		respondWithError(rw, 403, "Unauthorized")
		return
	}

	err = cfg.Db.DeleteChirpById(context.Background(), chirpUUID)

	if err != nil {
		respondWithError(rw, 500, err.Error())
		return
	}

	rw.WriteHeader(204)
}
