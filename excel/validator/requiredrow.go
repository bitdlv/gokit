package validator

import (
	"github.com/bitdlv/gokit/chain"
	"github.com/bitdlv/gokit/excel/cell"
	"github.com/bitdlv/gokit/excel/consts"
	"github.com/bitdlv/gokit/helper"
)

// RequiredOneCol 验证多个字段中至少有一字段不能为空
func RequiredOneCol(cols []int, actions []consts.Action, message ...string) chain.Handler[[]*cell.Value, *cell.Warning] {
	m := "字段中至少有一项不能为空"
	if len(message) > 0 {
		m = message[0]
	}

	return func(c *chain.Chain[[]*cell.Value, *cell.Warning]) {
		pass := false
		row := 0
		for _, item := range c.Data {
			if !helper.Contains([]consts.Action{item.RowAction}, actions) {
				pass = true
				break
			}
			if row == 0 {
				row = item.Row
			}
			for _, col := range cols {
				if item.Col == col && !item.IsEmpty() {
					pass = true
					break
				}
			}
			if pass {
				break
			}
		}

		if !pass {
			c.AppendResult(&cell.Warning{
				Row:     row,
				Message: m,
			})
		}
	}
}
