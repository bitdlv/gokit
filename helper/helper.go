package helper

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/bitdlv/gokit/nacos"
	"github.com/mattn/go-runewidth"
	"github.com/zeromicro/go-zero/core/conf"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bitdlv/gokit/pb"
	"github.com/gogf/gf/v2/util/gconv"
)

// GetRandString 获取长度为len*2的16进制的随机字符串
func GetRandString(len int) string {
	bt := make([]byte, len)
	rand.Read(bt)
	return hex.EncodeToString(bt)
}

// GetField 尝试获取结构体中结构体字段的字段，可以设置默认值
func GetField[T any](s any, field string, def ...T) (result T) {
	if len(def) > 0 {
		result = def[0]
	}

	fields := strings.Split(field, ".")
	value := reflect.ValueOf(s)
	for _, f := range fields {
		if !value.IsValid() || value.IsZero() {
			return
		}
		if value.Kind() == reflect.Pointer {
			value = value.Elem()
		}
		value = value.FieldByName(f)
	}

	if value.Kind() == reflect.Invalid {
		return *new(T)
	} else {
		return value.Interface().(T)
	}
}

// Ternary 三元运算(￣▽￣)"
func Ternary[T any](e bool, t T, f T) T {
	if e {
		return t
	} else {
		return f
	}
}

func Contains[T comparable](finds any, data []T) (found bool) {
	switch x := finds.(type) {
	case []T:
		for _, find := range x {
			for _, item := range data {
				if item == find {
					found = true
					return
				}
			}
		}
	case T:
		for _, item := range data {
			if item == x {
				found = true
				return
			}
		}
	default:
		panic(errors.New("type is mismatched"))
	}

	return
}

// Pluck 取对象数组中的字段，作为新数组，支持同时取多个字段，用法看测试
func Pluck(data any, container any) {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice && dataValue.Kind() != reflect.Array && dataValue.Kind() != reflect.Map {
		panic(errors.New("only support slice and array"))
	}
	cValue := reflect.ValueOf(container)
	if cValue.Kind() != reflect.Pointer {
		panic(errors.New("container only support ptr"))
	}
	cValue = cValue.Elem()

	myAppend := func(s reflect.Value, field reflect.Value, fieldName string) {
		if s.IsNil() {
			s.Set(reflect.MakeSlice(s.Type(), 0, dataValue.Len()))
		}

		if field.Kind() == reflect.Pointer {
			field = field.Elem()
		}
		field = field.FieldByName(fieldName)
		if field.IsValid() && field.Type() == s.Type().Elem() {
			s.Set(reflect.Append(s, field))
		}
	}

	switch dataValue.Kind() {
	case reflect.Map:
		item := dataValue.MapRange()
		for item.Next() {
			for j := 0; j < cValue.NumField(); j++ {
				myAppend(cValue.Field(j), item.Value(), cValue.Type().Field(j).Name)
			}
		}
	default:
		for i := 0; i < dataValue.Len(); i++ {
			for j := 0; j < cValue.NumField(); j++ {
				myAppend(cValue.Field(j), dataValue.Index(i), cValue.Type().Field(j).Name)
			}
		}
	}

	return
}

// Json2Map json字符串转map，如果有错误，返回nil
func Json2Map(jsonStr string) (result map[string]interface{}) {
	json.Unmarshal([]byte(jsonStr), &result)
	return
}

// Map2Json map装json字符串，如果有错误，返回“”
func Map2Json(m map[string]interface{}) (str string) {
	result, _ := json.Marshal(m)
	return string(result)
}

type IntBase interface {
	~int | ~int64 | ~int32
}

func FormatTs[T IntBase](unix T, format string) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(int64(unix), 0).Format(format)
}

// JoinField 将结构体slice每个元素的相同字段拼接成字符串
func JoinField(slice any, field string, sep string) (result string) {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return
	}
	if v.Len() <= 0 {
		return
	}

	var arr []string
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Pointer {
			item = item.Elem()
		}
		if item.Kind() != reflect.Struct {
			return
		}
		field := item.FieldByName(field)
		if !field.IsValid() {
			continue
		}
		arr = append(arr, gconv.String(field.Interface()))
	}

	return strings.Join(arr, sep)
}

