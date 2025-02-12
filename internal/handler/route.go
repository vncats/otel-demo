package handler

import (
	"net/http"
)

func initRoutes(h *Handler) []*httpkit.RouteHandler {
	return []*httpkit.RouteHandler{
		{
			Route: &httpkit.Route{
				Name:         "get_movies",
				Method:       http.MethodGet,
				Path:         "/movies",
				AuthRequired: false,
			},
			Handle: h.GetMovies,
		},
		{
			Route: &httpkit.Route{
				Name:         "rate_movie",
				Method:       http.MethodGet,
				Path:         "/movies/{id}/ratings/{score}",
				AuthRequired: false,
			},
			Handle: h.RateMovie,
		},
	}
}
