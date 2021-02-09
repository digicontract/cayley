package gizmo

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

const constructMethodPrefix = "New"

type fieldNameMapper struct{}

func (fieldNameMapper) FieldName(t reflect.Type, f reflect.StructField) string {
	return lcFirst(f.Name)
}

func (fieldNameMapper) MethodName(t reflect.Type, m reflect.Method) string {
	if strings.HasPrefix(m.Name, constructMethodPrefix) {
		return strings.TrimPrefix(m.Name, constructMethodPrefix)
	}
	return lcFirst(m.Name)
}

func lcFirst(str string) string {
	ch, size := utf8.DecodeRuneInString(str)
	return string(unicode.ToLower(ch)) + str[size:]
}
