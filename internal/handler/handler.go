package handler

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"github.com/vncats/otel-demo/internal/cache"
	"github.com/vncats/otel-demo/internal/producer"
	"github.com/vncats/otel-demo/internal/store"
	"net/http"
	"strconv"
)

type GetMoviesResp struct {
	Movies []*store.Movie `json:"movies"`
}

type RateMovieReq struct {
	ID    int    `validate:"required,gt=0"`
	UID   string `validate:"required"`
	Score int    `validate:"required,gt=0,lte=5"`
}

type User struct {
	UID string `json:"uid"`
}

func NewHandler(st store.IStore, p producer.IProducer, cache cache.ICache) *Handler {
	return &Handler{
		store:     st,
		producer:  p,
		cache:     cache,
		validator: validator.New(),
	}
}

type Handler struct {
	store     store.IStore
	producer  producer.IProducer
	cache     cache.ICache
	validator *validator.Validate
}

func (h *Handler) GetMovies(ctx *httpkit.RequestContext) {
	movies, err := h.cache.GetMovies(ctx.GetContext())
	if err != nil {
		_ = ctx.SendError(err)
		return
	}

	resp := &GetMoviesResp{
		Movies: movies,
	}

	_ = ctx.SendSuccess("success", resp)
}

func (h *Handler) RateMovie(ctx *httpkit.RequestContext) {
	req := &RateMovieReq{
		UID:   getUserID(ctx),
		ID:    parseInt(ctx.Request.PathValue("id")),
		Score: parseInt(ctx.Request.PathValue("score")),
	}
	if err := h.validator.Struct(req); err != nil {
		_ = ctx.SendJSON(http.StatusBadRequest, netkit.VerdictInvalidParameters, "invalid param", container.Map{})
		return
	}

	rating := &store.Rating{
		MovieID: req.ID,
		UID:     req.UID,
		Score:   req.Score,
	}
	err := h.store.CreateRating(ctx.GetContext(), rating)
	if err != nil {
		_ = ctx.SendError(err)
		return
	}

	b, err := json.Marshal(rating)
	if err != nil {
		_ = ctx.SendError(err)
		return
	}

	err = h.producer.Produce("movie.rating-created", strconv.Itoa(req.ID), b)
	if err != nil {
		_ = ctx.SendError(err)
		return
	}

	_ = ctx.SendSuccess("rating created", rating)
}

func parseInt(str string) int {
	v, _ := strconv.Atoi(str)
	return v
}

func getUserID(ctx *httpkit.RequestContext) string {
	return ctx.Request.Header.Get("X-User-ID")
}
