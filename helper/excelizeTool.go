package helper

import "fmt"

// GetColumnName 根据列索引获取 Excel 列名，例如 1 -> A, 2 -> B, ..., 27 -> AA
func GetColumnName(index int) string {
	// Excel 列是从 1 开始的，所以需要减去 1
	index -= 1
	result := ""
	for index >= 0 {
		remainder := index % 26
		result = fmt.Sprintf("%c", 'A'+remainder) + result
		index = index/26 - 1
	}
	return result
}
