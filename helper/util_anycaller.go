package helper

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// AnyCall 通过字符串调用与该字符串同名的方法
// 被调用方法 返回值数量为 2 个 T, nil
func AnyCall[T any](obj interface{}, methodName string, args ...interface{}) (T, error) {
	var zeroResp T
	objValue := reflect.ValueOf(obj)
	methodName = fmt.Sprintf("%s", strings.Title(methodName))
	method := objValue.MethodByName(methodName)
	if !method.IsValid() {
		return zeroResp, errors.New("method not found: " + methodName)
	}
	if len(args) != method.Type().NumIn() {
		return zeroResp, errors.New("incorrect number of arguments")
	}
	methodArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		if reflect.TypeOf(arg) != method.Type().In(i) {
			return zeroResp, fmt.Errorf("argument %d should be of type %s", i+1, method.Type().In(i))
		}
		methodArgs[i] = reflect.ValueOf(arg)
	}
	results := method.Call(methodArgs)
	if len(results) > 0 {
		if result, ok := results[0].Interface().(T); ok {
			return result, nil
		}
		return zeroResp, fmt.Errorf("method returned value cannot be converted to type %T", zeroResp)
	}
	return zeroResp, nil
}
