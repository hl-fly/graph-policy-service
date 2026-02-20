package model

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/awalterschulze/gographviz"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hector-leite/graph-policy-service/internal/domain/entity"
)

type InferenceRequest struct {
	PolicyDOT string                 `json:"policy_dot"`
	Input     map[string]interface{} `json:"input"`
}

func (req *InferenceRequest) GetPolicy(policyCache *lru.Cache[string, *entity.Policy]) (*entity.Policy, error) {
	hash := md5.Sum([]byte(req.PolicyDOT))
	key := hex.EncodeToString(hash[:])
	if policy, ok := policyCache.Get(key); ok {
		return policy, nil
	}

	// Essa bomba foi colocada aqui para não pedir que você troquem o padrão do payload, rezando para não ser muito custoso
	preprocessedDOT := strings.ReplaceAll(req.PolicyDOT, "result=", "label=")
	preprocessedDOT = strings.ReplaceAll(preprocessedDOT, "cond=", "xlabel=")

	graphAst, err := gographviz.ParseString(preprocessedDOT)
	if err != nil {
		return nil, err
	}

	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return nil, err
	}

	policy := &entity.Policy{
		Nodes: make(map[string]string),
		Edges: make(map[string][]entity.Edge),
	}

	for _, node := range graph.Nodes.Lookup {
		if res, ok := node.Attrs["label"]; ok {
			policy.Nodes[node.Name] = strings.Trim(res, `"`)
		}
	}

	for _, edge := range graph.Edges.Edges {
		condValue := ""
		if xlabel, ok := edge.Attrs["xlabel"]; ok {
			condValue = strings.Trim(xlabel, `"`)
		}

		var prog *vm.Program
		if condValue != "" {
			prog, err = expr.Compile(condValue)
			if err != nil {
				return nil, err
			}
		}

		policy.Edges[edge.Src] = append(policy.Edges[edge.Src], entity.Edge{
			To:      edge.Dst,
			Program: prog,
		})
	}

	policyCache.Add(key, policy)

	return policy, nil
}
