package util

import (
	"archive/zip"
	"bytes"
	"github.com/tealeg/xlsx/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
)

type Excel struct {
	Titles []string
	File   *xlsx.File
	Sheet  *xlsx.Sheet
}

func (excel *Excel) BuildFile() {
	excel.File = xlsx.NewFile()
	sheet, err := excel.File.AddSheet("Sheet1")

	if err != nil {
		logx.Error("build excel fail please check")
	}

	excel.Sheet = sheet
}

func (excel *Excel) BuildTitle(titles []string) {
	row := excel.Sheet.AddRow()
	excel.Titles = titles
	for _, title := range excel.Titles {
		cell := row.AddCell()
		cell.Value = title
	}
}

func (excel *Excel) WriteBody(datas []string) {
	row := excel.Sheet.AddRow()
	for _, data := range datas {
		cell := row.AddCell()
		cell.Value = data
	}
}

func (excel *Excel) Write(writer io.Writer) {
	excel.File.Write(writer)
}

// GetZipBytes get the File zip bytes
func (excel *Excel) GetZipBytes() ([]byte, error) {
	parts, err := excel.File.MarshallParts()
	if err != nil {
		return nil, err
	}
	bts := bytes.NewBuffer(make([]byte, 0))
	zipWriter := zip.NewWriter(bts)
	for partName, part := range parts {
		w, err := zipWriter.Create(partName)
		if err != nil {
			return nil, err
		}
		_, err = w.Write([]byte(part))
		if err != nil {
			return nil, err
		}
	}
	return bts.Bytes(), nil
}
