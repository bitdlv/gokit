package errx

import (
	"google.golang.org/grpc/codes"
	"net/http"
)

// 我们对官方的错误码做一个补充
const (
	// General Errors
	UNKNOWN_ERROR         codes.Code = 100000 // 未知错误
	INVALID_REQUEST       codes.Code = 100001 // 请求格式、参数或结构无效
	INTERNAL_SERVER_ERROR codes.Code = 100002 // 服务器内部错误
	RESOURCE_NOT_FOUND    codes.Code = 100003 // 请求的资源未找到
	PERMISSION_DENIED     codes.Code = 100004 // 用户没有执行操作的权限

	// Database Errors
	DB_CONNECTION_FAILED codes.Code = 100100 // 数据库连接失败
	DB_QUERY_FAILED      codes.Code = 100101 // 数据库查询错误

	// Authentication and Authorization Errors

	AUTH_FAILED               codes.Code = 100200 // 认证失败，例如密码错误
	TOKEN_EXPIRED             codes.Code = 100201 // 身份验证令牌已过期
	TOKEN_INVALID             codes.Code = 100202 // 身份验证令牌无效或格式错误
	AUTHORIZATION_REQUIRED    codes.Code = 100203 // 需要认证才能访问资源
	AUTH_ACCOUNT_NOT_ACTIVITE codes.Code = 100204 // 用户未激活
	AUTH_ACCOUNT_NOT_ENABLE   codes.Code = 100205 // 用户未启用
	AUTH_ACCOUNT_EXPIRED      codes.Code = 100206 // 用户已失效

	// Rate Limiting and Throttling

	RATE_LIMIT_EXCEEDED codes.Code = 100300 // 用户已超出请求速率限制

	// Business Logic Errors (generic)

	OPERATION_FAILED codes.Code = 100400 // 业务逻辑操作失败，不具体到某一原因

	// Other Errors

	THIRD_PARTY_SERVICE_FAILURE codes.Code = 100500 // 第三方服务调用失败或超时
)

var message = map[codes.Code]string{
	UNKNOWN_ERROR:               "未知错误",
	INVALID_REQUEST:             "请求格式、参数或结构无效",
	INTERNAL_SERVER_ERROR:       "服务器内部错误",
	RESOURCE_NOT_FOUND:          "请求的资源未找到",
	PERMISSION_DENIED:           "用户没有执行操作的权限",
	DB_CONNECTION_FAILED:        "数据库连接失败",
	DB_QUERY_FAILED:             "数据库查询错误",
	AUTH_FAILED:                 "认证失败，例如密码错误",
	TOKEN_EXPIRED:               "身份验证令牌已过期",
	TOKEN_INVALID:               "身份验证令牌无效或格式错误",
	AUTHORIZATION_REQUIRED:      "需要认证才能访问资源",
	AUTH_ACCOUNT_NOT_ACTIVITE:   "用户未激活",
	AUTH_ACCOUNT_NOT_ENABLE:     "用户未启用",
	AUTH_ACCOUNT_EXPIRED:        "用户已失效",
	RATE_LIMIT_EXCEEDED:         "用户已超出请求速率限制",
	OPERATION_FAILED:            "业务逻辑操作失败，不具体到某一原因",
	THIRD_PARTY_SERVICE_FAILURE: "第三方服务调用失败或超时",
}

func MapErrMsg(errCode codes.Code) string {
	if msg, ok := message[errCode]; ok {
		return msg
	} else {
		return "服务器开小差啦,稍后再来试一试"
	}
}

func IsCodeErr(errCode codes.Code) bool {
	if _, ok := message[errCode]; ok {
		return true
	} else {
		return false
	}
}

func ToHttpStatus(code codes.Code) int {
	//兼容原生错误码
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.InvalidArgument, codes.AlreadyExists, INVALID_REQUEST:
		return http.StatusBadRequest
	case codes.NotFound, RESOURCE_NOT_FOUND:
		return http.StatusNotFound
	case codes.PermissionDenied, PERMISSION_DENIED:
		return http.StatusForbidden
	case codes.Unauthenticated, AUTH_FAILED, TOKEN_EXPIRED, TOKEN_INVALID, AUTHORIZATION_REQUIRED:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
