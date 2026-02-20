package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
	"github.com/hector-leite/graph-policy-service/internal/domain/service/inferenceservice"
	"github.com/hector-leite/graph-policy-service/internal/server/model"
)

type InferenceHandler struct {
	logger           *slog.Logger
	policyCache      *lru.Cache[string, *entity.Policy]
	inferenceService inferenceservice.InferenceService
}

func NewInferenceHandler(
	logger *slog.Logger,
	policyCache *lru.Cache[string, *entity.Policy],
	inferenceService inferenceservice.InferenceService,
) *InferenceHandler {
	return &InferenceHandler{
		logger:           logger,
		policyCache:      policyCache,
		inferenceService: inferenceService,
	}
}

func (h *InferenceHandler) ExecuteInference(w http.ResponseWriter, r *http.Request) {
	var req model.InferenceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Error decoding request body.", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result":  "invalid request body",
			"details": err.Error(),
		})
		return
	}

	policy, err := req.GetPolicy(h.policyCache)
	if err != nil {
		h.logger.Error("Error parsing policy DOT.", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result":  "invalid policy DOT",
			"details": err.Error(),
		})
		return
	}

	res, err := h.inferenceService.ExecuteInference(policy, req.Input)
	if err != nil {
		h.logger.Error("Error executing inference.", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result":  "inference execution error",
			"details": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"output": res,
	})
}
