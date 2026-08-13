package clone

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

func buildOptions(opts ...Option) *Options {
	o := defaultOptions()

	for _, opt := range opts {
		opt(o)
	}

	return o
}

func deepCopy[T any](src T) T {
	data, err := json.Marshal(src)
	if err != nil {
		panic(fmt.Sprintf("clone: json.Marshal failed: %v", err))
	}

	var dst T
	if err := json.Unmarshal(data, &dst); err != nil {
		panic(fmt.Sprintf("clone: json.Unmarshal failed: %v", err))
	}

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

		if s, ok := any(k).(string); ok {

			if _, exists := options.IgnoreFields[s]; exists {
				delete(dst, k)
			}

			if options.IgnorePrimaryKey &&
				(s == "ID" || s == "Id") {
				delete(dst, k)
			}

			if options.IgnoreCreatedAt && s == "CreatedAt" {
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
			// 嵌入匿名 struct（类型未导出）整体不可 Set，
			// 但其导出子字段仍可 Set — 用 unsafe 构造可寻址指针递归清理
			if fieldType.Anonymous &&
				fieldValue.Kind() == reflect.Struct &&
				fieldValue.CanAddr() {

				realPtr := reflect.NewAt(
					fieldValue.Type(),
					unsafe.Pointer(fieldValue.UnsafeAddr()),
				)
				cleanStruct(realPtr.Interface(), opts)
			}
			continue
		}

		if shouldIgnoreField(fieldType, opts) {
			resetValue(fieldValue)
			continue
		}

		// 递归处理嵌套 struct 与 *struct
		switch fieldValue.Kind() {
		case reflect.Struct:
			if fieldValue.Type() == timeType {
				continue
			}

			cleanStruct(
				fieldValue.Addr().Interface(),
				opts,
			)

		case reflect.Pointer:
			if fieldValue.IsNil() {
				continue
			}

			elem := fieldValue.Elem()
			if elem.Kind() != reflect.Struct {
				continue
			}

			if elem.Type() == timeType {
				continue
			}

			// fieldValue 本身就是 *struct，可直接传入
			cleanStruct(fieldValue.Interface(), opts)
		}
	}
}
