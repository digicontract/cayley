package gizmo

// Adds special traversal functions to JS Gizmo objects. Most of these just
// build the chain of objects, and won't often need the session.

import (
	"regexp"

	"github.com/cayleygraph/quad"
	"github.com/dop251/goja"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/graph/iterator"
	"github.com/cayleygraph/cayley/graph/path"
	"github.com/cayleygraph/cayley/graph/shape"
)

// pathObject is a Path object in Gizmo.
//
// Both `.M()` and `.V()` create path objects, which provide the following
// traversal methods.
//
// For these examples, suppose we have the following graph:
//
//	+-------+                        +------+
//	| alice |-----                 ->| fred |<--
//	+-------+     \---->+-------+-/  +------+   \-+-------+
//	              ----->| #bob# |       |         |*emily*|
//	+---------+--/  --->+-------+       |         +-------+
//	| charlie |    /                    v
//	+---------+   /                  +--------+
//	  \---    +--------+             |*#greg#*|
//	      \-->| #dani# |------------>+--------+
//	          +--------+
//
// Where every link is a `<follows>` relationship, and the nodes with an extra
// `#` in the name have an extra `<status>` link. As in,
//
//	<dani> -- <status> --> "cool_person"
//
// Perhaps these are the influencers in our community. So too are extra `*`s in
// the name -- these are our smart people, according to the `<smart_graph>`
// label, eg, the quad:
//
//	<greg> <status> "smart_person" <smart_graph> .
type pathObject struct {
	s    *Session
	path *path.Path
}

func (p *pathObject) new(np *path.Path) *pathObject {
	return &pathObject{
		s:    p.s,
		path: np,
	}
}

func (p *pathObject) newVal(np *path.Path) goja.Value {
	return p.s.vm.ToValue(p.new(np))
}

func (p *pathObject) clonePath() *path.Path {
	np := p.path.Clone()
	// most likely path will be continued, so we'll put non-capped stack slice
	// into new path object instead of preserving it in an old one
	p.path, np = np, p.path
	return np
}

func (p *pathObject) buildIteratorTree() graph.Iterator {
	if p.path == nil {
		return iterator.NewNull()
	}
	return p.path.BuildIteratorOn(p.s.qs)
}

// Is filters all paths to ones which, at this point, result are on the given node.
// Signature: is(...node: Value[]): Path;
//
// Arguments:
//
// * `node`: A quad.Value, string, boolean, number, or unknown.
//
// Example:
//	// javascript
//	// Starting from all nodes in the graph, find the paths that follow bob.
//	// Results in three paths for bob (from alice, charlie and dani).all()
//	g.V().out("<follows>").is("<bob>").all()
func (p *pathObject) Is(call goja.FunctionCall) goja.Value {
	args, err := toQuadValues(exportArgs(call.Arguments))
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}
	np := p.clonePath().Is(args...)
	return p.newVal(np)
}

// In is inverse of Out.
// Starting with the nodes in `path` on the object, follow the quads with
// predicates defined by `predicatePath` to their subjects.
// Signature: in(preds: MaybeArray<Path | Value>, ...tags: Tag[]): Path
//
// Arguments:
//
// * `predicatePath` (Optional): One of:
//   * null or undefined: All predicates pointing into this node
//   * a string: The predicate name to follow into this node
//   * a list of strings: The predicates to follow into this node
//   * a query path object: The target of which is a set of predicates to follow.
// * `tags` (Optional): One of:
//   * null or undefined: No tags
//   * a string: A single tag to add the predicate used to the output set.
//   * a list of strings: Multiple tags to use as keys to save the predicate
//   	used to the output set.
//
// Example:
//
//	// javascript
//	// Find the cool people, bob, dani and greg
//	g.V("cool_person").in("<status>").all()
//	// Find who follows bob, in this case, alice, charlie, and dani
//	g.V("<bob>").in("<follows>").all()
//	// Find who follows the people emily follows, namely, bob and emily
//	g.V("<emily>").out("<follows>").in("<follows>").all()
func (p *pathObject) In(call goja.FunctionCall) goja.Value {
	return p.inout(call, true)
}

