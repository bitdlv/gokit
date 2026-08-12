package convert

type Num interface {
	uint8 | int8 | uint16 | int16 | uint32 | int32 | int | uint | int64 | uint64
}

// NumToNum 各类型数字切片相互转换
func NumToNum[a Num, b Num](sli []a) []b {
	var ret []b
	for _, v := range sli {
		ret = append(ret, b(v))
	}
	return ret
}

// Unique 切片去重
func Unique[T comparable](s []T) []T {
	seen := make(map[T]struct{})
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}
