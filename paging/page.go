package paging

// Normalize 归一化 page / pageSize 并计算 offset。
//
//   - page <= 0    → 1
//   - pageSize <= 0 → defaultPageSize（默认 10）
//   - offset = (page - 1) * pageSize
//
// defaultPageSize 可选：不传则用 10。仅取第一个可变参数。
//
// 用法：
//
//	page, size, offset := paging.Normalize(req.Page, req.PageSize)
//	page, size, offset := paging.Normalize(req.Page, req.PageSize, 20) // 自定义默认 20
func Normalize(page, pageSize int, defaultPageSize ...int) (int, int, int) {
	def := 10
	if len(defaultPageSize) > 0 && defaultPageSize[0] > 0 {
		def = defaultPageSize[0]
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = def
	}
	offset := (page - 1) * pageSize
	return page, pageSize, offset
}
