package gizmo

import "fmt"

var (
	errNoVia       = fmt.Errorf("expected predicate list")
	errRegexpOnIRI = fmt.Errorf("regexps are not allowed on IRIs")
)

type errArgCountNum struct {
	Expected int
	Got      int
}

func (e errArgCountNum) Error() string {
	return fmt.Sprintf("expected %d arguments, got %d", e.Expected, e.Got)
}

type errArgCount struct {
	Got int
}

func (e errArgCount) Error() string {
	return fmt.Sprintf("unexpected arguments count: %d", e.Got)
}

type errUnknownType struct {
	Val interface{}
}

func (e errUnknownType) Error() string {
	return fmt.Sprintf("unsupported type %T", e.Val)
}

type errType struct {
	Expected interface{}
	Got      interface{}
}

func (e errType) Error() string {
	return fmt.Sprintf("expected type %T, got %T", e.Expected, e.Got)
}

type errNotQuadValue struct {
	Val interface{}
}

func (e errNotQuadValue) Error() string {
	return fmt.Sprintf("not a quad.Value: %T", e.Val)
}

type Error struct {
	Errors []interface{}
}

func (e Error) Error() string {
	return fmt.Sprintf("%+v", e.Errors)
}
