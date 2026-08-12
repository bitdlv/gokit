package util

import (
	"fmt"
	"github.com/tealeg/xlsx/v2"
	"reflect"
)

// objType待解析目标对象实例，返回objType类型数组
func ParseExcel(objInstance interface{}, file *xlsx.File) ([]interface{}, error) {
	var maxRow int = 5000
	var result []interface{}
	instanceType := reflect.TypeOf(objInstance)
	for _, sheet := range file.Sheets {
		if len(sheet.Rows) > maxRow {
			return nil, fmt.Errorf("导入数据超过%d条", maxRow)
		}
		for i, row := range sheet.Rows {
			if i == 0 {
				continue
			}
			columnCount := len(row.Cells)
			if columnCount > instanceType.NumField() {
				columnCount = instanceType.NumField()
			}
			//反射new实例对象
			instance := reflect.New(instanceType).Elem()
			for i := 0; i < columnCount; i++ {
				//实例化对象赋值
				//TODO 导入数据类型支持（数值、时间等）
				instance.FieldByName(instanceType.Field(i).Name).Set(reflect.ValueOf(row.Cells[i].Value))
			}
			result = append(result, instance.Interface())
		}
	}
	return result, nil
}
