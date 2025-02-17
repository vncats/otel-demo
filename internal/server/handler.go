package server

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/vncats/otel-demo/internal/cache"
	"github.com/vncats/otel-demo/internal/message"
	"github.com/vncats/otel-demo/internal/store"
	"github.com/vncats/otel-demo/internal/workflow"
	"github.com/vncats/otel-demo/pkg/otel/log"
	"github.com/vncats/otel-demo/pkg/prim"
	"go.temporal.io/sdk/client"
)

type IHandler interface {
	GetMovies(ctx *RequestContext)
	RateMovie(ctx *RequestContext)
	TrackUserAction(ctx context.Context, payload prim.Map)
}

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

var _ IHandler = (*Handler)(nil)

func NewHandler(
	st store.IStore,
	p message.IProducer,
	cache cache.ICache,
	tc client.Client,
) *Handler {
	return &Handler{
		store:     st,
		producer:  p,
		cache:     cache,
		wfClient:  tc,
		validator: validator.New(),
	}
}

type Handler struct {
	store     store.IStore
	producer  message.IProducer
	cache     cache.ICache
	wfClient  client.Client
	validator *validator.Validate
}

func (h *Handler) GetMovies(ctx *RequestContext) {
	movies, err := h.cache.GetMovies(ctx.Context())
	if err != nil {
		ctx.SendError()
		return
	}

	resp := &GetMoviesResp{
		Movies: movies,
	}

	log.Info(ctx.Context(), "get movies successfully")

	ctx.SendSuccess("get movies successfully", resp)
}

func (h *Handler) RateMovie(ctx *RequestContext) {
	req := &RateMovieReq{
		UID:   getUserID(ctx.Request),
		ID:    parseInt(ctx.Request.PathValue("id")),
		Score: parseInt(ctx.Request.PathValue("score")),
	}
	if err := h.validator.Struct(req); err != nil {
		log.Error(ctx.Context(), "invalid request", "error", err)
		ctx.SendBadRequest()
		return
	}

	rating := &store.Rating{
		MovieID: req.ID,
		UID:     req.UID,
		Score:   req.Score,
	}
	err := h.store.CreateRating(ctx.Context(), rating)
	if err != nil {
		ctx.SendError()
		return
	}

	_, err = h.producer.Produce(ctx.Context(), "private.movie.rating.created", strconv.Itoa(req.ID), rating)
	if err != nil {
		ctx.SendError()
		return
	}

	ctx.SendSuccess("rating created", rating)
}

func (h *Handler) TrackUserAction(ctx context.Context, payload prim.Map) {
	_, _ = h.wfClient.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        uuid.NewString(),
		TaskQueue: workflow.TaskQueue,
	}, workflow.TrackUserActionWorkflow, payload)
}

func parseInt(str string) int {
	v, _ := strconv.Atoi(str)
	return v
}

func getUserID(req *http.Request) string {
	return req.Header.Get("X-User-ID")
}

func getRequestID(req *http.Request) string {
	return req.Header.Get("X-Request-ID")
}