// Out is the work-a-day way to get between nodes, in the forward direction.
// Starting with the nodes in `path` on the subject, follow the quads with
// predicates defined by `predicatePath` to their objects.
// Signature: out(preds: MaybeArray<Path | Value>, ...tags: Tag[]): Path
//
// Arguments:
//
// * `predicatePath` (Optional): One of:
//   * null or undefined: All predicates pointing out from this node
//   * a string: The predicate name to follow out from this node
//   * a list of strings: The predicates to follow out from this node
//   * a query path object: The target of which is a set of predicates to follow.
// * `tags` (Optional): One of:
//   * null or undefined: No tags
//   * a string: A single tag to add the predicate used to the output set.
//   * a list of strings: Multiple tags to use as keys to save the predicate
//   	used to the output set.
//
// Example:
//
//	// javascript
//	// The working set of this is bob and dani
//	g.V("<charlie>").out("<follows>").all()
//	// The working set of this is fred, as alice follows bob and bob follows fred.
//	g.V("<alice>").out("<follows>").out("<follows>").all()
//	// Finds all things dani points at. Result is bob, greg and cool_person
//	g.V("<dani>").out().all()
//	// Finds all things dani points at on the status linkage.
//	// Result is bob, greg and cool_person
//	g.V("<dani>").out(["<follows>", "<status>"]).all()
//	// Finds all things dani points at on the status linkage, given from a separate query path.
//	// Result is {"id": "cool_person", "pred": "<status>"}
//	g.V("<dani>").out(g.V("<status>"), "pred").all()
func (p *pathObject) Out(call goja.FunctionCall) goja.Value {
	return p.inout(call, false)
}

func (p *pathObject) inout(call goja.FunctionCall, in bool) goja.Value {
	preds, _, ok := toViaData(exportArgs(call.Arguments))
	if !ok {
		panic(p.s.vm.ToValue(errNoVia))
	}
	np := p.clonePath()
	if in {
		np = np.In(preds...)
	} else {
		np = np.Out(preds...)
	}
	return p.newVal(np)
}

// Both follow the predicate in either direction. Same as Out or In.
// Signature: both(preds: MaybeArray<Path | Value>, ...tags: Tag[]): Path
//
// Example:
//	// javascript
//	// Find all followers/followees of fred. Returns bob, emily and greg
//	g.V("<fred>").both("<follows>").all()
func (p *pathObject) Both(call goja.FunctionCall) goja.Value {
	preds, _, ok := toViaData(exportArgs(call.Arguments))
	if !ok {
		panic(p.s.vm.ToValue(errNoVia))
	}
	np := p.clonePath().Both(preds...)
	return p.newVal(np)
}

// Follow is the way to use a path prepared with Morphism. Applies the path
// chain on the morphism object to the current path.
//
// Starts as if at the g.M() and follows through the morphism path.
//
// Example:
// 	// javascript:
//	var friendOfFriend = g.Morphism().Out("<follows>").Out("<follows>")
//	// Returns the followed people of who charlie follows -- a simplistic
//	//	"friend of my friend" and whether or not they have a "cool" status.
//	//	Potential for recommending followers abounds.
//	// Returns bob and greg
//	g.V("<charlie>").follow(friendOfFriend).has("<status>", "cool_person").all()
func (p *pathObject) Follow(path *pathObject) *pathObject {
	return p.follow(path, false)
}

// FollowReverse is the same as Follow but follows the chain in the reverse
// direction. Flips "In" and "Out" where appropriate, the net result being a
// virtual predicate followed in the reverse direction.
//
// Starts at the end of the morphism and follows it backwards (with appropriate
// flipped directions) to the g.M() location.
//
// Example:
// 	// javascript:
//	var friendOfFriend = g.Morphism().Out("<follows>").Out("<follows>")
//	// Returns the third-tier of influencers -- people who follow people who
//	//	follow the cool people.
//	// Returns charlie (from bob), charlie (from greg), bob and emily
//	g.V().has("<status>", "cool_person").followR(friendOfFriend).all()
func (p *pathObject) FollowReverse(path *pathObject) *pathObject {
	return p.follow(path, true)
}

func (p *pathObject) follow(ep *pathObject, rev bool) *pathObject {
	if ep == nil {
		return p
	}
	np := p.clonePath()
	if rev {
		np = np.FollowReverse(ep.path)
	} else {
		np = np.Follow(ep.path)
	}
	return p.new(np)
}

// FollowRecursive is the same as Follow but follows the chain recursively.
//
// Starts as if at the g.M() and follows through the morphism path multiple
// times, returning all nodes encountered.
//
// Example:
// 	// javascript:
//	var friend = g.Morphism().out("<follows>")
//	// Returns all people in Charlie's network.
//	// Returns bob and dani (from charlie), fred (from bob) and greg (from dani).
//	g.V("<charlie>").followRecursive(friend).all()
func (p *pathObject) FollowRecursive(call goja.FunctionCall) goja.Value {
	preds, maxDepth, _, ok := toViaDepthData(exportArgs(call.Arguments))
	if !ok || len(preds) == 0 {
		panic(p.s.vm.ToValue(errNoVia))
	} else if len(preds) != 1 {
		panic(p.s.vm.ToValue("expected one predicate or path for recursive follow"))
	}
	np := p.clonePath()
	np = np.FollowRecursive(preds[0], maxDepth, []string{})
	return p.newVal(np)
}

