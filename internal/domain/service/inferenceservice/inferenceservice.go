package inferenceservice

import (
	"log/slog"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
)

type InferenceService interface {
	ExecuteInference(policy *entity.Policy, input map[string]interface{}) (map[string]interface{}, error)
}

type inferenceService struct {
	logger *slog.Logger
}

func NewInferenceService(logger *slog.Logger) InferenceService {
	return &inferenceService{
		logger: logger,
	}
}

func (s *inferenceService) ExecuteInference(policy *entity.Policy, input map[string]interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{}, len(input)+5)
	for k, v := range input {
		output[k] = v
	}

	currentNode := "start"
	for {
		if res, ok := policy.Nodes[currentNode]; ok && res != "" {
			s.applyResult(res, output)
		}

		edges, ok := policy.Edges[currentNode]
		if !ok {
			break
		}

		found := false
		for _, edge := range edges {
			if edge.Program == nil {
				currentNode = edge.To
				found = true
				break
			}

			result, err := expr.Run(edge.Program, output)
			if err != nil {
				s.logger.Error("Error evaluating edge program.", "error", err)
				return nil, err
			}

			if result.(bool) {
				currentNode = edge.To
				found = true
				break
			}
		}

		if !found {
			break
		}
	}

	return output, nil
}

func (s *inferenceService) applyResult(res string, output map[string]interface{}) {
	parts := strings.Split(res, ",")
	for _, p := range parts {
		kv := strings.Split(p, "=")
		if len(kv) == 2 {
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])

			switch v {
			case "true":
				output[k] = true
			case "false":
				output[k] = false
			default:
				output[k] = v
			}
		}
	}
}
