package rpc

import (
	"fmt"
	"reflect"
)

func formatConversionErrMsg(i int, call *reflect.Value) string {
	return fmt.Sprintf("Param [%d] can't be converted to %s", i, call.Type().In(i).Name())
}
