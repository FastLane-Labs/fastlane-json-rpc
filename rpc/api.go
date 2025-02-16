package rpc

import "reflect"

type Api interface {
	RuntimeMethod(methodName string) reflect.Value
}
