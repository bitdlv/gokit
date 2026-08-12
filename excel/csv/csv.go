package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
)

// ReadCsvFile 读取 CSV 文件
func ReadCsvFile(csvPath string, first bool) ([][]string, error) {
	var rows [][]string
	csvFile, err := os.Open(csvPath)
	if err != nil {
		return rows, err
	}
	csvReader := csv.NewReader(csvFile)
	if !first {
		_, err := csvReader.Read()
		if err != nil {
			fmt.Println(err)
			return rows, err
		}
	}
	rows, err = csvReader.ReadAll()
	if err != nil {
		fmt.Println(err)
		return rows, err
	}
	return rows, nil
}

var UTF8BOM = []byte{239, 187, 191}

func hasBOM(in []byte) bool {
	return bytes.HasPrefix(in, UTF8BOM)
}

func stripBOM(in []byte) []byte {
	return bytes.TrimPrefix(in, UTF8BOM)
}
