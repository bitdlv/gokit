package consts

type Action int

const (
	Create Action = iota + 1
	Update
	Delete
)

// ── merged from nexcel/consts (string labels) ──
const (
	CreateLabel string = "新增"
	UpdateLabel string = "修改"
)
