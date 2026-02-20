package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
	"github.com/hector-leite/graph-policy-service/internal/domain/service/inferenceservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteInference_AgeAbove18_ReturnsApprovedTrue(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	requestBody := map[string]interface{}{
		"policy_dot": `digraph { 
			start [result=""]; 
			ok [result="approved=true"]; 
			no [result="approved=false"]; 
			start -> ok [cond="age>=18"]; 
			start -> no [cond="age<18"]; 
		}`,
		"input": map[string]interface{}{
			"age": 20,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ExecuteInference(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	output, ok := response["output"].(map[string]interface{})
	require.True(t, ok, "output should be a map")
	assert.Equal(t, float64(20), output["age"])
	assert.Equal(t, true, output["approved"])
}

func TestExecuteInference_AgeBelow18_ReturnsApprovedFalse(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	requestBody := map[string]interface{}{
		"policy_dot": `digraph { 
			start [result=""]; 
			ok [result="approved=true"]; 
			no [result="approved=false"]; 
			start -> ok [cond="age>=18"]; 
			start -> no [cond="age<18"]; 
		}`,
		"input": map[string]interface{}{
			"age": 15,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ExecuteInference(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	output, ok := response["output"].(map[string]interface{})
	require.True(t, ok, "output should be a map")
	assert.Equal(t, float64(15), output["age"])
	assert.Equal(t, false, output["approved"])
}

func TestExecuteInference_ComplexPolicy_ReturnsApprovedAndSegment(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	requestBody := map[string]interface{}{
		"policy_dot": `digraph Policy { 
			start [result=""]; 
			approved [result="approved=true,segment=prime"]; 
			rejected [result="approved=false"]; 
			review [result="approved=false,segment=manual"]; 
			start -> approved [cond="age>=18 && score>700"]; 
			start -> review [cond="age>=18 && score<=700"]; 
			start -> rejected [cond="age<18"]; 
		}`,
		"input": map[string]interface{}{
			"age":   25,
			"score": 720,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ExecuteInference(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	output, ok := response["output"].(map[string]interface{})
	require.True(t, ok, "output should be a map")
	assert.Equal(t, float64(25), output["age"])
	assert.Equal(t, float64(720), output["score"])
	assert.Equal(t, true, output["approved"])
	assert.Equal(t, "prime", output["segment"])
}

func TestExecuteInference_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader([]byte(`{"invalid": json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ExecuteInference(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request body", response["result"])
	assert.NotEmpty(t, response["details"])
}

func TestExecuteInference_InvalidDOT_ReturnsBadRequest(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	requestBody := map[string]interface{}{
		"policy_dot": `invalid dot syntax {{{{`,
		"input": map[string]interface{}{
			"age": 20,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Act
	handler.ExecuteInference(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "invalid policy DOT", response["result"])
	assert.NotEmpty(t, response["details"])
}

func TestExecuteInference_CacheUsed_SecondCallFaster(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	requestBody := map[string]interface{}{
		"policy_dot": `digraph { start [result=""]; ok [result="approved=true"]; start -> ok [cond="age>=18"]; }`,
		"input": map[string]interface{}{
			"age": 20,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	// Act
	req1 := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	handler.ExecuteInference(w1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.ExecuteInference(w2, req2)

	// Assert
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, http.StatusOK, w2.Code)
	var response1, response2 map[string]interface{}
	json.NewDecoder(w1.Body).Decode(&response1)
	json.NewDecoder(w2.Body).Decode(&response2)

	assert.Equal(t, response1["output"], response2["output"])
}
