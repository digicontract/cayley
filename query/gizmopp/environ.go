package gizmopp

// Builds a new Gizmo environment pointing at a session.

import (
	"fmt"
	"time"

	"github.com/dop251/goja"

	"github.com/cayleygraph/quad"
	"github.com/cayleygraph/quad/voc"

	"github.com/cayleygraph/cayley/graph/path"
)

// graphObject is a root graph object.
//
// Name: `graph`, Alias: `g`
//
// This is the only special object in the environment, generates the query objects.
// Under the hood, they're simple objects that get compiled to a Go iterator tree when executed.
// Methods starting with "New" are accessible in JavaScript with a capital letter (e.g. NewV becomes V)
type graphObject struct {
	s *Session
}

// AddNamespace associates prefix with a given IRI namespace.
func (g *graphObject) AddNamespace(pref, ns string) {
	g.s.ns.Register(voc.Namespace{Prefix: pref + ":", Full: ns})
}

// AddDefaultNamespaces register all default namespaces for automatic IRI resolution.
func (g *graphObject) AddDefaultNamespaces() {
	voc.CloneTo(&g.s.ns)
}

// LoadNamespaces loads all namespaces saved to graph.
func (g *graphObject) LoadNamespaces() error {
	return g.s.sch.LoadNamespaces(g.s.ctx, g.s.qs, &g.s.ns)
}

// Vertex starts a query path at the given vertex/vertices. No ids means "all vertices".
// Signature: ([nodeId],[nodeId]...)
//
// Arguments:
//
// * `nodeId` (Optional): A string or list of strings representing the starting vertices.
//
// Returns: Path object
func (g *graphObject) NewV(call goja.FunctionCall) goja.Value {
	qv, err := toQuadValues(exportArgs(call.Arguments))
	if err != nil {
		panic(g.s.vm.ToValue(err))
	}

	return g.s.vm.ToValue(&pathObject{
		s:    g.s,
		path: path.StartMorphism(qv...),
	})
}

// Morphism creates a morphism path object. Unqueryable on it's own, defines one end of the path.
// Saving these to variables with
//
//	// javascript
//	var shorterPath = graph.Morphism().out("foo").out("bar")
//
// is the common use case. See also: path.follow(), path.followR().
func (g *graphObject) NewM() goja.Value {
	return g.s.vm.ToValue(&pathObject{
		s:    g.s,
		path: path.StartMorphism(),
	})
}

// Emit adds data programmatically to the JSON result list. Can be any JSON type.
//
//	// javascript
//	g.emit({name:"bob"}) // push {"name":"bob"} as a result
func (g *graphObject) Emit(call goja.FunctionCall) goja.Value {
	value := call.Argument(0)
	if !goja.IsNull(value) && !goja.IsUndefined(value) {
		val := exportArgs([]goja.Value{value})[0]
		if val != nil {
			g.s.send(nil, &Result{Val: val})
		}
	}
	return goja.Null()
}

var defaultEnv = map[string]func(vm *goja.Runtime, call goja.FunctionCall) goja.Value{
	"type":  q1value(typeCheck),
	"iri":   s1string(func(s string) quad.Value { return quad.IRI(s) }),
	"bnode": s1string(func(s string) quad.Value { return quad.BNode(s) }),
	"str":   s1string(func(s string) quad.Value { return quad.String(s) }),
	"int":   s1int(func(s int64) quad.Value { return quad.Int(s) }),
	"float": s1float(func(s float64) quad.Value { return quad.Float(s) }),
	"bool":  s1bool(func(s bool) quad.Value { return quad.Bool(s) }),
	"time":  s1date(func(s time.Time) quad.Value { return quad.Time(s) }),
	"lang": s1string۰s2string(func(s, lang string) quad.Value {
		return quad.LangString{Value: quad.String(s), Lang: lang}
	}),
	"typed": s1string۰q1iri(func(s string, typ quad.IRI) quad.Value {
		return quad.TypedString{Value: quad.String(s), Type: typ}
	}),
}

func typeCheck(s quad.Value) string {
	switch s.(type) {
	case quad.IRI:
		return "iri"
	case quad.BNode:
		return "bnode"
	case quad.String:
		return "str"
	case quad.Int:
		return "int"
	case quad.Float:
		return "float"
	case quad.Bool:
		return "bool"
	case quad.Time:
		return "time"
	case quad.LangString:
		return "lang"
	case quad.TypedString:
		return "typed"
	default:
		return "unknown"
	}
}

