package inferenceservice

import (
	"log/slog"
	"os"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteInference_ShouldApproveWhenAgeAndScoreValid(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	approvedExpr, _ := expr.Compile("age >= 18 && score > 700")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":    "",
			"approved": "approved=true,segment=prime",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "approved", Program: approvedExpr},
			},
		},
	}
	input := map[string]interface{}{
		"age":   25,
		"score": 750,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 25, output["age"])
	assert.Equal(t, 750, output["score"])
	assert.Equal(t, true, output["approved"])
	assert.Equal(t, "prime", output["segment"])
}

func TestExecuteInference_ShouldRejectWhenAgeBelowMinimum(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	rejectedExpr, _ := expr.Compile("age < 18")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":    "",
			"rejected": "approved=false",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "rejected", Program: rejectedExpr},
			},
		},
	}
	input := map[string]interface{}{
		"age":   17,
		"score": 800,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 17, output["age"])
	assert.Equal(t, 800, output["score"])
	assert.Equal(t, false, output["approved"])
}

func TestExecuteInference_ShouldSendToReviewWhenScoreLow(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	reviewExpr, _ := expr.Compile("age >= 18 && score <= 700")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":  "",
			"review": "approved=false,segment=manual",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "review", Program: reviewExpr},
			},
		},
	}
	input := map[string]interface{}{
		"age":   30,
		"score": 650,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 30, output["age"])
	assert.Equal(t, 650, output["score"])
	assert.Equal(t, false, output["approved"])
	assert.Equal(t, "manual", output["segment"])
}

func TestExecuteInference_ShouldTakeFirstMatchingEdge(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	expr1, _ := expr.Compile("score > 800")
	expr2, _ := expr.Compile("score > 600")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":    "",
			"premium":  "tier=premium",
			"standard": "tier=standard",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "premium", Program: expr1},
				{To: "standard", Program: expr2},
			},
		},
	}
	input := map[string]interface{}{
		"score": 750,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 750, output["score"])
	assert.Equal(t, "standard", output["tier"])
}

func TestExecuteInference_ShouldFollowUnconditionalEdge(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":   "",
			"default": "status=processed",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "default", Program: nil},
			},
		},
	}
	input := map[string]interface{}{
		"user": "john",
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "john", output["user"])
	assert.Equal(t, "processed", output["status"])
}

func TestExecuteInference_ShouldStopWhenNoEdgesFound(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start": "initialized=true",
		},
		Edges: map[string][]entity.Edge{},
	}
	input := map[string]interface{}{
		"data": "test",
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "test", output["data"])
	assert.Equal(t, true, output["initialized"])
}

func TestExecuteInference_ShouldStopWhenNoEdgeMatches(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	expr1, _ := expr.Compile("score > 900")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":   "processed=true",
			"premium": "tier=premium",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "premium", Program: expr1},
			},
		},
	}
	input := map[string]interface{}{
		"score": 500,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 500, output["score"])
	assert.Equal(t, true, output["processed"])
	assert.Nil(t, output["tier"])
}

func TestExecuteInference_ShouldReturnErrorWhenExpressionFails(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	invalidExpr, _ := expr.Compile("nonExistentField > 100")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start": "",
			"end":   "done=true",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "end", Program: invalidExpr},
			},
		},
	}
	input := map[string]interface{}{
		"age": 25,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, output)
}

func TestExecuteInference_ShouldChainMultipleNodes(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewInferenceService(logger)
	expr1, _ := expr.Compile("age >= 18")
	expr2, _ := expr.Compile("score > 700")
	policy := &entity.Policy{
		Nodes: map[string]string{
			"start":      "",
			"checkScore": "ageValid=true",
			"approved":   "approved=true",
		},
		Edges: map[string][]entity.Edge{
			"start": {
				{To: "checkScore", Program: expr1},
			},
			"checkScore": {
				{To: "approved", Program: expr2},
			},
		},
	}
	input := map[string]interface{}{
		"age":   25,
		"score": 750,
	}

	// Act
	output, err := service.ExecuteInference(policy, input)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 25, output["age"])
	assert.Equal(t, 750, output["score"])
	assert.Equal(t, true, output["ageValid"])
	assert.Equal(t, true, output["approved"])
}

func TestApplyResult_ShouldApplyBooleanTrue(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("approved=true", output)

	// Assert
	assert.Equal(t, true, output["approved"])
	assert.Len(t, output, 1)
}

func TestApplyResult_ShouldApplyBooleanFalse(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("rejected=false", output)

	// Assert
	assert.Equal(t, false, output["rejected"])
	assert.Len(t, output, 1)
}

func TestApplyResult_ShouldApplyStringValue(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("segment=premium", output)

	// Assert
	assert.Equal(t, "premium", output["segment"])
	assert.Len(t, output, 1)
}

func TestApplyResult_ShouldApplyMultipleKeyValuePairs(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("approved=true,segment=prime,status=active", output)

	// Assert
	assert.Equal(t, true, output["approved"])
	assert.Equal(t, "prime", output["segment"])
	assert.Equal(t, "active", output["status"])
	assert.Len(t, output, 3)
}

func TestApplyResult_ShouldHandleEmptyString(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("", output)

	// Assert
	assert.Len(t, output, 0)
}

func TestApplyResult_ShouldTrimWhitespace(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult(" approved = true , segment = prime ", output)

	// Assert
	assert.Equal(t, true, output["approved"])
	assert.Equal(t, "prime", output["segment"])
	assert.Len(t, output, 2)
}

func TestApplyResult_ShouldIgnoreOddNumberOfValues(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("approved=true,segment", output)

	// Assert
	assert.Equal(t, true, output["approved"])
	assert.Len(t, output, 1) // "segment" without = is ignored
}

func TestApplyResult_ShouldApplyMixedTypes(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("approved=true,rejected=false,tier=gold,count=42", output)

	// Assert
	assert.Equal(t, true, output["approved"])
	assert.Equal(t, false, output["rejected"])
	assert.Equal(t, "gold", output["tier"])
	assert.Equal(t, "42", output["count"]) // Numbers stored as strings
	assert.Len(t, output, 4)
}

func TestApplyResult_ShouldOverwriteExistingKeys(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := map[string]interface{}{
		"status": "pending",
		"tier":   "bronze",
	}

	// Act
	service.applyResult("status=completed,tier=gold", output)

	// Assert
	assert.Equal(t, "completed", output["status"])
	assert.Equal(t, "gold", output["tier"])
	assert.Len(t, output, 2)
}

func TestApplyResult_ShouldHandleSingleComma(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult(",", output)

	// Assert
	assert.Len(t, output, 0) // Single comma is ignored
}

func TestApplyResult_ShouldHandleEmptyValues(t *testing.T) {
	// Arrange
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := &inferenceService{logger: logger}
	output := make(map[string]interface{})

	// Act
	service.applyResult("key1=,key2=value2", output)

	// Assert
	assert.Equal(t, "", output["key1"]) // Empty string value
	assert.Equal(t, "value2", output["key2"])
	assert.Len(t, output, 2)
}
