package util

import (
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"math"
	"regexp"

	"github.com/tealeg/xlsx/v2"
)

type tableToExcel struct {
	file         *xlsx.File
	sheetNum     int
	currentSheet *xlsx.Sheet
	head         []string
	body         [][]string
	isHorizontal bool
}

func TableToExcel() *tableToExcel {
	excel := tableToExcel{}
	excel.file = xlsx.NewFile()
	excel.NextSheet()
	return &excel
}

func (excel *tableToExcel) NextSheet() *tableToExcel {
	excel.writeCurrentSheet()
	excel.sheetNum++
	sheetName := fmt.Sprintf("Sheet %d", excel.sheetNum)
	sheet, err := excel.file.AddSheet(sheetName)
	if err != nil {
		logx.Error("build excel fail please check")
		return excel
	}
	excel.currentSheet = sheet
	excel.head = make([]string, 0)
	excel.body = make([][]string, 0)
	excel.isHorizontal = false
	return excel
}

func (excel *tableToExcel) Horizontal() *tableToExcel {
	excel.isHorizontal = true
	return excel
}

func (excel *tableToExcel) Vertical() *tableToExcel {
	excel.isHorizontal = false
	return excel
}

func (excel *tableToExcel) SetTitle(title string) *tableToExcel {
	excel.currentSheet.Name = title
	return excel
}

func (excel *tableToExcel) SetHead(ths ...string) *tableToExcel {
	excel.head = ths
	return excel
}

func (excel *tableToExcel) SetHeadByInterface(ths ...interface{}) *tableToExcel {
	excel.head = make([]string, len(ths))
	for i, th := range ths {
		if th == nil {
			excel.head[i] = ""
		} else {
			excel.head[i] = fmt.Sprintf("%v", th)
		}
	}
	return excel
}

func (excel *tableToExcel) SetBody(body [][]string) *tableToExcel {
	excel.body = body
	return excel
}

func (excel *tableToExcel) AddRecord(tds ...interface{}) *tableToExcel {
	record := make([]string, len(tds))
	for i, th := range tds {
		if th == nil {
			record[i] = ""
		} else {
			record[i] = fmt.Sprintf("%v", th)
		}
	}
	excel.body = append(excel.body, record)
	return excel
}

func (excel *tableToExcel) Write(writer io.Writer) *tableToExcel {
	excel.writeCurrentSheet()
	err := excel.file.Write(writer)
	if err != nil {
		logx.Error("failed to write: " + err.Error())
	}
	return excel
}

func (excel *tableToExcel) writeCurrentSheet() *tableToExcel {
	if excel.sheetNum <= 0 {
		return excel
	}
	// 初始化列宽
	var columnWidths []int
	if excel.isHorizontal {
		columnWidths = make([]int, len(excel.body)+1)
	} else {
		columnWidths = make([]int, len(excel.head))
	}
	// 写表头
	headStyle := excel.getHeadStyle()
	for x, th := range excel.head {
		y := 0
		if excel.isHorizontal {
			x, y = y, x
		}
		cell := excel.currentSheet.Cell(y, x)
		cell.Value = th
		cell.SetStyle(headStyle)
		// 记录列宽
		width := getDisplayWidth(cell.Value)
		if columnWidths[x] < width {
			columnWidths[x] = width
		}
	}
	// 写表体
	bodyStyle := excel.getBodyStyle()
	for idx, record := range excel.body {
		for x, td := range record {
			y := idx + 1
			if excel.isHorizontal {
				x, y = y, x
			}
			cell := excel.currentSheet.Cell(y, x)
			cell.Value = td
			cell.SetStyle(bodyStyle)
			// 记录列宽
			width := getDisplayWidth(cell.Value)
			if x < len(columnWidths) && columnWidths[x] < width {
				columnWidths[x] = width
			}
		}
	}
	// 调整列宽
	for x, cw := range columnWidths {
		width := math.Min(math.Max(float64(cw+2), 8), 60)
		excel.currentSheet.SetColWidth(x+1, x+1, width)
	}
	return excel
}

func (excel *tableToExcel) getHeadStyle() *xlsx.Style {
	style := xlsx.NewStyle()
	style.Alignment = xlsx.Alignment{Horizontal: "center", Vertical: "center"}
	style.Border = *xlsx.NewBorder("thin", "thin", "thin", "thin")
	style.Font = *xlsx.NewFont(11, "Microsoft YaHei")
	style.Font.Bold = true
	style.ApplyAlignment = true
	style.ApplyBorder = true
	style.ApplyFont = true
	return style
}

func (excel *tableToExcel) getBodyStyle() *xlsx.Style {
	style := xlsx.NewStyle()
	style.Alignment = xlsx.Alignment{Vertical: "center"}
	style.Font = *xlsx.NewFont(11, "Microsoft YaHei")
	style.ApplyAlignment = true
	style.ApplyFont = true
	return style
}

var tester *regexp.Regexp

func getDisplayWidth(s string) int {
	if tester == nil {
		tester, _ = regexp.Compile("[^\\x00-\\xff]")
	}
	return len(tester.ReplaceAllString(s, "01"))
}
