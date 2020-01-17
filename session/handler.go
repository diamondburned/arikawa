package session

import (
	"errors"
	"reflect"
)

type handler struct {
	event    reflect.Type
	callback reflect.Value
}

func reflectFn(function interface{}) (*handler, error) {
	fnV := reflect.ValueOf(function)
	fnT := fnV.Type()

	if fnT.Kind() != reflect.Func {
		return nil, errors.New("given interface is not a function")
	}

	if fnT.NumIn() != 1 {
		return nil, errors.New("function can only accept 1 event as argument")
	}

	argT := fnT.In(0)

	if argT.Kind() != reflect.Ptr {
		return nil, errors.New("first argument is not pointer")
	}

	return &handler{
		event:    argT,
		callback: fnV,
	}, nil
}

func (h handler) not(event reflect.Type) bool {
	return h.event != event
}

func (h handler) call(event reflect.Value) {
	h.callback.Call([]reflect.Value{event})
}
