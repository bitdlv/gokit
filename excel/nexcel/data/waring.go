package data

type Warning struct {
	//Row 单元格横坐标 从1开始
	Row int
	//Col 单元格纵坐标 从1开始
	Col     int
	Message string
}
