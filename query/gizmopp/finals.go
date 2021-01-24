package gizmopp

import (
	"github.com/dop251/goja"

	"github.com/cayleygraph/quad"

	"github.com/cayleygraph/cayley/graph/iterator"
)

const TopResultTag = "id"

// ToArray executes a query and returns the results at the end of the query path as an JS array.
//
// Example:
// 	// javascript
//	// bobFollowers contains an Array of followers of bob (alice, charlie, dani).
//	var bobFollowers = g.V("<bob>").In("<follows>").ToArray()
func (p *pathObject) ToArray(call goja.FunctionCall) goja.Value {
	return p.toArray(call, false)
}

func (p *pathObject) toArray(call goja.FunctionCall, withTags bool) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) > 1 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
	}
	limit := -1
	if len(args) > 0 {
		limit, _ = toInt(args[0])
	}
	it := p.buildIteratorTree()
	it = iterator.Tag(it, TopResultTag)
	var (
		array interface{}
		err   error
	)
	if !withTags {
		array, err = p.s.runIteratorToArrayNoTags(it, limit)
	} else {
		array, err = p.s.runIteratorToArray(it, limit)
	}
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}
	return p.s.vm.ToValue(array)
}

// ToValue is the same as ToArray, but limited to one result node.
func (p *pathObject) ToValue() (interface{}, error) {
	return p.toValue(false)
}

func (p *pathObject) toValue(withTags bool) (interface{}, error) {
	it := p.buildIteratorTree()
	it = iterator.Tag(it, TopResultTag)
	const limit = 1
	if !withTags {
		array, err := p.s.runIteratorToArrayNoTags(it, limit)
		if err != nil {
			return nil, err
		}
		if len(array) == 0 {
			return nil, nil
		}
		return array[0], nil
	} else {
		array, err := p.s.runIteratorToArray(it, limit)
		if err != nil {
			return nil, err
		}
		if len(array) == 0 {
			return nil, nil
		}
		return array[0], nil
	}
}

// ForEach calls callback(data) for each result.
// Signature: (callback: (val: quad.Value) => void, limit?: number): void
//
// Arguments:
//
// * `callback`: A javascript function of the form `function(data)`
// * `limit` (Optional): An integer value on the first `limit` paths to process.
//
// Example:
// 	// javascript
//	// Simulate query.All().All()
//	graph.V("<alice>").ForEach(function(d) { g.Emit(d) } )
func (p *pathObject) ForEach(call goja.FunctionCall) goja.Value {
	it := p.buildIteratorTree()
	if n := len(call.Arguments); n != 1 && n != 2 {
		panic(p.s.vm.ToValue(errArgCount{Got: len(call.Arguments)}))
	}
	callback := call.Argument(0)
	args := exportArgs(call.Arguments[1:])
	limit := -1
	if len(args) != 0 {
		limit, _ = toInt(args[0])
	}
	err := p.s.runIteratorWithCallback(it, callback, call, limit)
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}
	return goja.Undefined()
}

// Count returns a number of results and returns it as a value.
//
// Example:
//	// javascript
//	// Save count as a variable
//	var n = g.V().count()
//	// Send it as a query result
//	g.emit(n)
func (p *pathObject) Count() (int64, error) {
	it := p.buildIteratorTree()
	return p.s.countResults(it)
}

func quadValueToString(v quad.Value) string {
	if s, ok := v.(quad.String); ok {
		return string(s)
	}
	return quad.StringOf(v)
}
