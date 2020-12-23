package presenter

import (
	"github.com/dop251/goja"
)

type Condition struct {
	Name       string       `json:"name"`
	Value      interface{}  `json:"value"`
	Operator   string       `json:"operator"`
	Conditions []*Condition `json:"conditions"`
	Runtime    *goja.Runtime
}

func NewCondition() *Condition {
	return &Condition{}
}

func (condition *Condition) InitRuntime() {
	condition.Runtime = goja.New()
	condition.Runtime.SetFieldNameMapper(goja.UncapFieldNameMapper())
}
