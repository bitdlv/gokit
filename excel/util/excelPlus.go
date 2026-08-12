package util

import (
	"fmt"
	"github.com/tealeg/xlsx/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
)

type ExcelPlus struct {
	Titles []string
	File   *xlsx.File
	Sheets []*xlsx.Sheet
}

func (excel *ExcelPlus) BuildFile(numOfSheet int) {
	excel.File = xlsx.NewFile()
	sheets := make([]*xlsx.Sheet, 0)
	for i := 1; i <= numOfSheet; i++ {
		sheet, err := excel.File.AddSheet(fmt.Sprintf("Sheet%d", i))
		if err != nil {
			logx.Error("build excel fail please check")
		}
		sheets = append(sheets, sheet)
	}

	excel.Sheets = sheets
}

func (excel *ExcelPlus) BuildTitle(titles []string, sheet int) {
	if len(excel.Sheets) < sheet {
		logx.Error("out of sheet boundary")
		return
	}
	row := excel.Sheets[sheet-1].AddRow()
	excel.Titles = titles
	for _, title := range excel.Titles {
		cell := row.AddCell()
		cell.Value = title
	}
}

func (excel *ExcelPlus) WriteBody(datas []string, sheet int) {
	if len(excel.Sheets) < sheet {
		logx.Error("out of sheet boundary")
		return
	}
	row := excel.Sheets[sheet-1].AddRow()
	for _, data := range datas {
		cell := row.AddCell()
		cell.Value = data
	}
}

func (excel *ExcelPlus) Write(writer io.Writer) {
	excel.File.Write(writer)
}
