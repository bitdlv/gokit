package cell

import (
	"github.com/bitdlv/gokit/excel/consts"
	"github.com/gogf/gf/v2/util/gconv"
)

const CleanFlag = "清空"

type Value struct {
	Row       int
	Col       int
	Value     any
	RowAction consts.Action
	Header    string
}

func (c *Value) ToString() string {
	if c.Value == CleanFlag {
		return ""
	}
	return gconv.String(c.Value)
}

func (c *Value) ToInt32() int32 {
	if c.Value == CleanFlag {
		return int32(0)
	}
	return gconv.Int32(c.Value)
}

func (c *Value) ShouldClear() bool {
	return c.Value == CleanFlag
}

// IsEmpty 将值转换为字符串，判断是否为空字符串
func (c *Value) IsEmpty() bool {
	return c.ToString() == ""
}
