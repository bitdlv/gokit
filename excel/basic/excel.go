package basic

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/bitdlv/gokit/chain"
	"github.com/bitdlv/gokit/excel/cell"
	"github.com/bitdlv/gokit/excel/consts"

	"github.com/gogf/gf/util/gconv"
	"github.com/xuri/excelize/v2"
)

// WithSkipFirstRows 设置跳过前面的行数,一般用来跳过表头,默认为1
func WithSkipFirstRows(num int) func(importer *Importer) {
	return func(importer *Importer) {
		importer.skipFirstRows = num
	}
}

func WithCellValidators(validators ...chain.Handler[*cell.Value, *cell.Warning]) func(importer *Importer) {
	return func(importer *Importer) {
		importer.cellValidators = chain.New(validators)
	}
}

func WithRowValidators(validators ...chain.Handler[[]*cell.Value, *cell.Warning]) func(importer *Importer) {
	return func(importer *Importer) {
		importer.rowValidators = chain.New(validators)
	}
}

func WithCreateProcessor(processor DataProcessor) func(*Importer) {
	return func(i *Importer) {
		i.create = processor
	}
}

func WithUpdateProcessor(processor DataProcessor) func(*Importer) {
	return func(i *Importer) {
		i.update = processor
	}
}

func WithDeleteProcessor(processor DataProcessor) func(*Importer) {
	return func(i *Importer) {
		i.delete = processor
	}
}

func NewImporter(opts ...func(importer *Importer)) *Importer {
	importer := &Importer{
		skipFirstRows: 1,
	}
	for _, opt := range opts {
		opt(importer)
	}
	return importer
}

// Importer excel导入工具，需自己编写数据处理逻辑
type Importer struct {
	create         DataProcessor
	update         DataProcessor
	delete         DataProcessor
	skipFirstRows  int
	cellValidators *chain.Chain[*cell.Value, *cell.Warning]
	rowValidators  *chain.Chain[[]*cell.Value, *cell.Warning]
}

// Import 开始导入，col设置每行需要读取的列
func (e *Importer) Import(content []byte, col int) (allWarnings []*cell.Warning, err error) {
	data, allWarnings, err := e.parseExcel(content, col)
	if err != nil {
		return
	}

	if data[consts.Create] != nil && len(data[consts.Create]) > 0 {
		var warnings []*cell.Warning
		warnings, err = e.create(data[consts.Create])
		if err != nil {
			return
		}
		allWarnings = append(allWarnings, warnings...)
	}
	if data[consts.Update] != nil && len(data[consts.Update]) > 0 {
		var warnings []*cell.Warning
		warnings, err = e.update(data[consts.Update])
		if err != nil {
			return
		}
		allWarnings = append(allWarnings, warnings...)
	}
	if data[consts.Delete] != nil && len(data[consts.Delete]) > 0 {
		var warnings []*cell.Warning
		warnings, err = e.delete(data[consts.Delete])
		if err != nil {
			return
		}
		allWarnings = append(allWarnings, warnings...)
	}

	return
}

// GetDataFromActiveSheet 从默认活动的sheet获取数据
//
// Deprecated: 与业务逻辑不符，请使用Import方法
func (e *Importer) GetDataFromActiveSheet(content []byte, col int) (data [][]string, err error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return
	}
	defer f.Close()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	rows, err := f.GetRows(sheet)
	data = make([][]string, 0, len(rows))
	for index := range rows {
		c := make([]string, 0, col)
		for i := 0; i < col; i++ {
			var v string
			v, err = f.GetCellValue(sheet, Cell(i+1, index+1))
			if err != nil {
				return
			}
			c = append(c, v)
		}
		data = append(data, c)
	}
	return
}

// parseExcel excel解析方法，根据新增，修改，删除分组
//
// 验证不通过的行将会被过滤，并将警告加到warnings中
func (e *Importer) parseExcel(content []byte, col int) (data map[consts.Action][][]*cell.Value, warnings []*cell.Warning, err error) {
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return
	}
	defer f.Close()

	sheet := f.GetSheetName(1)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return
	}

	if len(rows) == e.skipFirstRows { // 空文件判断
		return
	}

	headers := rows[0]

	data = make(map[consts.Action][][]*cell.Value, 3)
	skip := 0
	// TODO: 如何并发?
	for index := range rows {
		if skip < e.skipFirstRows { // 跳过前面不需要的行
			skip++
			continue
		}

		var actionStr string
		actionStr, err = f.GetCellValue(sheet, Cell(1, index+1))
		if err != nil {
			return
		}

		var action consts.Action

		switch actionStr {
		case "新增":
			action = consts.Create
		case "修改":
			action = consts.Update
		case "删除":
			action = consts.Delete
		default:
			warnings = append(warnings, &cell.Warning{
				Row:     index + 1,
				Col:     1,
				Message: fmt.Sprintf("操作[%s]无效的操作类型", actionStr),
			})
			continue
		}

		c := make([]*cell.Value, 0, col)
		shouldContinue := false
		// 读取整行内容，边读取边验证，一旦有cell验证不通过，整行都不用，有两个固定字段: 操作类型和操作原因
		for i := 0; i < col; i++ {
			var v string
			v, err = f.GetCellValue(sheet, Cell(i+1, index+1))
			if err != nil {
				return
			}

			cel := &cell.Value{
				Col:       i + 1,
				Row:       index + 1,
				Value:     v,
				RowAction: action,
			}
			if len(headers) >= i+1 {
				cel.Header = headers[i]
			}

			// 验证
			if e.cellValidators != nil {
				e.cellValidators.Exec(cel)
				if results := e.cellValidators.Results(); len(results) > 0 {
					warnings = append(warnings, results...)
					shouldContinue = true
					break
				}
			}

			c = append(c, cel)
		}

		if shouldContinue {
			continue
		}

		if e.rowValidators != nil {
			e.rowValidators.Exec(c)
			if results := e.rowValidators.Results(); len(results) > 0 {
				warnings = append(warnings, results...)
				continue
			}
		}

		if _, ok := data[action]; !ok {
			data[action] = make([][]*cell.Value, 0, len(rows))
		}
		data[action] = append(data[action], c)
	}

	return
}

func Cell(col int, row int) string {
	return fmt.Sprintf("%s%d", ColIndexByNum(col), row)
}

func ColIndexByNum(num int) string {
	s := make([]string, 0, num/26)
	for num != 0 {
		tmp := num % 26
		num /= 26

		// 此处略微关键，当为0时，其实是26，也就是Z，
		// 而且当你将0调整为26后，需要从数字中去除26代表的这个数
		if tmp == 0 {
			tmp = 26
			num -= 1
		}
		s = append(s, fmt.Sprintf("%c", 'A'+tmp-1))
	}

	for i := 0; i < len(s)/2; i++ {
		temp := s[i]
		s[i] = s[len(s)-1-i]
		s[len(s)-1-i] = temp
	}

	return strings.Join(s, "")
}

type DataProcessor func(data [][]*cell.Value) (warnings []*cell.Warning, err error)

func GenUsingDefaultSheet(getDataFunc func() (data [][]string, err error)) (content []byte, err error) {
	data, err := getDataFunc()
	if err != nil {
		return
	}
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	for index, item := range data {
		p := gconv.SliceAny(item)
		err = f.SetSheetRow(sheet, fmt.Sprintf("A%d", index+1), &p)
		if err != nil {
			return
		}
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return
	}
	content, err = io.ReadAll(buffer)
	return
}
