package grpcgw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	maskCachePrefix = "resp:mask:"
	maskCacheTTL    = 5 * time.Minute
	maskCacheNone   = "__none__" // 哨兵值：表示该 path 无需脱敏，缓存空结果
)

// maskField 描述一个需要脱敏的返回体字段
type maskField struct {
	Kkey string `json:"kkey"`
	Type string `json:"type"`
}

// ResponseHandler 响应脱敏处理器
//
// 依赖：须在 JwtAuth + Header2Context 之后注册，因为它需要 Header 中的 userId。
type ResponseHandler struct {
	Db  *gorm.DB
	Rdb *redis.Client
}

// NewResponseHandler 返回一个响应脱敏处理器，符合 grpcgw.ResponseHandler 签名。
// 须传入 Redis 客户端以启用缓存；传 nil 则每次请求直接查询数据库。
//
// 使用方式：
//
//	grpcgw.WithResponseHandler(middlewares.NewResponseHandler(ctx.RedisClient))
func NewResponseHandler(rdb *redis.Client) func(r *http.Request, db *gorm.DB, data map[string]any) map[string]any {
	return (&ResponseHandler{Rdb: rdb}).ResponseHandler
}

func NewResponseHandler4Api(rdb *redis.Client, db *gorm.DB) *ResponseHandler {
	return &ResponseHandler{Rdb: rdb, Db: db}
}

func (p *ResponseHandler) ResponseHandle(ctx context.Context, userId int64, method, path string, data map[string]any) map[string]any {
	//fields, err := p.getMaskFields(r.Context(), userId, method, path)
	fields, err := p.getMaskFields(ctx, userId, method, path)
	if err != nil {
		logx.Errorf("ResponseHandler: getMaskFields failed, userId=%d, path=%s, err=%v", userId, path, err)
		return data
	}

	if len(fields) == 0 {
		return data
	}

	// 将 data 序列化为 JSON，用 sjson 按路径设置零值，再反序列化回 map。
	// keyPath 全部以 "data." 开头，去除该前缀后即为相对于 data 的路径。
	b, err := json.Marshal(data)
	if err != nil {
		logx.Errorf("ResponseHandler: marshal data failed, err=%v", err)
		return data
	}
	result := string(b)
	for _, f := range fields {
		keyPath := strings.TrimPrefix(f.Kkey, "data.")
		result, err = sjson.Set(result, keyPath, zeroValueFor(f.Type))
		if err != nil {
			logx.Errorf("ResponseHandler: sjson.Set failed, kkey=%s, err=%v", f.Kkey, err)
			continue
		}
	}
	var out map[string]any
	if err = json.Unmarshal([]byte(result), &out); err != nil {
		logx.Errorf("ResponseHandler: unmarshal masked data failed, err=%v", err)
		return data
	}
	return out
}

func ApplyDynamicMask(jsonStr string, rulePath string, maskValue interface{}) string {
	// 1. 将规则路径按点拆分成片段
	segments := strings.Split(rulePath, ".")

	// 2. 递归查找并生成所有真实存在的绝对路径
	absolutePaths := findAbsolutePaths(jsonStr, "", segments)

	// 3. 循环使用 sjson 进行精确修改
	for _, absPath := range absolutePaths {
		jsonStr, _ = sjson.Set(jsonStr, absPath, maskValue)
	}

	return jsonStr
}

// findAbsolutePaths 核心递归函数：将通配符路径转换为绝对路径列表
func findAbsolutePaths(jsonStr string, currentPath string, segments []string) []string {
	// 终止条件：路径片段用完了，说明已经拼凑出一条完整的绝对路径
	if len(segments) == 0 {
		if currentPath == "" {
			return nil
		}
		return []string{currentPath}
	}

	segment := segments[0]
	var paths []string

	if segment == "#" {
		// 【数组通配符处理】
		// 构造向 gjson 查询数组长度的路径 (例如当前在 "users"，则查询 "users.#")
		queryPath := "#"
		if currentPath != "" {
			queryPath = currentPath + ".#"
		}

		// 获取当前层级数组的长度
		count := gjson.Get(jsonStr, queryPath).Int()

		// 发散遍历数组中的每一个元素
		for i := 0; i < int(count); i++ {
			nextPath := strconv.Itoa(i)
			if currentPath != "" {
				nextPath = currentPath + "." + nextPath
			}
			// 拿着剩余的路径片段，递归深入下一层
			subPaths := findAbsolutePaths(jsonStr, nextPath, segments[1:])
			paths = append(paths, subPaths...)
		}
	} else {
		// 【普通对象键名处理】
		nextPath := segment
		if currentPath != "" {
			nextPath = currentPath + "." + nextPath
		}

		// 优化：检查当前这个节点在原 JSON 中是否真的存在
		// 如果不存在，提前剪枝，避免 sjson 无中生有创建出多余的嵌套对象
		if gjson.Get(jsonStr, nextPath).Exists() {
			subPaths := findAbsolutePaths(jsonStr, nextPath, segments[1:])
			paths = append(paths, subPaths...)
		}
	}

	return paths
}

