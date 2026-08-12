package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var typeRegistry sync.Map

func Register(obj interface{}) {
	t := reflect.TypeOf(obj).Elem()
	typeRegistry.Store(strings.ToLower(t.Name()), t)
}

func RegisterInitializer(name string, initializer interface{}) {
	typeRegistry.Store(strings.ToLower(name), initializer)
}

func GetInstance[T any](name string, args ...interface{}) (ins T, err error) {
	name = strings.ToLower(name)
	initializer, ok := typeRegistry.Load(name)
	if !ok {
		return ins, errors.New("type not found")
	}

	fnValue := reflect.ValueOf(initializer)
	if fnValue.Kind() != reflect.Func {
		return ins, errors.New("initializer is not a function")
	}
	// 确保函数参数与传入参数匹配
	fnType := fnValue.Type()
	if fnType.NumIn() != len(args) {
		return ins, fmt.Errorf("initializer requires %d arguments, but %d provided", fnType.NumIn(), len(args))
	}
	// 准备反射参数
	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		expectedType := fnType.In(i)
		if reflect.TypeOf(arg) == expectedType {
			in[i] = reflect.ValueOf(arg)
			continue
		}
		argValue := reflect.ValueOf(arg)
		if jsonNumber, ok := arg.(json.Number); ok && expectedType.Kind() == reflect.Int {
			parsedValue, err := jsonNumber.Int64() // 尝试解析为 Int64
			if err != nil {
				return ins, fmt.Errorf("argument %d cannot be converted from json.Number to int: %v", i+1, err)
			}
			argValue = reflect.ValueOf(int(parsedValue))
		} else if !argValue.Type().ConvertibleTo(expectedType) {
			return ins, fmt.Errorf("argument %d should be of type %s, but got %s and cannot be converted", i+1, expectedType, reflect.TypeOf(arg))
		}

		in[i] = argValue.Convert(expectedType)
	}
	// 调用初始化函数
	inst := fnValue.Call(in)[0].Interface()
	// 确保返回值实现了目标接口 T
	ins, iok := inst.(T)
	if !iok {
		return ins, errors.New("returned type does not implement SocketClient")
	}
	return ins, nil

}
