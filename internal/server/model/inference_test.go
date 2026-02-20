package model

import (
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPolicy_ValidDOT_AllNodesAndEdgesParsed(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              approved [result="approved=true,segment=prime"];
              rejected [result="approved=false"];
              review [result="approved=false,segment=manual"];
              start -> approved [cond="age>=18 && score>700"];
              start -> review [cond="age>=18 && score<=700"];
              start -> rejected [cond="age<18"];
          }`,
		Input: map[string]interface{}{
			"age":   25,
			"score": 720,
		},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, policy)
	assert.Len(t, policy.Nodes, 4)
	assert.Equal(t, "", policy.Nodes["start"])
	assert.Equal(t, "approved=true,segment=prime", policy.Nodes["approved"])
	assert.Equal(t, "approved=false", policy.Nodes["rejected"])
	assert.Equal(t, "approved=false,segment=manual", policy.Nodes["review"])
	assert.Len(t, policy.Edges["start"], 3)

	for _, edge := range policy.Edges["start"] {
		assert.NotNil(t, edge.Program, "Edge para '%s' deve ter programa compilado", edge.To)
	}

	destinations := make(map[string]bool)
	for _, edge := range policy.Edges["start"] {
		destinations[edge.To] = true
	}
	assert.True(t, destinations["approved"])
	assert.True(t, destinations["review"])
	assert.True(t, destinations["rejected"])
}

func TestGetPolicy_CacheHit_ReturnsCachedPolicy(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	dotString := `digraph Policy {
          start [result=""];
          approved [result="approved=true"];
          start -> approved [cond="score>700"];
      }`
	req1 := &InferenceRequest{
		PolicyDOT: dotString,
		Input:     map[string]interface{}{"score": 750},
	}
	req2 := &InferenceRequest{
		PolicyDOT: dotString,
		Input:     map[string]interface{}{"score": 800},
	}

	// Act
	policy1, err1 := req1.GetPolicy(cache)
	policy2, err2 := req2.GetPolicy(cache)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Same(t, policy1, policy2, "Deve retornar mesma instância do cache")
}

func TestGetPolicy_UnconditionalEdge_ProgramIsNil(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              default [result="status=default"];
              start -> default;
          }`,
		Input: map[string]interface{}{},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, policy)
	require.Len(t, policy.Edges["start"], 1)
	edge := policy.Edges["start"][0]
	assert.Equal(t, "default", edge.To)
	assert.Nil(t, edge.Program, "Edge sem condição deve ter Program=nil")
}

func TestGetPolicy_NodeWithoutResultAttribute_EmptyString(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start;
              approved [result="approved=true"];
              start -> approved [cond="score>700"];
          }`,
		Input: map[string]interface{}{"score": 750},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, policy)
	_, exists := policy.Nodes["start"]
	assert.False(t, exists, "Node sem atributo result não deve estar no map Nodes")
	assert.Equal(t, "approved=true", policy.Nodes["approved"])
}

func TestGetPolicy_InvalidDOTSyntax_ReturnsError(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy { invalid syntax @#$ }`,
		Input:     map[string]interface{}{"age": 25},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	assert.Error(t, err, "DOT inválido deve retornar erro")
	assert.Nil(t, policy)
}

func TestGetPolicy_InvalidConditionExpression_ReturnsError(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              approved [result="approved=true"];
              start -> approved [cond="invalid @#$ expression"];
          }`,
		Input: map[string]interface{}{},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	assert.Error(t, err, "Expressão inválida deve retornar erro ao compilar")
	assert.Nil(t, policy)
}

func TestGetPolicy_EmptyDOT_ReturnsError(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: "",
		Input:     map[string]interface{}{},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	assert.Error(t, err, "DOT vazio deve retornar erro")
	assert.Nil(t, policy)
}

func TestGetPolicy_ComplexCondition_SuccessfullyCompiled(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              vip [result="segment=vip"];
              start -> vip [cond="age>=21 && score>800 && income>5000"];
          }`,
		Input: map[string]interface{}{
			"age":    25,
			"score":  850,
			"income": 6000,
		},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, policy)
	require.Len(t, policy.Edges["start"], 1)
	edge := policy.Edges["start"][0]
	assert.Equal(t, "vip", edge.To)
	assert.NotNil(t, edge.Program, "Condição complexa deve compilar com sucesso")
}

func TestGetPolicy_QuotesInResultAttribute_TrimmedCorrectly(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              approved [result="approved=true,segment=prime"];
              start -> approved [cond="score>700"];
          }`,
		Input: map[string]interface{}{"score": 750},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "approved=true,segment=prime", policy.Nodes["approved"],
		"Aspas devem ser removidas do atributo result")
}

func TestGetPolicy_MultipleEdgesFromSameNode_AllParsed(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              node1 [result="result1=a"];
              node2 [result="result2=b"];
              node3 [result="result3=c"];
              start -> node1 [cond="x>10"];
              start -> node2 [cond="x>20"];
              start -> node3 [cond="x>30"];
          }`,
		Input: map[string]interface{}{"x": 25},
	}

	// Act
	policy, err := req.GetPolicy(cache)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, policy)
	edges := policy.Edges["start"]
	assert.Len(t, edges, 3, "Deve parsear todas as 3 edges do node 'start'")

	destinations := make(map[string]int)
	for _, edge := range edges {
		destinations[edge.To]++
		assert.NotNil(t, edge.Program)
	}
	assert.Equal(t, 1, destinations["node1"])
	assert.Equal(t, 1, destinations["node2"])
	assert.Equal(t, 1, destinations["node3"])
}

func TestGetPolicy_DifferentDOTs_DifferentCacheKeys(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req1 := &InferenceRequest{
		PolicyDOT: `digraph Policy { start [result=""]; approved [result="approved=true"]; start -> approved; }`,
		Input:     map[string]interface{}{},
	}
	req2 := &InferenceRequest{
		PolicyDOT: `digraph Policy { start [result=""]; rejected [result="approved=false"]; start -> rejected; }`,
		Input:     map[string]interface{}{},
	}

	// Act
	policy1, err1 := req1.GetPolicy(cache)
	policy2, err2 := req2.GetPolicy(cache)

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotSame(t, policy1, policy2, "DOTs diferentes devem gerar policies diferentes")
	assert.NotEqual(t, policy1.Nodes, policy2.Nodes)
}

func TestGetPolicy_CacheIsPopulated_AfterFirstCall(t *testing.T) {
	// Arrange
	cache, _ := lru.New[string, *entity.Policy](10)
	req := &InferenceRequest{
		PolicyDOT: `digraph Policy {
              start [result=""];
              end [result="done=true"];
              start -> end;
          }`,
		Input: map[string]interface{}{},
	}

	// Act
	initialCacheSize := cache.Len()
	_, err := req.GetPolicy(cache)
	finalCacheSize := cache.Len()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 0, initialCacheSize, "Cache deve estar vazio inicialmente")
	assert.Equal(t, 1, finalCacheSize, "Cache deve conter 1 policy após primeira chamada")
}