// Intersect filters all paths by the result of another query path.
//
// This is essentially a join where, at the stage of each path, a node is shared.
// Example:
// 	// javascript
//	var cFollows = g.V("<charlie>").Out("<follows>")
//	var dFollows = g.V("<dani>").Out("<follows>")
//	// People followed by both charlie (bob and dani) and dani (bob and greg) -- returns bob.
//	cFollows.Intersect(dFollows).All()
//	// Equivalently, g.V("<charlie>").Out("<follows>").And(g.V("<dani>").Out("<follows>")).All()
func (p *pathObject) Intersect(call goja.FunctionCall) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) != 1 && len(args) != 2 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
	}

	via, ok := args[0].(*path.Path)
	if !ok {
		panic(p.s.vm.ToValue(errType{Expected: &pathObject{}, Got: via}))
	}

	follow := false
	if len(args) > 1 {
		follow = toBool(args[1])
	}

	if via == nil {
		return p.s.vm.ToValue(p)
	}

	np := p.clonePath().And(via, follow)
	return p.newVal(np)
}

// Union returns the combined paths of the two queries.
//
// Notice that it's per-path, not per-node. Once again, if multiple paths reach
// the same destination, they might have had different ways of getting there
// (and different tags). See also: `path.Tag()`
//
// Example:
// 	// javascript
//	var cFollows = g.V("<charlie>").Out("<follows>")
//	var dFollows = g.V("<dani>").Out("<follows>")
//	// People followed by both charlie (bob and dani) and dani (bob and greg)
//	//	-- returns bob (from charlie), dani, bob (from dani), and greg.
//	cFollows.Union(dFollows).All()
func (p *pathObject) Union(call goja.FunctionCall) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) != 1 && len(args) != 2 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
	}

	via, ok := args[0].(*path.Path)
	if !ok {
		panic(p.s.vm.ToValue(errType{Expected: &pathObject{}, Got: via}))
	}

	follow := false
	if len(args) > 1 {
		follow = toBool(args[1])
	}

	if via == nil {
		return p.s.vm.ToValue(p)
	}

	np := p.clonePath().Or(via, follow)
	return p.newVal(np)
}

// Has filters all paths which are, at this point, on the subject for the given
// predicate and object, but do not follow the path, merely filter the possible
// paths.
//
// Usually useful for starting with all nodes, or limiting to a subset
// depending on some predicate/value pair.
//
// Signature: (predicate: Path | Value, ...objs: Value[]): Path
//
// Arguments:
//
// * `predicate`: A string for a predicate node.
// * `object`: A string for a object node or a set of filters to find it.
//
// Example:
// 	// javascript
//	// Start from all nodes that follow bob -- results in alice, charlie and dani
//	g.V().has("<follows>", "<bob>").all()
//	// People charlie follows who then follow fred. Results in bob.
//	g.V("<charlie>").Out("<follows>").has("<follows>", "<fred>").all()
//	// People with friends who have names sorting lower then "f".
//	g.V().has("<follows>", gt("<f>")).all()
func (p *pathObject) Has(call goja.FunctionCall) goja.Value {
	return p.has(call, false)
}

// HasReverse is the same as Has, but sets constraint in reverse direction.
func (p *pathObject) HasReverse(call goja.FunctionCall) goja.Value {
	return p.has(call, true)
}

func (p *pathObject) has(call goja.FunctionCall, rev bool) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) == 0 {
		panic(p.s.vm.ToValue(errArgCount{Got: len(args)}))
	}
	via := args[0]
	args = args[1:]
	if vp, ok := via.(*pathObject); ok {
		via = vp.path
	} else {
		var err error
		via, err = toQuadValue(via)
		if err != nil {
			panic(p.s.vm.ToValue(err))
		}
	}
	qv, err := toQuadValues(args)
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}
	np := p.clonePath()
	if rev {
		np = np.HasReverse(via, qv...)
	} else {
		np = np.Has(via, qv...)
	}
	return p.newVal(np)
}

