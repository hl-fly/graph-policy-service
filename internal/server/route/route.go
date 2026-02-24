package route

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hector-leite/graph-policy-service/internal/server/handler"
)

func New(inferencehandler handler.InferenceHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/infer", inferencehandler.ExecuteInference)

	return r
}
