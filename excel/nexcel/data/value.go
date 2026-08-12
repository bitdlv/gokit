package data

import (
	"github.com/gogf/gf/v2/util/gconv"
)

func NewValue(col, row int, value string) *Value {
	return &Value{
		Row:   row,
		Col:   col,
		value: value,
	}
}

const CleanFlag = "清空"

type Value struct {
	Row   int
	Col   int
	value string
}

func (c *Value) ToString() string {
	if c.value == CleanFlag {
		return ""
	}
	return c.value
}

func (c *Value) ToInt32() int32 {
	if c.value == CleanFlag {
		return int32(0)
	}
	return gconv.Int32(c.value)
}

func (c *Value) ShouldClear() bool {
	return c.value == CleanFlag
}

// IsEmpty 将值转换为字符串，判断是否为空字符串
func (c *Value) IsEmpty() bool {
	return c.value == ""
}

type Group struct {
	Header []*Value
	Cells  []*Value
}

// func (g *Group) Foreach(f func(cell *Value, header *Value) (stop bool, err error)) (err error) {
// 	for index, c := range g.Cells {
// 		var header *Value
// 		if len(g.Header) > index {
// 			header = g.Header[index]
// 		}
// 		var stop bool
// 		stop, err = f(c, header)
// 		if stop || err != nil {
// 			return
// 		}
// 	}
// 	return
// }