func Map2Pb(val any) (m map[string]*pb.Value, err error) {
	v := reflect.ValueOf(val)
	m = make(map[string]*pb.Value, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		key := iter.Key()
		if key.Kind() != reflect.String {
			err = errors.New("map key can only be string")
			return
		}
		m[key.String()], err = Any2PbValue(iter.Value().Interface())
		if err != nil {
			return
		}
	}

	return
}

func Any2PbValue(val any) (value *pb.Value, err error) {
	value = &pb.Value{}
	switch x := val.(type) {
	case int32:
		value.Kind = &pb.Value_Int32Val{
			Int32Val: x,
		}
	case int64:
		value.Kind = &pb.Value_Int64Val{
			Int64Val: x,
		}
	case string:
		value.Kind = &pb.Value_StrVal{
			StrVal: x,
		}
	case int:
		switch strconv.IntSize {
		case 32:
			value.Kind = &pb.Value_Int32Val{
				Int32Val: int32(x),
			}
		case 64:
			value.Kind = &pb.Value_Int64Val{
				Int64Val: int64(x),
			}
		}
	default:
		v := reflect.ValueOf(val)
		switch v.Kind() {
		case reflect.Map:
			var m map[string]*pb.Value
			m, err = Map2Pb(v.Interface())
			if err != nil {
				return
			}
			value.Kind = &pb.Value_MapVal{
				MapVal: &pb.Map{
					Fields: m,
				},
			}
		case reflect.Slice:
			s := &pb.Value_ListVal{
				ListVal: &pb.List{
					List: make([]*pb.Value, 0, v.Len()),
				},
			}
			for i := 0; i < v.Len(); i++ {
				item := v.Index(i)
				var pbValue *pb.Value
				pbValue, err = Any2PbValue(item.Interface())
				if err != nil {
					return
				}
				s.ListVal.List = append(s.ListVal.List, pbValue)
			}
			value.Kind = s
		}
	}
	return
}

func PbValue2Any(pbValue *pb.Value) (v any, err error) {
	switch x := pbValue.Kind.(type) {
	case *pb.Value_Int32Val:
		v = x.Int32Val
	case *pb.Value_Int64Val:
		v = x.Int64Val
	case *pb.Value_MapVal:
		v, err = PbMap2MapStrAny(x.MapVal)
	case *pb.Value_ListVal:
		v, err = PbList2SliceAny(x.ListVal)
	case *pb.Value_StrVal:
		v = x.StrVal
	}
	return
}

func PbList2SliceAny(pbList *pb.List) (s []any, err error) {
	tmp := make([]any, len(pbList.List))
	for _, item := range pbList.List {
		var res any
		res, err = PbValue2Any(item)
		if err != nil {
			break
		}
		tmp = append(tmp, res)
	}

	return
}

func PbMap2MapStrAny(pbMap *pb.Map) (m map[string]any, err error) {
	m = make(map[string]any, len(pbMap.Fields))
	for key, item := range pbMap.Fields {
		m[key], err = PbValue2Any(item)
		if err != nil {
			return
		}
	}

	return
}

func FindSlice[T any](s []T, finder func(item T) bool) (result T, ok bool) {
	for _, item := range s {
		if finder(item) {
			return item, true
		}
	}

	return
}

func Diff[T comparable](a []T, b []T) (result []T) {
	result = make([]T, 0, Max(len(a), len(b)))
	for _, aItem := range a {
		find := false
		for _, bItem := range b {
			if aItem == bItem {
				find = true
				break
			}
		}
		if !find {
			result = append(result, aItem)
		}
	}
	return
}

type Number interface {
	~int | ~int32 | ~int64 |
		~uint | ~uint32 | ~uint64 |
		~float32 | ~float64
}

func Max[T Number](list ...T) (m T) {
	for i, item := range list {
		if i == 0 {
			m = item
			continue
		}

		if m < item {
			m = item
		}
	}

	return
}

