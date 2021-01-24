// Copyright 2014 The Cayley Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package iterator

import (
	"context"

	"github.com/cayleygraph/quad"

	"github.com/cayleygraph/cayley/graph"
)

var _ graph.IteratorFuture = &ValueMapper{}

type ValueMapper struct {
	it *valueMapper
	graph.Iterator
}

type ValueMapperFunc func(quad.Value) (quad.Value, error)

func NewValueMapper(qs graph.Namer, sub graph.Iterator, mapper ValueMapperFunc) *ValueMapper {
	it := &ValueMapper{
		it: newValueMapper(qs, graph.AsShape(sub), mapper),
	}
	it.Iterator = graph.NewLegacy(it.it, it)
	return it
}

func (it *ValueMapper) AsShape() graph.IteratorShape {
	it.Close()
	return it.it
}

var _ graph.IteratorShapeCompat = (*valueMapper)(nil)

type valueMapper struct {
	sub    graph.IteratorShape
	mapper ValueMapperFunc
	qs     graph.Namer
}

func newValueMapper(qs graph.Namer, sub graph.IteratorShape, mapper ValueMapperFunc) *valueMapper {
	return &valueMapper{
		sub:    sub,
		qs:     qs,
		mapper: mapper,
	}
}

func (it *valueMapper) Iterate() graph.Scanner {
	return newValueMapperNext(it.qs, it.sub.Iterate(), it.mapper)
}

func (it *valueMapper) Lookup() graph.Index {
	return newValueMapperContains(it.qs, it.sub.Lookup(), it.mapper)
}

func (it *valueMapper) AsLegacy() graph.Iterator {
	it2 := &ValueMapper{it: it}
	it2.Iterator = graph.NewLegacy(it, it2)
	return it2
}

func (it *valueMapper) SubIterators() []graph.IteratorShape {
	return []graph.IteratorShape{it.sub}
}

func (it *valueMapper) String() string {
	return "ValueMapper"
}

// There's nothing to optimize, locally, for a value-comparison iterator.
// Replace the underlying iterator if need be.
// potentially replace it.
func (it *valueMapper) Optimize(ctx context.Context) (graph.IteratorShape, bool) {
	newSub, changed := it.sub.Optimize(ctx)
	if changed {
		it.sub = newSub
	}
	return it, true
}

// We're only as expensive as our subiterator.
// Again, optimized value comparison iterators should do better.
func (it *valueMapper) Stats(ctx context.Context) (graph.IteratorCosts, error) {
	st, err := it.sub.Stats(ctx)
	st.Size.Size = st.Size.Size/2 + 1
	st.Size.Exact = false
	return st, err
}

type valueMapperNext struct {
	sub    graph.Scanner
	mapper ValueMapperFunc
	qs     graph.Namer
	result graph.Ref
	err    error
}

func newValueMapperNext(qs graph.Namer, sub graph.Scanner, mapper ValueMapperFunc) *valueMapperNext {
	return &valueMapperNext{
		sub:    sub,
		qs:     qs,
		mapper: mapper,
	}
}

func (it *valueMapperNext) doMap(val graph.Ref) quad.Value {
	qval := it.qs.NameOf(val)
	ok, err := it.mapper(qval)
	if err != nil {
		it.err = err
	}
	return ok
}

func (it *valueMapperNext) Close() error {
	return it.sub.Close()
}

func (it *valueMapperNext) Next(ctx context.Context) bool {
	for it.sub.Next(ctx) {
		val := it.sub.Result()
		nval := it.qs.ValueOf(it.doMap(val))
		if nval != nil {
			it.result = nval
			return true
		}
	}
	it.err = it.sub.Err()
	return false
}

func (it *valueMapperNext) Err() error {
	return it.err
}

func (it *valueMapperNext) Result() graph.Ref {
	return it.result
}

func (it *valueMapperNext) NextPath(ctx context.Context) bool {
	return it.sub.NextPath(ctx)
}

// If we failed the check, then the subiterator should not contribute to the result
// set. Otherwise, go ahead and tag it.
func (it *valueMapperNext) TagResults(dst map[string]graph.Ref) {
	it.sub.TagResults(dst)
}

func (it *valueMapperNext) String() string {
	return "ValueMapperNext"
}

type valueMapperContains struct {
	sub    graph.Index
	mapper ValueMapperFunc
	qs     graph.Namer
	result graph.Ref
	err    error
}

func newValueMapperContains(qs graph.Namer, sub graph.Index, mapper ValueMapperFunc) *valueMapperContains {
	return &valueMapperContains{
		sub:    sub,
		qs:     qs,
		mapper: mapper,
	}
}

func (it *valueMapperContains) doMap(val graph.Ref) quad.Value {
	qval := it.qs.NameOf(val)
	ok, err := it.mapper(qval)
	if err != nil {
		it.err = err
	}
	return ok
}

func (it *valueMapperContains) Close() error {
	return it.sub.Close()
}

func (it *valueMapperContains) Err() error {
	return it.err
}

func (it *valueMapperContains) Result() graph.Ref {
	return it.result
}

func (it *valueMapperContains) NextPath(ctx context.Context) bool {
	return it.sub.NextPath(ctx)
}

func (it *valueMapperContains) Contains(ctx context.Context, val graph.Ref) bool {
	nval := it.qs.ValueOf(it.doMap(val))
	if nval == nil {
		return false
	}
	ok := it.sub.Contains(ctx, nval)
	if !ok {
		it.err = it.sub.Err()
	}
	return ok
}

// If we failed the check, then the subiterator should not contribute to the result
// set. Otherwise, go ahead and tag it.
func (it *valueMapperContains) TagResults(dst map[string]graph.Ref) {
	it.sub.TagResults(dst)
}

func (it *valueMapperContains) String() string {
	return "ValueMapperContains"
}
