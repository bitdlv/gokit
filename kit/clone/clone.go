package clone

import (
	"encoding/json"
	"reflect"
)

func buildOptions(opts ...Option) *Options {
	o := defaultOptions()

	for _, opt := range opts {
		opt(o)
	}

	return o
}

func deepCopy[T any](src T) T {
	data, _ := json.Marshal(src)

	var dst T

	_ = json.Unmarshal(data, &dst)

	return dst
}

func Struct[T any](src T, opts ...Option) T {
	dst := deepCopy(src)

	options := buildOptions(opts...)

	cleanStruct(&dst, options)

	return dst
}

func Slice[T any](src []T, opts ...Option) []T {
	dst := deepCopy(src)

	options := buildOptions(opts...)

	for i := range dst {
		cleanStruct(&dst[i], options)
	}

	return dst
}

func Map[K comparable, V any](
	src map[K]V,
	opts ...Option,
) map[K]V {

	if src == nil {
		return nil
	}

	dst := deepCopy(src)

	options := buildOptions(opts...)

	for k := range dst {

		key := any(k).(interface{})

		if s, ok := key.(string); ok {

			if _, exists := options.IgnoreFields[s]; exists {
				delete(dst, k)
			}

			if s == "ID" ||
				s == "Id" ||
				s == "CreatedAt" {
				delete(dst, k)
			}
		}
	}

	return dst
}

func cleanStruct(obj any, opts *Options) {
	v := reflect.ValueOf(obj)

	if v.Kind() != reflect.Pointer {
		return
	}

	v = v.Elem()

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()

	for i := 0; i < v.NumField(); i++ {

		fieldValue := v.Field(i)
		fieldType := t.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if shouldIgnoreField(fieldType, opts) {
			resetValue(fieldValue)
			continue
		}

		// 递归处理嵌套 struct
		if fieldValue.Kind() == reflect.Struct {

			if fieldValue.Type() == timeType {
				continue
			}

			cleanStruct(
				fieldValue.Addr().Interface(),
				opts,
			)
		}
	}
}
