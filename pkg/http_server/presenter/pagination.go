package presenter

import (
	"github.com/dop251/goja"
)

type Pagination struct {
	Limit   interface{} `json:"limit"`
	Page    interface{} `json:"page"`
	Runtime *goja.Runtime
}

func New() *Pagination {
	return &Pagination{}
}

func (pagination *Pagination) InitRuntime() {
	pagination.Runtime = goja.New()
	pagination.Runtime.SetFieldNameMapper(goja.UncapFieldNameMapper())
}
