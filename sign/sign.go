package sign

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

/*
 * 接口签名验证
 * 1. 从请求参数中提取sign字段
 * 2. 生成待验证的签名
 * 3. 比较签名是否一致
 */

// VerifySign 验证签名
// params: 请求参数（包含sign字段）
// secret: 密钥
func VerifySign(params map[string]interface{}, secret string) bool {
	signVal, ok := params["sign"]
	if !ok {
		return false
	}
	expected := GenerateSign(params, secret)
	return strings.EqualFold(fmt.Sprintf("%v", signVal), expected)
}

// GenerateSign 生成签名
// params: 请求参数
// secret: 密钥
func GenerateSign(params map[string]interface{}, secret string) string {
	flat := make(map[string]string)
	flattenParams("", params, flat)
	// 排序 key
	keys := make([]string, 0, len(flat))
	for k := range flat {
		if k == "sign" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// 拼接字符串
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("&")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(flat[k])
	}
	// 拼接 secret
	signStr := sb.String() + secret
	// 计算 MD5
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}

// 递归展开参数，保证顺序稳定
func flattenParams(prefix string, data interface{}, result map[string]string) {
	switch v := data.(type) {
	case map[string]interface{}:
		// 遍历 map，按 key 排序
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			newKey := k
			if prefix != "" {
				newKey = prefix + "." + k
			}
			flattenParams(newKey, v[k], result)
		}
	case []interface{}:
		for i, item := range v {
			newKey := prefix + "[" + strconv.Itoa(i) + "]"
			flattenParams(newKey, item, result)
		}
	default:
		if v != nil {
			result[prefix] = fmt.Sprintf("%v", v)
		}
	}
}
