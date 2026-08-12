package helper

import (
	"fmt"
	"strconv"
)

func Average_StrArr(itemArr []string) (float64, error) {
	valueArr := []float64{}
	for _, item := range itemArr {
		value, err := strconv.ParseFloat(item, 64)
		if err != nil {
			return 0, fmt.Errorf("can not parse item[%s] to float64", item)
		}
		valueArr = append(valueArr, value)
	}

	return Average_Flt64Arr(valueArr), nil
}

func Average_Flt32Arr(itemArr []float32) float64 {
	newItemArr := make([]float64, len(itemArr))
	for index, item := range itemArr {
		newItemArr[index] = float64(item)
	}

	return Average_Flt64Arr(newItemArr)
}

func Average_Flt64Arr(itemArr []float64) float64 {
	total := float64(0)
	count := float64(0)
	for _, item := range itemArr {
		total += item
		if item != 0 {
			count += 1
		}
	}
	if count != 0 {
		return total / count
	}

	return 0
}

func Sum_Flt32Arr(itemArr []float32) float32 {
	sum := float32(0)
	for _, item := range itemArr {
		sum += item
	}

	return sum
}