func (p *ResponseHandler) ResponseHandle2Api(ctx context.Context, userId int64, method, path string, data any) any {
	//fields, err := p.getMaskFields(r.Context(), userId, method, path)
	fields, err := p.getMaskFields(ctx, userId, method, path)
	if err != nil {
		logx.Errorf("ResponseHandler: getMaskFields failed, userId=%d, path=%s, err=%v", userId, path, err)
		return data
	}

	if len(fields) == 0 {
		return data
	}

	// 将 data 序列化为 JSON，用 sjson 按路径设置零值，再反序列化回 map。
	// keyPath 全部以 "data." 开头，去除该前缀后即为相对于 data 的路径。
	b, err := json.Marshal(data)
	if err != nil {
		logx.Errorf("ResponseHandler: marshal data failed, err=%v", err)
		return data
	}
	result := string(b)
	for _, f := range fields {
		keyPath := strings.TrimPrefix(f.Kkey, "data.")
		//result, err = sjson.Set(result, keyPath, zeroValueFor(f.Type))
		//result, err = sjson.Set(result, "#(sectionKey==\"totalCostExpenseList\").data.0.tax", zeroValueFor(f.Type))
		result = ApplyDynamicMask(result, keyPath, zeroValueFor(f.Type))
		//if err != nil {
		//	logx.Errorf("ResponseHandler: sjson.Set failed, kkey=%s, err=%v", f.Kkey, err)
		//	continue
		//}
	}
	var out any
	if err = json.Unmarshal([]byte(result), &out); err != nil {
		logx.Errorf("ResponseHandler: unmarshal masked data failed, err=%v", err)
		return data
	}
	return out
}

// ResponseHandler 实现 grpcgw.ResponseHandler 签名，执行基于角色的响应字段脱敏。
//
// 执行链路：
//
//	从请求 Header 获取 userId → 查询该用户角色被排除的返回体字段（Redis 缓存 → DB）→
//	按字段路径（dot-notation）将对应字段替换为类型零值 → 返回修改后的 data map
//
// 入参 data 是响应体 JSON envelope 中的 "data" 字段（已解析为 map[string]any）。
func (p *ResponseHandler) ResponseHandler(r *http.Request, db *gorm.DB, data map[string]any) map[string]any {
	if data == nil {
		return data
	}
	if userIDHeader == "" {
		return data
	}
	userIdStr := r.Header.Get(userIDHeader)
	if userIdStr == "" {
		return data
	}
	userId, err := strconv.ParseInt(userIdStr, 10, 64)
	if err != nil {
		return data
	}
	method := strings.ToUpper(r.Method)
	path := r.URL.Path
	// p.Db 可能为 nil（standalone 场景），此时使用 grpcgw 传入的 db
	if p.Db == nil {
		p.Db = db
	}
	return p.ResponseHandle(r.Context(), userId, method, path, data)
}

// getMaskFields 查询当前用户在指定接口下被排除（脱敏）的返回体字段列表。
// 结果通过 Redis 缓存，TTL = maskCacheTTL。
func (p *ResponseHandler) getMaskFields(ctx context.Context, userId int64, method, path string) ([]maskField, error) {
	cacheKey := fmt.Sprintf("%s%d:%s:%s", maskCachePrefix, userId, method, path)
	// 1. 尝试读缓存
	if p.Rdb != nil {
		val, err := p.Rdb.Get(ctx, cacheKey).Result()
		if err == nil {
			if val == maskCacheNone {
				return nil, nil
			}
			var cached []maskField
			if json.Unmarshal([]byte(val), &cached) == nil {
				return cached, nil
			}
		}
	}

	// 2. 缓存未命中，查询数据库
	//
	// 联查链路:
	//   sys_api (method+path) → sys_api_body → sys_role_api_body_exclude → sys_user_roles (userId)
	//
	// 结果：当前用户的角色被排除的所有字段（kkey, type），按 kkey 去重
	type row struct {
		Kkey string `gorm:"column:kkey"`
		Type string `gorm:"column:type"`
	}
	var rows []row
	err := p.Db.WithContext(ctx).
		Table("sys_api_body sab").
		Select("sab.kkey, sab.type").
		Joins("JOIN sys_role_api_body_exclude srabe ON srabe.api_body_id = sab.id").
		Joins("JOIN sys_user_roles sur ON sur.role_id = srabe.role_id AND sur.deleted_at IS NULL").
		Joins("JOIN sys_api sa ON sa.id = sab.api_id AND sa.method = ? AND sa.path = ?", method, path).
		Where("sur.user_id = ?", userId).
		Group("sab.id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 3. 回写缓存
	if p.Rdb != nil {
		if len(rows) == 0 {
			_ = p.Rdb.Set(ctx, cacheKey, maskCacheNone, maskCacheTTL).Err()
		} else {
			fields := make([]maskField, 0, len(rows))
			for _, r := range rows {
				fields = append(fields, maskField{Kkey: r.Kkey, Type: r.Type})
			}
			if b, err := json.Marshal(fields); err == nil {
				_ = p.Rdb.Set(ctx, cacheKey, string(b), maskCacheTTL).Err()
			}
			return fields, nil
		}
	}

	if len(rows) == 0 {
		return nil, nil
	}
	fields := make([]maskField, 0, len(rows))
	for _, r := range rows {
		fields = append(fields, maskField{Kkey: r.Kkey, Type: r.Type})
	}
	return fields, nil
}

// zeroValueFor 根据字段类型返回对应的零值，用于脱敏替换。
func zeroValueFor(fieldType string) interface{} {
	switch strings.ToLower(fieldType) {
	case "string":
		return "*"
	case "int", "int32", "int64", "integer", "float", "float32", "float64", "number", "double":
		return 0
	case "bool", "boolean":
		return false
	case "array":
		return []interface{}{}
	case "object":
		return map[string]interface{}{}
	default:
		return nil
	}
}