// Except removes all paths which match query from current path.
//
// In a set-theoretic sense, this is (A - B). While `g.V().Except(path)` to
// achieve `U - B = !B` is supported, it's often very slow.
//
// Example:
// 	// javascript
//	var cFollows = g.V("<charlie>").Out("<follows>")
//	var dFollows = g.V("<dani>").Out("<follows>")
//	// People followed by both charlie (bob and dani) and dani (bob and greg)
//	//	-- returns bob.
//	cFollows.Except(dFollows).All()
// 	// The set (dani) -- what charlie follows that dani does not also follow.
//	// Equivalently, g.V("<charlie>").Out("<follows>").Except(g.V("<dani>").Out("<follows>")).All()
func (p *pathObject) Except(call goja.FunctionCall) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) != 1 && len(args) != 2 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
	}

	via, ok := args[0].(*path.Path)
	if !ok {
		panic(p.s.vm.ToValue(errType{Expected: &pathObject{}, Got: via}))
	}

	follow := false
	if len(args) > 1 {
		follow = toBool(args[1])
	}

	if via == nil {
		return p.s.vm.ToValue(p)
	}

	np := p.clonePath().Except(via, follow)
	return p.newVal(np)
}

// Unique removes duplicate values from the path.
func (p *pathObject) Unique() *pathObject {
	np := p.clonePath().Unique()
	return p.new(np)
}

// Labels gets the list of inbound and outbound quad labels
func (p *pathObject) Labels() *pathObject {
	np := p.clonePath().Labels()
	return p.new(np)
}

// InPredicates gets the list of predicates that are pointing in to a node.
//
// Example:
// 	// javascript
//	// bob only has "<follows>" predicates pointing inward
//	// returns "<follows>"
//	g.V("<bob>").InPredicates().All()
func (p *pathObject) InPredicates() *pathObject {
	np := p.clonePath().InPredicates()
	return p.new(np)
}

// OutPredicates gets the list of predicates that are pointing out from a node.
//
// Example:
// 	// javascript
//	// bob has "<follows>" and "<status>" edges pointing outwards
//	// returns "<follows>", "<status>"
//	g.V("<bob>").OutPredicates().All()
func (p *pathObject) OutPredicates() *pathObject {
	np := p.clonePath().OutPredicates()
	return p.new(np)
}

// LabelContext sets (or removes) the subgraph context to consider in the following traversals.
// Affects all In(), Out(), and Both() calls that follow it. The default LabelContext is null (all subgraphs).
// Signature: ([labelPath])
//
// Arguments:
//
// * `predicatePath` (Optional): One of:
//   * null or undefined: In future traversals, consider all edges, regardless of subgraph.
//   * a string: The name of the subgraph to restrict traversals to.
//   * a list of strings: A set of subgraphs to restrict traversals to.
//   * a query path object: The target of which is a set of subgraphs.
//
// Example:
// 	// javascript
//	// Find the status of people Dani follows
//	g.V("<dani>").out("<follows>").out("<status>").all()
//	// Find only the statuses provided by the smart_graph
//	g.V("<dani>").out("<follows>").labelContext("<smart_graph>").out("<status>").all()
//	// Find all people followed by people with statuses in the smart_graph.
//	g.V().labelContext("<smart_graph>").in("<status>").labelContext(null).in("<follows>").all()
func (p *pathObject) LabelContext(call goja.FunctionCall) goja.Value {
	labels, _, ok := toViaData(exportArgs(call.Arguments))
	if !ok {
		panic(p.s.vm.ToValue(errNoVia))
	}
	np := p.clonePath().LabelContext(labels...)
	return p.newVal(np)
}

// Filter applies constraints to a set of nodes. Can be used to filter values by range or match strings.
func (p *pathObject) Filter(call goja.FunctionCall) goja.Value {
	if n := len(call.Arguments); n != 1 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(call.Arguments)}))
	}

	fn, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		panic(p.s.vm.ToValue("expected callback function"))
	}

	np := p.clonePath().Filters(filterCallback{sess: p.s, call: call, fn: fn})
	return p.newVal(np)
}

