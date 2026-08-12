package clone

import (
	"reflect"
	"strings"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

func shouldIgnoreField(
	field reflect.StructField,
	opts *Options,
) bool {

	// clone:"ignore"
	if field.Tag.Get(CloneTag) == CloneIgnore {
		return true
	}

	// 自定义忽略字段
	if _, ok := opts.IgnoreFields[field.Name]; ok {
		return true
	}

	// gorm 主键
	if opts.IgnorePrimaryKey {
		gormTag := field.Tag.Get("gorm")

		if strings.Contains(gormTag, "primaryKey") {
			return true
		}
	}

	// 常见主键字段
	if opts.IgnorePrimaryKey {
		switch field.Name {
		case "ID", "Id":
			return true
		}
	}

	// 创建时间
	if opts.IgnoreCreatedAt {

		switch field.Name {
		case "CreatedAt", "CreateTime":
			return true
		}

		if field.Type == timeType &&
			strings.Contains(
				strings.ToLower(field.Name),
				"created",
			) {
			return true
		}
	}

	return false
}

func resetValue(v reflect.Value) {
	v.Set(reflect.Zero(v.Type()))
}