func q1value(fn func(q1 quad.Value) string) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := exportArgs(call.Arguments)
		if len(args) != 1 {
			panic(vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
		}

		v, err := toQuadValue(args[0])
		if err != nil {
			panic(vm.ToValue(err))
		}

		return vm.ToValue(fn(v))
	}
}

func s1string(fn func(s string) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := toStrings(exportArgs(call.Arguments))
		if len(args) != 1 {
			panic(vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
		}
		return vm.ToValue(fn(args[0]))
	}
}

func s1int(fn func(s int64) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := toInts(exportArgs(call.Arguments))
		if len(args) != 1 {
			panic(vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
		}
		return vm.ToValue(fn(args[0]))
	}
}

func s1float(fn func(s float64) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := toFloats(exportArgs(call.Arguments))
		if len(args) != 1 {
			panic(vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
		}
		return vm.ToValue(fn(args[0]))
	}
}

func s1bool(fn func(s bool) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := toBools(exportArgs(call.Arguments))
		if len(args) != 1 {
			panic(vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
		}
		return vm.ToValue(fn(args[0]))
	}
}

func s1date(fn func(s time.Time) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args, err := toDates(exportArgs(call.Arguments))
		if err != nil {
			panic(vm.ToValue(err))
		}

		if len(args) != 1 {
			panic(vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
		}
		return vm.ToValue(fn(args[0]))
	}
}

func s1string۰s2string(fn func(s1, s2 string) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := toStrings(exportArgs(call.Arguments))
		if len(args) != 2 {
			panic(vm.ToValue(errArgCountNum{Expected: 2, Got: len(args)}))
		}
		return vm.ToValue(fn(args[0], args[1]))
	}
}

func s1string۰q1iri(fn func(s1 string, q1 quad.IRI) quad.Value) func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	return func(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
		args := toStrings(exportArgs(call.Arguments))
		if len(args) != 2 {
			panic(vm.ToValue(errArgCountNum{Expected: 2, Got: len(args)}))
		}

		v, err := toQuadValue(args[1])
		if err != nil {
			panic(vm.ToValue(err))
		}

		vt, ok := v.(quad.IRI)
		if !ok {
			panic(vm.ToValue(errType{Expected: quad.IRI(""), Got: v}))
		}

		return vm.ToValue(fn(args[0], vt))
	}
}

func unwrap(o interface{}) interface{} {
	switch v := o.(type) {
	case *pathObject:
		o = v.path
	case []interface{}:
		for i, val := range v {
			v[i] = unwrap(val)
		}
	case map[string]interface{}:
		for k, val := range v {
			v[k] = unwrap(val)
		}
	}
	return o
}

func exportArgs(args []goja.Value) []interface{} {
	if len(args) == 0 {
		return nil
	}
	out := make([]interface{}, 0, len(args))
	for _, a := range args {
		o := a.Export()
		out = append(out, unwrap(o))
	}
	return out
}

