package helper

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/zeromicro/go-zero/core/logx"
)

// GeExpressionComputeVal: 根据参数列表和表达式计算出结果
/*
  @param expression 表达式 eg："? + ? * (? - ?)"
  @param paramArr 参数列表 eg：["10.2", 2, 3, 4]
*/
func GeExpressionComputeVal(expression string, paramArr []interface{}) (result interface{}, err error) {
	functionName := "[GeExpressionComputeVal]"
	logx.Debugf("%s params: %s, %v", functionName, expression, paramArr)

	if strings.Count(expression, "?") != len(paramArr) {
		errMsg := fmt.Sprintf("%s 参数校验不通过，表达式所需参数个数与参数列表长度不一致", functionName)
		logx.Errorf("%s, expression: %s, paramArr: %v", errMsg, expression, paramArr)
		return result, errors.New(errMsg)
	}

	args := make([]interface{}, 0)
	for _, param := range paramArr {
		switch v := param.(type) {
		case int:
			args = append(args, v)
		case float64:
			args = append(args, v)
		case string:
			args = append(args, v)
		default:
			args = append(args, fmt.Sprintf("%v", v))
		}
	}
	expr := strings.ReplaceAll(expression, "?", "%v")
	getExpression := fmt.Sprintf(expr, args...)

	evaluableExpression, err := govaluate.NewEvaluableExpression(getExpression)
	if err != nil {
		errMsg := fmt.Sprintf("%s failed to govaluate.NewEvaluableExpression()", functionName)
		logx.Errorf("%s, param: %s, err: %s", errMsg, getExpression, err.Error())
		return result, errors.New(errMsg)
	}
	// fmt.Printf("evaluableExpression: %v", evaluableExpression)

	// 计算出表达式的结果
	result, err = evaluableExpression.Evaluate(nil)
	if err != nil {
		errMsg := fmt.Sprintf("%s failed to evaluableExpression.Evaluate()", functionName)
		logx.Errorf("%s, err: %s", errMsg, err.Error())
		return result, errors.New(errMsg)
	}

	logx.Infof("%s Success!", functionName)
	return
}

// ExpressionToPlaceholderExpression: 将表达式转变为占位符表达式
// eg："A + B * (3 - 4)" —> { "? + ? * (? - ?)" , [A, B, 3, 4] , false }
func ExpressionToPlaceholderExpression(expression string) (placeholderExpression string, variableArr []string, noCalculate bool) {
	functionName := "[ExpressionToPlaceholderExpression]"
	logx.Debugf("%s params: %s", functionName, expression)

	specialSymbolIndexArr := []int{}
	for index, val := range expression {
		switch string(val) {
		case "+", "-", "*", "/", "(", ")":
			specialSymbolIndexArr = append(specialSymbolIndexArr, index)
		}
	}
	// fmt.Println("specialSymbolIndexArr: ", specialSymbolIndexArr)

	if len(specialSymbolIndexArr) == 0 {
		noCalculate = true
		variableArr = append(variableArr, expression)
		return
	}
	leftIndex := 0
	for _, rightIndex := range specialSymbolIndexArr {
		if rightIndex == 0 {
			leftIndex += 1
			continue
		}
		if rightIndex == leftIndex {
			leftIndex += 1
			continue
		}

		// fmt.Printf("leftIndex: %d, rightIndex: %d \n", leftIndex, rightIndex)
		variable := strings.Trim(expression[leftIndex:rightIndex], " ")
		if len(variable) != 0 {
			variableArr = append(variableArr, variable)
		}
		leftIndex = rightIndex + 1
	}
	if len(expression)-1 > specialSymbolIndexArr[len(specialSymbolIndexArr)-1] {
		leftIndex := specialSymbolIndexArr[len(specialSymbolIndexArr)-1] + 1
		rightIndex := len(expression)
		variable := strings.Trim(expression[leftIndex:rightIndex], " ")
		if len(variable) != 0 {
			variableArr = append(variableArr, variable)
		}
	}
	// fmt.Println("variableArr: ", variableArr)

	for _, variable := range variableArr {
		expression = strings.Replace(expression, variable, "?", 1)
	}
	// fmt.Println("placeholderExpression: ", expression)

	placeholderExpression = expression

	logx.Infof("%s Success!", functionName)
	return
}
