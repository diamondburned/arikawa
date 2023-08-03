package handler

import (
	"fmt"
	"reflect"
	"sync"
)

var asserted sync.Map

// assertImpl asserts that ImplT implements T. If it does not, the program
// panics.
func assertImpl[T any, ImplT any]() {
	rt := reflect.TypeOf((*T)(nil))
	ri := reflect.TypeOf((*ImplT)(nil))

	_, checked := asserted.LoadOrStore([2]reflect.Type{rt, ri}, struct{}{})
	if checked {
		return
	}

	rt = rt.Elem()
	ri = ri.Elem()

	if rt.Kind() != reflect.Interface {
		panic(fmt.Sprintf("handler: assertImpl: T=%v is not an interface", rt))
	}

	if !ri.Implements(rt) {
		panic(fmt.Sprintf("handler: assertImpl: ImplT=%v does not implement T=%v", ri, rt))
	}
}
