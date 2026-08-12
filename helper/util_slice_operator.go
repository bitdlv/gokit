package helper

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// SliceOperator SliceOperate 切片操作
// 获取两切片的 合集 交集 差值(A—B) 差值(B—A)
func SliceOperator[T any](A, B []T, equalFunc func(a, b T) bool) (union, intersection, differenceAB, differenceBA []T) {
	// 使用切片来存储元素
	var setA []T
	var setB []T

	// 将数组 A 的元素存入 setA（去重）
	for _, item := range A {
		if !contains(setA, item, equalFunc) {
			setA = append(setA, item)
		}
	}

	// 将数组 B 的元素存入 setB（去重）
	for _, item := range B {
		if !contains(setB, item, equalFunc) {
			setB = append(setB, item)
		}
	}

	// 计算交集
	for _, itemA := range setA {
		for _, itemB := range setB {
			if equalFunc(itemA, itemB) {
				intersection = append(intersection, itemA)
				break
			}
		}
	}

	// 计算并集
	union = append(union, setA...)
	for _, itemB := range setB {
		if !contains(union, itemB, equalFunc) {
			union = append(union, itemB)
		}
	}

	// 计算差集 (A - B)
	for _, itemA := range setA {
		if !contains(setB, itemA, equalFunc) {
			differenceAB = append(differenceAB, itemA)
		}
	}

	// 计算差集 (B - A)
	for _, itemB := range setB {
		if !contains(setA, itemB, equalFunc) {
			differenceBA = append(differenceBA, itemB)
		}
	}

	return intersection, union, differenceAB, differenceBA
}

// contains 检查切片中是否包含某个元素
func contains[T any](slice []T, item T, equalFunc func(a, b T) bool) bool {
	for _, s := range slice {
		if equalFunc(s, item) {
			return true
		}
	}
	return false
}

func JoinAnySlice(slice interface{}, sep string) string {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return ""
	}
	var parts []string
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		parts = append(parts, fmt.Sprintf("%v", item))
	}
	return strings.Join(parts, sep)
}

type Supported interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string
}

// ParseSlice 将一个以 sep 分隔的字符串转换为目标类型的 slice
func ParseSlice[T Supported](input string, sep string) ([]T, error) {
	parts := strings.Split(input, sep)
	result := make([]T, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		var val T
		var err error

		switch any(val).(type) {
		case int:
			var v int64
			v, err = strconv.ParseInt(p, 10, 0)
			val = any(int(v)).(T)
		case int8:
			var v int64
			v, err = strconv.ParseInt(p, 10, 8)
			val = any(int8(v)).(T)
		case int16:
			var v int64
			v, err = strconv.ParseInt(p, 10, 16)
			val = any(int16(v)).(T)
		case int32:
			var v int64
			v, err = strconv.ParseInt(p, 10, 32)
			val = any(int32(v)).(T)
		case int64:
			var v int64
			v, err = strconv.ParseInt(p, 10, 64)
			val = any(v).(T)
		case uint:
			var v uint64
			v, err = strconv.ParseUint(p, 10, 0)
			val = any(uint(v)).(T)
		case uint8:
			var v uint64
			v, err = strconv.ParseUint(p, 10, 8)
			val = any(uint8(v)).(T)
		case uint16:
			var v uint64
			v, err = strconv.ParseUint(p, 10, 16)
			val = any(uint16(v)).(T)
		case uint32:
			var v uint64
			v, err = strconv.ParseUint(p, 10, 32)
			val = any(uint32(v)).(T)
		case uint64:
			var v uint64
			v, err = strconv.ParseUint(p, 10, 64)
			val = any(v).(T)
		case float32:
			var v float64
			v, err = strconv.ParseFloat(p, 32)
			val = any(float32(v)).(T)
		case float64:
			var v float64
			v, err = strconv.ParseFloat(p, 64)
			val = any(v).(T)
		case string:
			val = any(p).(T)
		default:
			return nil, errors.New("unsupported type")
		}
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ArrayColumn 从结构体切片中提取某个字段组成新切片
func ArrayColumn[T any, R any](items any, getField func(T) R) []R {
	result := make([]R, 0)
	switch v := items.(type) {
	case []T:
		for _, item := range v {
			result = append(result, getField(item))
		}
	case map[any]T:
		for _, item := range v {
			result = append(result, getField(item))
		}
	default:
		// 不支持的类型，返回空切片
	}
	return result
}
