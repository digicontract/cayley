package gizmo

import (
	"net/url"

	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/voc"
	"github.com/dop251/goja_nodejs/require"
)

func (s *Session) WithBase(base string) *Session {
	_ = s.vm.GlobalObject().Delete("base")
	context := s.ld.ctx["@context"].(map[string]interface{})
	context["@base"] = base
	s.ld.ctx = map[string]interface{}{
		"@context": context,
	}
	s.ld.opts.Base = base
	if s.ld.opts.Base != "" {
		s.vm.Set("base", func(frag string) quad.IRI {
			baseUrl, err := url.Parse(s.ld.opts.Base)
			if err != nil {
				panic(err)
			}

			relUrl, err := url.Parse(frag)
			if err != nil {
				panic(err)
			}

			return quad.IRI(baseUrl.ResolveReference(relUrl).String())
		})
	}
	return s
}

func (s *Session) WithNamespaces(ns *voc.Namespaces) *Session {
	ns.CloneTo(&s.ns)
	context := s.ld.ctx["@context"].(map[string]interface{})
	for _, entry := range s.ns.List() {
		context[entry.Prefix[:len(entry.Prefix)-1]] = entry.Full
	}
	s.ld.ctx = map[string]interface{}{
		"@context": context,
	}
	return s
}

func (s *Session) WithLogger(log Logger) *Session {
	s.log = log
	if s.log != nil {
		s.reg.RegisterNativeModule("console", NewConsole(s))
		s.vm.Set("console", require.Require(s.vm, "console"))
	}
	return s
}
