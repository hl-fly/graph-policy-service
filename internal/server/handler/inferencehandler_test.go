package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

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

func BenchmarkExecuteInference_max_rps(b *testing.B) {
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
			"score": 750,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	latencies := make([]int64, 0, b.N)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		start := time.Now()
		handler.ExecuteInference(w, req)
		latencies = append(latencies, time.Since(start).Milliseconds())

		require.Equal(b, http.StatusOK, w.Code)
	}

	require.NotEmpty(b, latencies)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p90Index := (90 * len(latencies)) / 100
	p90Latency := latencies[p90Index]

	require.Less(b, p90Latency, int64(30), "P90 latency should be below 30ms, got %dms", p90Latency)
	b.ReportMetric(float64(p90Latency), "ms")
}

func BenchmarkExecuteInference_50RPS_P90Under30ms(b *testing.B) {
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
			"score": 750,
		},
	}
	bodyBytes, _ := json.Marshal(requestBody)

	latencies := make([]int64, 0, b.N)

	interval := time.Duration(1000/50) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		<-ticker.C

		req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		start := time.Now()
		handler.ExecuteInference(w, req)
		elapsed := time.Since(start)
		latencies = append(latencies, elapsed.Nanoseconds())

		require.Equal(b, http.StatusOK, w.Code)
	}

	b.StopTimer()

	require.NotEmpty(b, latencies)
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p90Index := (90 * len(latencies)) / 100
	p90LatencyNs := latencies[p90Index]
	p90LatencyMs := float64(p90LatencyNs) / 1_000_000.0

	p99Index := (99 * len(latencies)) / 100
	p99LatencyNs := latencies[p99Index]
	p99LatencyMs := float64(p99LatencyNs) / 1_000_000.0

	totalLatency := int64(0)
	for _, lat := range latencies {
		totalLatency += lat
	}
	avgLatencyMs := float64(totalLatency) / float64(len(latencies)) / 1_000_000.0

	require.Less(b, p90LatencyMs, float64(30), "P90 latency should be below 30ms, got %.2fms", p90LatencyMs)

	b.ReportMetric(p90LatencyMs, "P90_ms")
	b.ReportMetric(p99LatencyMs, "P99_ms")
	b.ReportMetric(avgLatencyMs, "avg_ms")
	b.ReportMetric(float64(len(latencies)), "total_requests")
}

func BenchmarkExecuteInference_50RPS_RandomRequests_WithCache(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	policyCache, _ := lru.New[string, *entity.Policy](1000)
	inferenceService := inferenceservice.NewInferenceService(logger)
	handler := NewInferenceHandler(logger, policyCache, inferenceService)

	policies := []string{
		`digraph Policy1 { 
			start [result=""]; 
			approved [result="approved=true,segment=prime"]; 
			rejected [result="approved=false"]; 
			review [result="approved=false,segment=manual"]; 
			start -> approved [cond="age>=18 && score>700"]; 
			start -> review [cond="age>=18 && score<=700"]; 
			start -> rejected [cond="age<18"]; 
		}`,
		`digraph Policy2 { 
			start [result=""]; 
			approved [result="approved=true"]; 
			rejected [result="approved=false"]; 
			start -> approved [cond="score>750"]; 
			start -> rejected [cond="score<=750"]; 
		}`,
		`digraph Policy3 { 
			start [result=""]; 
			approved [result="approved=true,segment=vip"]; 
			rejected [result="approved=false,segment=standard"]; 
			start -> approved [cond="age>=21 && balance>5000"]; 
			start -> rejected [cond="true"]; 
		}`,
	}

	inputs := []map[string]interface{}{
		{"age": 25, "score": 750, "balance": 10000},
		{"age": 30, "score": 800, "balance": 20000},
		{"age": 20, "score": 650, "balance": 5000},
		{"age": 35, "score": 850, "balance": 50000},
		{"age": 22, "score": 700, "balance": 8000},
	}

	latenciesHit := make([]int64, 0)
	latenciesMiss := make([]int64, 0)

	interval := time.Duration(1000/50) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		<-ticker.C

		policyIdx := i % len(policies)
		inputIdx := i % len(inputs)
		requestBody := map[string]interface{}{
			"policy_dot": policies[policyIdx],
			"input":      inputs[inputIdx],
		}
		bodyBytes, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/inference", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		cacheKey := policies[policyIdx]
		_, cacheHit := policyCache.Get(cacheKey)

		start := time.Now()
		handler.ExecuteInference(w, req)
		elapsed := time.Since(start)
		latencyNs := elapsed.Nanoseconds()

		if cacheHit {
			latenciesHit = append(latenciesHit, latencyNs)
		} else {
			latenciesMiss = append(latenciesMiss, latencyNs)
		}

		require.Equal(b, http.StatusOK, w.Code)
	}

	b.StopTimer()

	allLatencies := append(latenciesHit, latenciesMiss...)
	require.NotEmpty(b, allLatencies)
	sort.Slice(allLatencies, func(i, j int) bool { return allLatencies[i] < allLatencies[j] })

	p90Index := (90 * len(allLatencies)) / 100
	p90LatencyMs := float64(allLatencies[p90Index]) / 1_000_000.0

	p99Index := (99 * len(allLatencies)) / 100
	p99LatencyMs := float64(allLatencies[p99Index]) / 1_000_000.0

	totalLatency := int64(0)
	for _, lat := range allLatencies {
		totalLatency += lat
	}
	avgLatencyMs := float64(totalLatency) / float64(len(allLatencies)) / 1_000_000.0

	cacheHitRate := float64(0)
	if len(allLatencies) > 0 {
		cacheHitRate = float64(len(latenciesHit)) / float64(len(allLatencies)) * 100
	}

	var avgCacheHitMs, avgCacheMissMs float64
	if len(latenciesHit) > 0 {
		totalCacheHit := int64(0)
		for _, lat := range latenciesHit {
			totalCacheHit += lat
		}
		avgCacheHitMs = float64(totalCacheHit) / float64(len(latenciesHit)) / 1_000_000.0
	}

	if len(latenciesMiss) > 0 {
		totalCacheMiss := int64(0)
		for _, lat := range latenciesMiss {
			totalCacheMiss += lat
		}
		avgCacheMissMs = float64(totalCacheMiss) / float64(len(latenciesMiss)) / 1_000_000.0
	}

	require.Less(b, p90LatencyMs, float64(30), "P90 latency should be below 30ms, got %.2fms", p90LatencyMs)

	b.ReportMetric(p90LatencyMs, "P90_ms")
	b.ReportMetric(p99LatencyMs, "P99_ms")
	b.ReportMetric(avgLatencyMs, "avg_ms")
	b.ReportMetric(cacheHitRate, "cache_hit_rate_%")
	b.ReportMetric(avgCacheHitMs, "avg_cache_hit_ms")
	b.ReportMetric(avgCacheMissMs, "avg_cache_miss_ms")
	b.ReportMetric(float64(len(latenciesHit)), "cache_hits")
	b.ReportMetric(float64(len(latenciesMiss)), "cache_misses")
	b.ReportMetric(float64(len(allLatencies)), "total_requests")
}