func Timeout(f func(ctx context.Context) error, t time.Duration) (err error) {
	c := make(chan struct{})
	var chanClosed atomic.Bool
	ctx, cancel := context.WithTimeout(context.Background(), t)

	defer func() {
		if !chanClosed.Load() {
			chanClosed.Store(true)
			close(c)
			cancel()
		}
	}()

	go func() {
		err = f(ctx)
		if !chanClosed.Load() {
			chanClosed.Store(true)
			close(c)
			cancel()
		}
	}()

	select {
	case <-c:
		return
	case <-ctx.Done():
		return errors.New("timeout")
	}
}

func BatchProcess[T comparable](s []T, batchSize int, f func(s []T) error) error {
	for i := 0; i < len(s); i += batchSize {
		end := i + batchSize
		if end > len(s) {
			end = len(s)
		}
		batchSlice := s[i:end]
		if err := f(batchSlice); err != nil {
			return err
		}
	}

	return nil
}

func AsyncBatchProcess[T comparable](s []T, batchSize int, f func(s []T) error, max int) (err error) {
	if max <= 0 {
		return errors.New("num of worker can not lte 0")
	}

	var (
		wg       sync.WaitGroup
		errStore atomic.Value
		dataChan = make(chan []T, 100)
	)

	worker := func() {
		defer wg.Done()
		for {
			if errStore.Load() != nil {
				return
			}
			select {
			case data, ok := <-dataChan:
				if !ok {
					return
				}
				err := f(data)
				if err != nil {
					if errStore.Load() == nil {
						errStore.Store(err)
					}
					return
				}
			}
		}
	}

	for i := 0; i < max; i++ {
		wg.Add(1)
		go worker()
	}

	for i := 0; i < len(s); i += batchSize {
		if errStore.Load() != nil {
			break
		}
		end := i + batchSize
		if end > len(s) {
			end = len(s)
		}
		batch := s[i:end]
		dataChan <- batch
	}

	close(dataChan)
	wg.Wait()

	if err := errStore.Load(); err != nil {
		return err.(error)
	}
	return
}

// TruncateString 将字符串截取为指定长度（以字符为单位），确保不会截断中文字符。
func TruncateString(s string, length int) string {
	if length <= 0 {
		return ""
	}

	var result string
	count := 0
	for _, runeValue := range s {
		runeWidth := runewidth.RuneWidth(runeValue)
		if count+runeWidth > length {
			break
		}
		result += string(runeValue)
		count += runeWidth
	}

	return result
}

func Time2LocalString(source time.Time) string {

	// 使用 .Local() 方法将时间对象转换为本地时区
	localTime := source.Local()

	// 格式化为字符串
	formattedTime := localTime.Format("2006-01-02 15:04:05")

	// 输出结果
	return formattedTime

}

func Filter[T any](src []T, f func(T) bool) (result []T) {
	result = make([]T, 0, len(src))
	for _, item := range src {
		if f(item) {
			result = append(result, item)
		}
	}

	return
}

func Avg[T Number](data []T) (result T) {
	if len(data) == 0 {
		return T(0)
	}

	var sum T
	for _, item := range data {
		sum += item
	}

	return sum / T(len(data))
}

func AdvPluck[T any](data any, field string) (result []T) {
	dataValue := reflect.ValueOf(data)
	if dataValue.Kind() != reflect.Slice {
		panic(errors.New("slice only"))
	}

	result = make([]T, 0, dataValue.Len())
	for i := 0; i < dataValue.Len(); i++ {
		item := dataValue.Index(i)
		if item.Kind() == reflect.Pointer {
			item = item.Elem()
		}
		result = append(result, item.FieldByName(field).Interface().(T))
	}

	return
}

func MapSlice[T any, K any](data []T, f func(item T) (K, error)) (result []K, err error) {
	result = make([]K, 0, len(data))
	for _, item := range data {
		var r K
		r, err = f(item)
		if err != nil {
			return
		}
		result = append(result, r)
	}

	return
}

func LoadConf(config string, c interface{}) error {
	dir, _ := os.Getwd()
	configPath := dir + "/etc/" + config + ".yaml"
	_, err := os.Stat(configPath)
	if !os.IsNotExist(err) {
		configFile := flag.String("f", configPath, "the config file")
		flag.Parse()
		conf.MustLoad(*configFile, c)
		return nil
	} else {
		err := nacos.LoadNsConfig(config, c)
		if err != nil {
			fmt.Printf("加载Nacos配置失败%v", err)
		}
		return err
	}
}
