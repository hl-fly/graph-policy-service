package entity

import "github.com/expr-lang/expr/vm"

type Policy struct {
	Nodes map[string]string
	Edges map[string][]Edge
}

type Edge struct {
	To      string
	Program *vm.Program
}
