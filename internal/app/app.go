package app

import (
	"github.com/noueii/go-http-server/internal/api"
)

type App struct {
	Api *api.API
}

func New() (*App, error) {

	api, err := api.Load()

	if err != nil {
		return nil, err
	}
	return &App{
		Api: api,
	}, nil
}

func CreateUser() {}
