package validator

import (
	"github.com/bitdlv/gokit/chain"
	"github.com/bitdlv/gokit/excel/cell"
	"github.com/bitdlv/gokit/excel/consts"
)

func RequiredCol(col int, actions []consts.Action, message ...string) chain.Handler[*cell.Value, *cell.Warning] {
	m := "字段不能为空"
	if len(message) > 0 {
		m = message[0]
	}
	return func(c *chain.Chain[*cell.Value, *cell.Warning]) {
		found := false
		for _, action := range actions {
			if c.Data.RowAction == action {
				found = true
				break
			}
		}

		if c.Data.Col == col && found && c.Data.IsEmpty() {
			c.AppendResult(&cell.Warning{
				Row:     c.Data.Row,
				Col:     c.Data.Col,
				Message: m,
			})
		}
	}
}
