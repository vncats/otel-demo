package api

import (
	"context"
	"encoding/json"
	"net/http"
)

type HttpResponse struct {
	Status  int         `json:"status"`
	Verdict string      `json:"verdict"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type RequestContext struct {
	Writer  http.ResponseWriter
	Request *http.Request
}

func (r *RequestContext) Context() context.Context {
	return r.Request.Context()
}

func (r *RequestContext) SendSuccess(message string, data any) {
	r.sendResponse(&HttpResponse{
		Status:  http.StatusOK,
		Verdict: "success",
		Message: message,
		Data:    data,
	})
}

func (r *RequestContext) SendError() {
	r.sendResponse(&HttpResponse{
		Status:  http.StatusInternalServerError,
		Verdict: "failure",
	})
}

func (r *RequestContext) SendBadRequest() {
	r.sendResponse(&HttpResponse{
		Status:  http.StatusBadRequest,
		Verdict: "bad_request",
	})
}

func (r *RequestContext) sendResponse(resp *HttpResponse) {
	r.Writer.Header().Set("Content-Type", "application/json")
	r.Writer.WriteHeader(resp.Status)
	_ = json.NewEncoder(r.Writer).Encode(resp)
}
