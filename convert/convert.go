package convert

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/copier"

	cTime "github.com/bitdlv/gokit/time"
)

// copier.Copy 辅助转换器
func WithTime2String() copier.TypeConverter {
	converters := copier.TypeConverter{
		SrcType: time.Time{},
		DstType: copier.String,
		Fn: func(src interface{}) (interface{}, error) {
			s, ok := src.(time.Time)

			if !ok {
				return nil, errors.New("src type not matching")
			}

			if s.IsZero() {
				return "", nil
			}

			return s.Format("2006-01-02 15:04:05"), nil
		},
	}
	return converters
}

func WithTime2LocalString() copier.TypeConverter {
	converters := copier.TypeConverter{
		SrcType: time.Time{},
		DstType: copier.String,
		Fn: func(src interface{}) (interface{}, error) {
			s, ok := src.(time.Time)
			if s.IsZero() {
				return "", nil
			}
			if !ok {
				return nil, errors.New("src type not matching")
			}

			location, _ := time.LoadLocation("Local")

			return s.In(location).Format("2006-01-02 15:04:05"), nil
		},
	}
	return converters
}

func WithTime2East8String() copier.TypeConverter {
	converters := copier.TypeConverter{
		SrcType: time.Time{},
		DstType: copier.String,
		Fn: func(src interface{}) (interface{}, error) {
			s, ok := src.(time.Time)

			if !ok {
				return nil, errors.New("src type not matching")
			}

			if s.IsZero() {
				return "", nil
			}

			targetLocation, err := time.LoadLocation("Asia/Shanghai")
			if err != nil {
				return nil, errors.New("无法加载时间")

			}
			return s.In(targetLocation), nil

		},
	}
	return converters
}

func WIthString2Int() copier.TypeConverter {

	return copier.TypeConverter{
		SrcType: copier.String,
		DstType: copier.Int,
		Fn: func(src interface{}) (interface{}, error) {
			s, ok := src.(string)

			if !ok {
				return nil, errors.New("src type not matching")
			}

			return strconv.Atoi(s)
		},
	}
}
func WithStringToLocalTime() copier.TypeConverter {
	converters := copier.TypeConverter{
		SrcType: copier.String,
		DstType: time.Time{},
		Fn: func(src interface{}) (interface{}, error) {
			t, ok := src.(string)
			if !ok {
				return nil, fmt.Errorf("src type not matching")
			}
			return time.ParseInLocation(time.DateTime, t, time.Local)
		},
	}
	return converters
}

func StringToDate() copier.TypeConverter {
	return copier.TypeConverter{
		SrcType: copier.String,
		DstType: time.Time{},
		Fn: func(src interface{}) (dst interface{}, err error) {
			s, ok := src.(string)
			if !ok {
				return nil, errors.New("parse string to date error")
			}
			return time.Parse("2006-01-02 15:04:05", s)
		},
	}
}

func WithStringToTime() copier.TypeConverter {
	converters := copier.TypeConverter{
		SrcType: copier.String,
		DstType: time.Time{},
		Fn: func(src interface{}) (interface{}, error) {
			t, ok := src.(string)
			if !ok {
				return nil, fmt.Errorf("src type not matching")
			}
			return cTime.ParseLocal(t)
		},
	}
	return converters
}

func arrToString(arr []string) string {
	var r string
	if len(arr) == 1 {
		r = arr[0]
		return r
	}
	for _, v := range arr[1:] {
		r += "," + v
	}
	return r
}

func StringToArr() copier.TypeConverter {
	return copier.TypeConverter{
		SrcType: copier.String,
		DstType: []string{},
		Fn: func(src interface{}) (dst interface{}, err error) {
			s, ok := src.(string)
			if !ok {
				return nil, errors.New("字符串转数组报错")
			}
			return strings.Split(s, ","), nil
		},
	}
}

func ArrToString() copier.TypeConverter {
	return copier.TypeConverter{
		SrcType: []string{},
		DstType: copier.String,
		Fn: func(src interface{}) (dst interface{}, err error) {
			s, ok := src.([]string)
			if !ok {
				return nil, errors.New("字符串数组转字符串报错")
			}
			return arrToString(s), nil
		},
	}
}