// Regex applies constraints to a set of nodes. Can be used to filter values by range or match strings.
func (p *pathObject) Regex(call goja.FunctionCall) goja.Value {
	if n := len(call.Arguments); n != 1 && n != 2 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(call.Arguments)}))
	}

	args := exportArgs(call.Arguments)
	v, err := toQuadValue(args[0])
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}
	allowRefs := false
	if len(args) > 1 {
		b, ok := args[1].(bool)
		if !ok {
			panic(p.s.vm.ToValue(errType{Expected: true, Got: args[1]}))
		}
		allowRefs = b
	}
	switch vt := v.(type) {
	case quad.String:
		if allowRefs {
			v = quad.IRI(vt)
		}
	case quad.IRI:
		if !allowRefs {
			panic(p.s.vm.ToValue(errRegexpOnIRI))
		}
	case quad.BNode:
		if !allowRefs {
			panic(p.s.vm.ToValue(errRegexpOnIRI))
		}
	default:
		panic(p.s.vm.ToValue(errUnknownType{Val: v}))
	}
	var (
		s    string
		refs bool
	)
	switch v := v.(type) {
	case quad.String:
		s = string(v)
	case quad.IRI:
		s, refs = string(v), true
	case quad.BNode:
		s, refs = string(v), true
	default:
		panic(p.s.vm.ToValue(errUnknownType{Val: v}))
	}
	re, err := regexp.Compile(s)
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}

	np := p.clonePath().Filters(shape.Regexp{Re: re, Refs: refs})
	return p.newVal(np)
}

// Like applies constraints to a set of nodes. Can be used to filter values by range or match strings.
func (p *pathObject) Like(call goja.FunctionCall) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) != 1 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 1, Got: len(args)}))
	}
	pattern, ok := args[0].(string)
	if !ok {
		panic(p.s.vm.ToValue(errType{Expected: "", Got: args[0]}))
	}

	np := p.clonePath().Filters(shape.Wildcard{Pattern: pattern})
	return p.newVal(np)
}

// Compare applies constraints to a set of nodes. Can be used to filter values by range or match strings.
func (p *pathObject) Compare(call goja.FunctionCall) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) != 2 {
		panic(p.s.vm.ToValue(errArgCountNum{Expected: 2, Got: len(args)}))
	}

	op, ok := toInt(args[0])
	if !ok {
		panic(p.s.vm.ToValue(errType{Expected: 1, Got: op}))
	}

	qv, err := toQuadValue(args[1])
	if err != nil {
		panic(p.s.vm.ToValue(err))
	}

	np := p.clonePath().Filters(shape.Comparison{Op: iterator.Operator(op), Val: qv})
	return p.newVal(np)
}

// Type applies constraints to a set of nodes. Can be used to filter values by range or match strings.
func (p *pathObject) Type(call goja.FunctionCall) goja.Value {
	args := exportArgs(call.Arguments)
	if len(args) == 0 {
		panic(p.s.vm.ToValue(errArgCount{Got: len(args)}))
	}

	np := p.clonePath().Filters(filterTypes{types: toStrings(args)})
	return p.newVal(np)
}

// Type applies constraints to a set of nodes. Can be used to filter values by range or match strings.
func (p *pathObject) Literal(_ goja.FunctionCall) goja.Value {
	np := p.clonePath().Filters(filterTypes{types: []string{"str", "int", "float", "bool", "time", "lang", "typed"}})
	return p.newVal(np)
}

// Map calls callback(data) for each result.
// Signature: (callback)
//
// Arguments:
//
// * `callback`: A javascript function of the form `function(data)`
//
// Example:
// 	// javascript
//	// Simulate query.All().All()
//	graph.V("<alice>").Map(function(d) { return "<bob>" } )
func (p *pathObject) Map(call goja.FunctionCall) goja.Value {
	if n := len(call.Arguments); n != 1 {
		panic(p.s.vm.ToValue(errArgCount{Got: len(call.Arguments)}))
	}

	fn, ok := goja.AssertFunction(call.Argument(0))
	if !ok {
		panic(p.s.vm.ToValue("expected callback function"))
	}

	np := p.clonePath().Maps(mapperCallback{sess: p.s, call: call, fn: fn})
	return p.newVal(np)
}

// Limit limits a number of nodes for current path.
//
// Arguments:
//
// * `limit`: A number of nodes to limit results to.
//
// Example:
// 	// javascript
//	// Start from all nodes that follow bob, and limit them to 2 nodes -- results in alice and charlie
//	g.V().has("<follows>", "<bob>").limit(2).all()
func (p *pathObject) Limit(limit int) *pathObject {
	np := p.clonePath().Limit(int64(limit))
	return p.new(np)
}

// Skip skips a number of nodes for current path.
//
// Arguments:
//
// * `offset`: A number of nodes to skip.
//
// Example:
//	// javascript
//	// Start from all nodes that follow bob, and skip 2 nodes -- results in dani
//	g.V().has("<follows>", "<bob>").skip(2).all()
func (p *pathObject) Skip(offset int) *pathObject {
	np := p.clonePath().Skip(int64(offset))
	return p.new(np)
}

func (p *pathObject) Order() *pathObject {
	np := p.clonePath().Order()
	return p.new(np)
}
