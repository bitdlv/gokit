package testconfig

import (
	"github.com/zeromicro/go-zero/core/logx"
)

func InitForTest() {
	logx.SetLevel(logx.ErrorLevel)
}

// ErrorFormatWithValue 需要准备四个参数：测试用例数据，测试函数返回的某个结果名（result.xxx），返回值，预期值
var ErrorFormatWithValue string = "\n testcase: %s \n\t %s: [%v] \t expect: [%v]"

// ErrorFormat 需要准备三个参数：测试用例数据，错误自描述，函数返回错误内容（err.Error)
var ErrorFormat string = "\n testcase: %s \n\t %s, err: %s"
