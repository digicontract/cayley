package gizmo

import (
	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/jsonld"
	"github.com/piprate/json-gold/ld"
)

type jsonLd struct {
	s    *Session
	ld   *ld.JsonLdProcessor
	opts *ld.JsonLdOptions
	ctx  map[string]interface{}
}

func (r *jsonLd) Compact(input interface{}) map[string]interface{} {
	compact, err := r.ld.Compact(input, r.ctx, r.opts)
	if err != nil {
		panic(r.s.vm.ToValue(err))
	}

	return compact
}

func (r *jsonLd) Expand(input interface{}) []interface{} {
	expanded, err := r.ld.Expand(input, r.opts)
	if err != nil {
		panic(r.s.vm.ToValue(err))
	}
	return expanded
}

func (r *jsonLd) FromValue(value quad.Value) interface{} {
	return jsonld.FromValue(value)
}
