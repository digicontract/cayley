package gizmopp

import (
	"github.com/cayleygraph/quad"
	"github.com/dop251/goja"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/iterator"
	"github.com/cayleygraph/cayley/graph/shape"
)

var _ shape.ValueFilter = filterCallback{}

type filterCallback struct {
	sess *Session
	call goja.FunctionCall
	fn   goja.Callable
}

func (r filterCallback) BuildIterator(qs graph.QuadStore, it graph.Iterator) graph.Iterator {
	return iterator.NewValueFilter(qs, it, func(val quad.Value) (bool, error) {
		done, err := r.fn(r.call.This, r.sess.vm.ToValue(val))
		if err != nil {
			return false, err
		}

		if done == nil {
			return false, errType{Expected: true, Got: done}
		}

		return done.ToBoolean(), err
	})
}

type filterTypes struct {
	types []string
}

func (r filterTypes) BuildIterator(qs graph.QuadStore, it graph.Iterator) graph.Iterator {
	return iterator.NewValueFilter(qs, it, func(val quad.Value) (bool, error) {
		vt := typeCheck(val)
		for _, typ := range r.types {
			if vt == typ {
				return true, nil
			}
		}
		return false, errUnknownType{Val: vt}
	})
}

var _ shape.ValueMapper = mapperCallback{}

type mapperCallback struct {
	sess *Session
	call goja.FunctionCall
	fn   goja.Callable
}

func (r mapperCallback) BuildIterator(qs graph.QuadStore, it graph.Iterator) graph.Iterator {
	return iterator.NewValueMapper(qs, it, func(val quad.Value) (quad.Value, error) {
		done, err := r.fn(r.call.This, r.sess.vm.ToValue(val))
		if err != nil {
			return nil, err
		}

		if done == nil {
			return nil, errType{Expected: true, Got: done}
		}

		return toQuadValue(done.Export())
	})
}
