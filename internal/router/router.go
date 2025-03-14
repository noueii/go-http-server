package router

import (
	"net/http"
)

type Router struct {
	serveMux *http.ServeMux
}

func New() (*Router, error) {
	return &Router{
		serveMux: http.NewServeMux(),
	}, nil
}
