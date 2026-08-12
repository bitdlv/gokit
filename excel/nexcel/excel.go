package nexcel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/bitdlv/gokit/excel/nexcel/data"
	"github.com/gogf/gf/v2/util/gconv"
	"google.golang.org/appengine/log"

	"github.com/xuri/excelize/v2"
)

const (
	ExcelModeRow = iota + 1 // 按行读/写
	ExcelModeCol            // 按列读/写
)

func WithValidator(validator Handler) func(*Excel) {
	return func(e *Excel) {
		e.validator = validator
	}
}

func WithModifier(modifier Handler) func(*Excel) {
	return func(e *Excel) {
		e.modifier = modifier
	}
}

func NewExcel(processor Handler, opts ...func(*Excel)) *Excel {
	e := &Excel{
		mode:      ExcelModeRow,
		skip:      1,
		workerNum: 100,
		processor: processor,
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

type Handler func(group *data.Group) (warns []*data.Warning)

type Excel struct {
	mode      int
	skip      int
	workerNum int
	validator Handler
	processor Handler
	modifier  Handler
}

func (e *Excel) LoadRowBySheetIndex(content []byte, sheetIndex int, endx int) (warns []*data.Warning) {
	e.mode = ExcelModeRow
	warns = e.readExcel(context.Background(), content, sheetIndex, endx, 0)
	return
}

func (e *Excel) readExcel(ctx context.Context, content []byte, sheetIndex int, endx, endy int) (allWarns []*data.Warning) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	f, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return
	}
	defer f.Close()

	sheetName := f.GetSheetName(sheetIndex)
	header := e.getHeader(f, sheetName, endx, endy)
	oriChan := make(chan *data.Group, 1)
	switch e.mode {
	case ExcelModeRow:
		go e.getRows(ctx, f, sheetName, header, oriChan)
	default:
		close(oriChan)
		panic(errors.New("unsupported"))
	}

	allWarns = make([]*data.Warning, 0)
	warnLock := sync.Mutex{}
	addWarns := func(warns []*data.Warning) {
		warnLock.Lock()
		defer warnLock.Unlock()
		allWarns = append(allWarns, warns...)
	}

	wg := sync.WaitGroup{}

	// 修改
	var modifiedChan chan *data.Group
	if e.modifier != nil {
		wg.Add(1)
		modifiedChan = make(chan *data.Group, 1)
		var wgLocal sync.WaitGroup
		for i := 0; i < e.workerNum; i++ {
			wgLocal.Add(1)
			go e.process2(ctx, &wgLocal, oriChan, modifiedChan, addWarns, e.modifier)
		}
		go func() {
			wgLocal.Wait()
			close(modifiedChan)
			wg.Done()
		}()
	} else {
		modifiedChan = oriChan
	}

	// 校验
	var validatedChan chan *data.Group
	if e.validator != nil {
		wg.Add(1)
		validatedChan = make(chan *data.Group, 1)
		var wgLocal sync.WaitGroup
		for i := 0; i < e.workerNum; i++ {
			wgLocal.Add(1)
			go e.process2(ctx, &wgLocal, modifiedChan, validatedChan, addWarns, e.validator)
		}
		go func() {
			wgLocal.Wait()
			close(validatedChan)
			wg.Done()
		}()
	} else {
		validatedChan = modifiedChan
	}

	// 处理
	for i := 0; i < e.workerNum; i++ {
		wg.Add(1)
		go e.process2(ctx, &wg, validatedChan, nil, addWarns, e.processor)
	}

	wg.Wait()

	return
}

func (e *Excel) process2(
	ctx context.Context,
	wg *sync.WaitGroup,
	in <-chan *data.Group,
	out chan<- *data.Group,
	addWarns func(warns []*data.Warning),
	processor func(group *data.Group) []*data.Warning,
) {
	defer func() {
		wg.Done()
	}()
validateLoop:
	for {
		select {
		case <-ctx.Done():
			break validateLoop
		case group, ok := <-in:
			if !ok {
				break validateLoop
			}
			warns := processor(group)
			if len(warns) > 0 {
				addWarns(warns)
			} else if out != nil {
				out <- group
			}
		}
	}
}

func (e *Excel) process(
	ctx context.Context,
	wg *sync.WaitGroup,
	in <-chan *data.Group,
	addWarns func(warns []*data.Warning),
	processor func(group *data.Group) []*data.Warning,
	end bool,
) (out chan *data.Group) {
	out = make(chan *data.Group)
	for i := 0; i < e.workerNum; i++ {
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
			}()
		validateLoop:
			for {
				select {
				case <-ctx.Done():
					break validateLoop
				case group, ok := <-in:
					if !ok {
						break validateLoop
					}
					warns := processor(group)
					if len(warns) > 0 {
						addWarns(warns)
					} else if !end {
						out <- group
					}
				}
			}
		}()
	}
	return
}

func (e *Excel) getHeader(f *excelize.File, name string, endx int, endy int) (result []*data.Value) {
	switch e.mode {
	case ExcelModeRow:
		if endx <= 0 {
			panic("invalidate x")
		}
		result = make([]*data.Value, 0, endx)
		for x := 1; x <= endx; x++ {
			v, err := f.GetCellValue(name, Cell(x, e.skip))
			if err != nil {
				panic(err)
			}
			result = append(result, data.NewValue(x, e.skip, v))
		}
	default:
		panic(errors.New("unsupported"))
	}

	return
}

func (e *Excel) getRows(ctx context.Context, f *excelize.File, name string, header []*data.Value, rowChan chan<- *data.Group) {
	defer close(rowChan)
	if len(header) <= 0 {
		panic("invalidate headers")
	}
	rows, err := f.Rows(name)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	rowIndex := 0

rowLoop:
	for rows.Next() {
		rowIndex++
		if rowIndex <= e.skip {
			continue
		}
		select {
		case <-ctx.Done():
			break rowLoop
		default:
			r, err := rows.Columns()
			if err != nil {
				panic(err)
			}

			r = append(r, make([]string, len(header)-len(r))...)

			row := make([]*data.Value, 0, len(r))
			for index, c := range r {
				row = append(row, data.NewValue(index+1, rowIndex, c))
			}
			if row[0].IsEmpty() {
				break rowLoop
			}
			g := &data.Group{
				Header: header,
				Cells:  row,
			}
			rowChan <- g
		}
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

func GenUsingDefaultSheet(getDataFunc func() (data [][]string, err error), num int) (content []byte, err error) {
	if num <= 0 {
		num = 100
	}
	type work struct {
		axis string
		data []any
	}
	var wg sync.WaitGroup
	writer := func(file *excelize.File, sheetName string, in <-chan *work) {
		defer wg.Done()
		for w := range in {
			err = file.SetSheetRow(sheetName, w.axis, &w.data)
			if err != nil {
				log.Warningf(context.Background(), "set row field: %s", err.Error())
			}
		}
	}

	list, err := getDataFunc()
	if err != nil {
		return
	}
	f := excelize.NewFile()
	defer f.Close()

	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	ch := make(chan *work)

	for i := 0; i < num; i++ {
		wg.Add(1)
		go writer(f, sheet, ch)
	}

	for index, item := range list {
		p := gconv.SliceAny(item)
		ch <- &work{
			axis: fmt.Sprintf("A%d", index+1),
			data: p,
		}
	}

	close(ch)
	wg.Wait()

	buffer, err := f.WriteToBuffer()
	if err != nil {
		return
	}
	content, err = io.ReadAll(buffer)
	return
}
