package gizmo

import (
	"github.com/cayleygraph/quad/voc"
)

type namespaces struct {
	s *Session
}

func (r *namespaces) Register(prefix, full string) {
	r.s.ns.Register(voc.Namespace{Prefix: prefix, Full: full})
}

func (r *namespaces) List() map[string]string {
	result := make(map[string]string)
	for _, entry := range r.s.ns.List() {
		result[entry.Prefix] = entry.Full
	}
	return result
}