func toInt(o interface{}) (int, bool) {
	switch v := o.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

func toInts(objs []interface{}) []int64 {
	if len(objs) == 0 {
		return nil
	}
	var out = make([]int64, 0, len(objs))
	for _, o := range objs {
		switch v := o.(type) {
		case int:
			out = append(out, int64(v))
		case int64:
			out = append(out, v)
		case []int:
			for _, e := range v {
				out = append(out, int64(e))
			}
		case []int64:
			out = append(out, v...)
		case []interface{}:
			out = append(out, toInts(v)...)
		default:
			panic(fmt.Errorf("expected int, got: %T", o))
		}
	}
	return out
}

func toFloats(objs []interface{}) []float64 {
	if len(objs) == 0 {
		return nil
	}
	var out = make([]float64, 0, len(objs))
	for _, o := range objs {
		switch v := o.(type) {
		case float32:
			out = append(out, float64(v))
		case float64:
			out = append(out, v)
		case []float32:
			for _, e := range v {
				out = append(out, float64(e))
			}
		case []float64:
			out = append(out, v...)
		case []interface{}:
			out = append(out, toFloats(v)...)
		default:
			panic(fmt.Errorf("expected float, got: %T", o))
		}
	}
	return out
}

func toBool(o interface{}) bool {
	switch o.(type) {
	case bool:
		return true
	default:
		return false
	}
}

func toBools(objs []interface{}) []bool {
	if len(objs) == 0 {
		return nil
	}
	var out = make([]bool, 0, len(objs))
	for _, o := range objs {
		switch v := o.(type) {
		case bool:
			out = append(out, v)
		case []bool:
			out = append(out, v...)
		case []interface{}:
			out = append(out, toBools(v)...)
		default:
			panic(fmt.Errorf("expected bool, got: %T", o))
		}
	}
	return out
}

func toDates(objs []interface{}) ([]time.Time, error) {
	if len(objs) == 0 {
		return []time.Time{}, nil
	}
	var out = make([]time.Time, 0, len(objs))
	for _, o := range objs {
		switch v := o.(type) {
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		case []string:
			for _, e := range v {
				t, err := time.Parse(time.RFC3339, e)
				if err != nil {
					return nil, err
				}
				out = append(out, t)
			}
		case []interface{}:
			ts, err := toDates(v)
			if err != nil {
				return nil, err
			}
			out = append(out, ts...)
		default:
			panic(fmt.Errorf("expected date, got: %T", o))
		}
	}
	return out, nil
}

func toQuadValue(o interface{}) (quad.Value, error) {
	var qv quad.Value
	switch v := o.(type) {
	case quad.Value:
		qv = v
	case string:
		qv = quad.StringToValue(v)
	case bool:
		qv = quad.Bool(v)
	case int:
		qv = quad.Int(v)
	case int64:
		qv = quad.Int(v)
	case float64:
		if float64(int(v)) == v {
			qv = quad.Int(int64(v))
		} else {
			qv = quad.Float(v)
		}
	case time.Time:
		qv = quad.Time(v)
	default:
		return nil, errNotQuadValue{Val: o}
	}
	return qv, nil
}

func toQuadValues(objs []interface{}) ([]quad.Value, error) {
	if len(objs) == 0 {
		return nil, nil
	}
	vals := make([]quad.Value, 0, len(objs))
	for _, o := range objs {
		qv, err := toQuadValue(o)
		if err != nil {
			return nil, err
		}
		vals = append(vals, qv)
	}
	return vals, nil
}

func toStrings(objs []interface{}) []string {
	if len(objs) == 0 {
		return nil
	}
	var out = make([]string, 0, len(objs))
	for _, o := range objs {
		switch v := o.(type) {
		case string:
			out = append(out, v)
		case quad.Value:
			out = append(out, quad.StringOf(v))
		case []string:
			out = append(out, v...)
		case []interface{}:
			out = append(out, toStrings(v)...)
		default:
			panic(fmt.Errorf("expected string, got: %T", o))
		}
	}
	return out
}

func toVia(via []interface{}) []interface{} {
	if len(via) == 0 {
		return nil
	} else if len(via) == 1 {
		if via[0] == nil {
			return nil
		} else if v, ok := via[0].([]interface{}); ok {
			return toVia(v)
		} else if v, ok := via[0].([]string); ok {
			arr := make([]interface{}, 0, len(v))
			for _, s := range v {
				arr = append(arr, s)
			}
			return toVia(arr)
		}
	}
	for i := range via {
		if _, ok := via[i].(*path.Path); ok {
			// bypass
		} else if vp, ok := via[i].(*pathObject); ok {
			via[i] = vp.path
		} else if qv, err := toQuadValue(via[i]); err == nil {
			via[i] = qv
		} else {
			panic(fmt.Errorf("unsupported type: %T", via[i]))
		}
	}
	return via
}

func toViaData(objs []interface{}) (predicates []interface{}, tags []string, ok bool) {
	if len(objs) != 0 {
		predicates = toVia([]interface{}{objs[0]})
	}
	if len(objs) > 1 {
		tags = toStrings(objs[1:])
	}
	ok = true
	return
}

func toViaDepthData(objs []interface{}) (predicates []interface{}, maxDepth int, tags []string, ok bool) {
	if len(objs) != 0 {
		predicates = toVia([]interface{}{objs[0]})
	}
	if len(objs) > 1 {
		maxDepth, ok = toInt(objs[1])
		if ok {
			if len(objs) > 2 {
				tags = toStrings(objs[2:])
			}
		} else {
			tags = toStrings(objs[1:])
		}
	}
	ok = true
	return
}
