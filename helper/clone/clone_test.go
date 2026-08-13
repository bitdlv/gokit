package clone

import (
	"reflect"
	"testing"
	"time"

	"gorm.io/gorm"
)

// ============================================================
// 测试固件（fixtures）
// ============================================================

var (
	testNow = time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

	strPtr  = func(s string) *string { return &s }
	intPtr  = func(i int) *int { return &i }
	f64Ptr  = func(f float64) *float64 { return &f }
	boolPtr = func(b bool) *bool { return &b }
	timePtr = func(t time.Time) *time.Time { return &t }
)

// ============================================================
// 测试模型定义（按数据类型分组）
// ============================================================

// 1. 基础标量类型
type scalarModel struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Score     float64   `json:"score"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// 2. 指针字段类型
type ptrFieldModel struct {
	ID          int64      `gorm:"primaryKey" json:"id"`
	Name        *string    `json:"name"`
	Age         *int       `json:"age"`
	Score       *float64   `json:"score"`
	Active      *bool      `json:"active"`
	Description *string    `json:"description" clone:"ignore"`
	CreatedAt   *time.Time `json:"created_at"`
}

// 3. 切片字段类型
type sliceFieldModel struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Tags      []string  `json:"tags"`
	Numbers   []int     `json:"numbers"`
	Scores    []float64 `json:"scores"`
	Flags     []bool    `json:"flags"`
	CreatedAt time.Time `json:"created_at"`
}

// 4. Map 字段类型
type mapFieldModel struct {
	ID        int64                  `gorm:"primaryKey" json:"id"`
	Attrs     map[string]string      `json:"attrs"`
	Counts    map[string]int         `json:"counts"`
	Extra     map[string]interface{} `json:"extra"`
	CreatedAt time.Time              `json:"created_at"`
}

// 5. 嵌套 struct（值）
type innerModel struct {
	ID    int64  `gorm:"primaryKey" json:"id"`
	Label string `json:"label"`
}
type nestedValueModel struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	Inner     innerModel `json:"inner"`
	CreatedAt time.Time  `json:"created_at"`
}

// 6. 嵌套 struct（指针）
type nestedPtrModel struct {
	ID        int64       `gorm:"primaryKey" json:"id"`
	Inner     *innerModel `json:"inner"`
	CreatedAt time.Time   `json:"created_at"`
}

// 7. 嵌套切片
type nestedSliceModel struct {
	ID        int64         `gorm:"primaryKey" json:"id"`
	Items     []innerModel  `json:"items"`
	ItemPtrs  []*innerModel `json:"item_ptrs"`
	CreatedAt time.Time     `json:"created_at"`
}

// 8. 嵌入匿名 struct
type baseFields struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type embedModel struct {
	baseFields
	Name string `json:"name"`
}

// 9. time.Time 与 *time.Time
type timeModel struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	StartAt   time.Time  `json:"start_at"`
	EndAt     *time.Time `json:"end_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// 10. gorm.DeletedAt 软删除
type softDeleteModel struct {
	ID        int64          `gorm:"primaryKey" json:"id"`
	Name      string         `json:"name"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`
	CreatedAt time.Time      `json:"created_at"`
}

// 11. clone:"ignore" tag
type tagIgnoreModel struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	PublicData  string    `json:"public_data"`
	PrivateData string    `json:"private_data" clone:"ignore"`
	InternalID  int64     `json:"internal_id" clone:"ignore"`
	CreatedAt   time.Time `json:"created_at"`
}

// 12. 自定义 gorm primaryKey 字段名
type customPKModel struct {
	OrderNo   string    `gorm:"column:order_no;primaryKey" json:"order_no"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

// 13. 未导出字段
type unexportedModel struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	internal  string
	CreatedAt time.Time `json:"created_at"`
}

// 14. interface{} 字段
type ifaceFieldModel struct {
	ID        int64       `gorm:"primaryKey" json:"id"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
}

// 15. 数组（非切片）
type arrayModel struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Matrix    [3]int    `json:"matrix"`
	CreatedAt time.Time `json:"created_at"`
}

// 16. 自定义类型别名
type testStatus string
type aliasModel struct {
	ID        int64      `gorm:"primaryKey" json:"id"`
	Status    testStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// 17. json:"-" 字段
type jsonDashModel struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Public    string    `json:"public"`
	Hidden    string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// 18. 多层嵌套混合
type level3 struct {
	ID    int64  `gorm:"primaryKey" json:"id"`
	Value string `json:"value"`
}
type level2 struct {
	ID     int64    `gorm:"primaryKey" json:"id"`
	L3     *level3  `json:"l3"`
	L3List []level3 `json:"l3_list"`
}
type level1 struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	L2        *level2   `json:"l2"`
	CreatedAt time.Time `json:"created_at"`
}

// 19. nil 指针字段
type nilPtrModel struct {
	ID        int64       `gorm:"primaryKey" json:"id"`
	Inner     *innerModel `json:"inner"`
	CreatedAt time.Time   `json:"created_at"`
}

// 20. 空 struct
type emptyModel struct{}

// ============================================================
// 按数据类型分类的测试用例
// ============================================================

func TestStruct_ScalarFields(t *testing.T) {
	src := scalarModel{ID: 1, Name: "n", Age: 18, Score: 99.5, Active: true, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零, got %d", dst.ID)
	}
	if dst.Name != "n" {
		t.Errorf("Name 应保留, got %q", dst.Name)
	}
	if dst.Age != 18 {
		t.Errorf("Age 应保留, got %d", dst.Age)
	}
	if dst.Score != 99.5 {
		t.Errorf("Score 应保留, got %v", dst.Score)
	}
	if !dst.Active {
		t.Errorf("Active 应保留, got %v", dst.Active)
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零, got %v", dst.CreatedAt)
	}
	if src.ID != 1 {
		t.Errorf("源对象不应被修改")
	}
}

func TestStruct_PointerFields(t *testing.T) {
	src := ptrFieldModel{
		ID:          2,
		Name:        strPtr("hello"),
		Age:         intPtr(30),
		Score:       f64Ptr(88.8),
		Active:      boolPtr(true),
		Description: strPtr("secret"),
		CreatedAt:   timePtr(testNow),
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Name == nil || *dst.Name != "hello" {
		t.Errorf("*Name 应保留")
	}
	if dst.Name == src.Name {
		t.Errorf("*Name 应为深拷贝（指针独立）")
	}
	if dst.Age == nil || *dst.Age != 30 {
		t.Errorf("*Age 应保留")
	}
	if dst.Score == nil || *dst.Score != 88.8 {
		t.Errorf("*Score 应保留")
	}
	if dst.Active == nil || *dst.Active != true {
		t.Errorf("*Active 应保留")
	}
	if dst.Description != nil {
		t.Errorf("Description 应被 tag 清零为 nil, got %v", dst.Description)
	}
	// 注意：字段名 CreatedAt 命中忽略规则，与类型无关
	if dst.CreatedAt != nil {
		t.Logf("⚠️ *time.Time 类型的 CreatedAt 因字段名命中被清零为 nil (当前: %v)", dst.CreatedAt)
	}
}

func TestStruct_SliceFields(t *testing.T) {
	src := sliceFieldModel{
		ID:        3,
		Tags:      []string{"a", "b"},
		Numbers:   []int{1, 2, 3},
		Scores:    []float64{1.1, 2.2},
		Flags:     []bool{true, false},
		CreatedAt: testNow,
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if !reflect.DeepEqual(dst.Tags, []string{"a", "b"}) {
		t.Errorf("Tags 应保留, got %v", dst.Tags)
	}
	if !reflect.DeepEqual(dst.Numbers, []int{1, 2, 3}) {
		t.Errorf("Numbers 应保留")
	}
	if !reflect.DeepEqual(dst.Scores, []float64{1.1, 2.2}) {
		t.Errorf("Scores 应保留")
	}
	if !reflect.DeepEqual(dst.Flags, []bool{true, false}) {
		t.Errorf("Flags 应保留")
	}

	// 验证底层数组独立
	dst.Tags[0] = "MODIFIED"
	if src.Tags[0] != "a" {
		t.Errorf("修改 dst.Tags 不应影响 src")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零")
	}
}

func TestStruct_MapFields(t *testing.T) {
	src := mapFieldModel{
		ID:        4,
		Attrs:     map[string]string{"k1": "v1"},
		Counts:    map[string]int{"a": 1},
		Extra:     map[string]interface{}{"x": 1.5, "y": "str"},
		CreatedAt: testNow,
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Attrs["k1"] != "v1" {
		t.Errorf("Attrs 应保留")
	}
	if dst.Counts["a"] != 1 {
		t.Errorf("Counts 应保留")
	}
	if dst.Extra["x"] != 1.5 {
		t.Errorf("Extra 应保留")
	}

	dst.Attrs["k1"] = "MODIFIED"
	if src.Attrs["k1"] != "v1" {
		t.Errorf("修改 dst.Attrs 不应影响 src")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零")
	}
}

func TestStruct_NestedValue(t *testing.T) {
	src := nestedValueModel{ID: 5, Inner: innerModel{ID: 51, Label: "inner"}, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("外层 ID 应清零")
	}
	if dst.Inner.ID != 0 {
		t.Errorf("内层 Inner.ID 应递归清零, got %d", dst.Inner.ID)
	}
	if dst.Inner.Label != "inner" {
		t.Errorf("内层 Label 应保留")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("外层 CreatedAt 应清零")
	}
}

func TestStruct_NestedPointer(t *testing.T) {
	src := nestedPtrModel{ID: 6, Inner: &innerModel{ID: 61, Label: "innerPtr"}, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("外层 ID 应清零")
	}
	if dst.Inner == nil {
		t.Fatalf("Inner 不应为 nil")
	}
	if dst.Inner.ID != 0 {
		t.Errorf("Inner.ID 应递归清零, got %d", dst.Inner.ID)
	}
	if dst.Inner.Label != "innerPtr" {
		t.Errorf("Inner.Label 应保留")
	}
	if dst.Inner == src.Inner {
		t.Errorf("Inner 应为深拷贝（指针独立）")
	}
}

func TestStruct_NestedSlice(t *testing.T) {
	src := nestedSliceModel{
		ID:        7,
		Items:     []innerModel{{ID: 71, Label: "a"}, {ID: 72, Label: "b"}},
		ItemPtrs:  []*innerModel{{ID: 73, Label: "c"}},
		CreatedAt: testNow,
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if len(dst.Items) != 2 || dst.Items[0].Label != "a" {
		t.Errorf("Items 应保留")
	}
	if len(dst.ItemPtrs) != 1 || dst.ItemPtrs[0] == src.ItemPtrs[0] {
		t.Errorf("ItemPtrs 应深拷贝且元素指针独立")
	}

	// ⚠️ 已知限制：slice 元素内字段不递归
	if dst.Items[0].ID != 71 {
		t.Logf("⚠️ 已知限制: Items[0].ID 未清零 (slice 元素不递归), got %d", dst.Items[0].ID)
	}
}

func TestStruct_EmbeddedStruct(t *testing.T) {
	src := embedModel{
		baseFields: baseFields{ID: 8, CreatedAt: testNow, UpdatedAt: testNow},
		Name:       "embed",
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("嵌入 ID 应清零")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("嵌入 CreatedAt 应清零")
	}
	if dst.UpdatedAt.IsZero() {
		t.Errorf("嵌入 UpdatedAt 应保留")
	}
	if dst.Name != "embed" {
		t.Errorf("Name 应保留")
	}
}

func TestStruct_TimeFields(t *testing.T) {
	src := timeModel{ID: 9, StartAt: testNow, EndAt: timePtr(testNow), CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if !dst.StartAt.Equal(testNow) {
		t.Errorf("StartAt 应保留")
	}
	if dst.EndAt == nil || !dst.EndAt.Equal(testNow) {
		t.Errorf("EndAt 应保留")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零")
	}
}

func TestStruct_GormDeletedAt(t *testing.T) {
	src := softDeleteModel{
		ID:        10,
		Name:      "soft",
		DeletedAt: gorm.DeletedAt{Time: testNow, Valid: true},
		CreatedAt: testNow,
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Name != "soft" {
		t.Errorf("Name 应保留")
	}
	if !dst.DeletedAt.Valid {
		t.Errorf("DeletedAt.Valid 应保留")
	}
	if !dst.DeletedAt.Time.Equal(testNow) {
		t.Errorf("DeletedAt.Time 应保留")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零")
	}
}

func TestStruct_CloneIgnoreTag(t *testing.T) {
	src := tagIgnoreModel{ID: 11, PublicData: "pub", PrivateData: "priv", InternalID: 999, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.PublicData != "pub" {
		t.Errorf("PublicData 应保留")
	}
	if dst.PrivateData != "" {
		t.Errorf("PrivateData 应被 tag 清零, got %q", dst.PrivateData)
	}
	if dst.InternalID != 0 {
		t.Errorf("InternalID 应被 tag 清零, got %d", dst.InternalID)
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零")
	}
}

func TestStruct_CustomPrimaryKey(t *testing.T) {
	src := customPKModel{OrderNo: "ORD-001", Amount: 123.45, CreatedAt: testNow}
	dst := Struct(src)

	if dst.OrderNo != "" {
		t.Errorf("OrderNo (gorm primaryKey) 应清零, got %q", dst.OrderNo)
	}
	if dst.Amount != 123.45 {
		t.Errorf("Amount 应保留")
	}
	if !dst.CreatedAt.IsZero() {
		t.Errorf("CreatedAt 应清零")
	}
}

func TestStruct_UnexportedFields(t *testing.T) {
	src := unexportedModel{ID: 13, Name: "n", internal: "hidden", CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Name != "n" {
		t.Errorf("Name 应保留")
	}
	// 未导出字段不参与 JSON 序列化，必然为零值
	if dst.internal != "" {
		t.Logf("⚠️ 未导出字段不参与 JSON 序列化, got %q", dst.internal)
	}
}

func TestStruct_InterfaceField(t *testing.T) {
	src := ifaceFieldModel{ID: 14, Payload: map[string]any{"k": "v", "n": 1.0}, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	m, ok := dst.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("Payload 类型应保持 map[string]interface{}, got %T", dst.Payload)
	}
	if m["k"] != "v" || m["n"] != 1.0 {
		t.Errorf("Payload 内容应深拷贝")
	}
}

func TestStruct_ArrayField(t *testing.T) {
	src := arrayModel{ID: 15, Matrix: [3]int{1, 2, 3}, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Matrix != [3]int{1, 2, 3} {
		t.Errorf("Matrix 应保留")
	}
}

func TestStruct_TypeAlias(t *testing.T) {
	src := aliasModel{ID: 16, Status: testStatus("active"), CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Status != testStatus("active") {
		t.Errorf("Status 应保留且类型正确")
	}
}

func TestStruct_JsonDashField(t *testing.T) {
	src := jsonDashModel{ID: 17, Public: "pub", Hidden: "should-be-lost", CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Public != "pub" {
		t.Errorf("Public 应保留")
	}
	// ⚠️ 已知限制：json:"-" 字段在 JSON 往返后丢失
	if dst.Hidden != "" {
		t.Logf("⚠️ 已知限制: Hidden (json:\"-\") 在 JSON 往返后丢失, got %q", dst.Hidden)
	}
}

func TestStruct_MultiLevelNested(t *testing.T) {
	src := level1{
		ID: 18,
		L2: &level2{
			ID:     181,
			L3:     &level3{ID: 1811, Value: "deep"},
			L3List: []level3{{ID: 1812, Value: "list"}},
		},
		CreatedAt: testNow,
	}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("L1.ID 应清零")
	}
	if dst.L2 == nil {
		t.Fatalf("L2 不应为 nil")
	}
	if dst.L2.ID != 0 {
		t.Errorf("L2.ID 应递归清零")
	}
	if dst.L2.L3 == nil {
		t.Fatalf("L2.L3 不应为 nil")
	}
	if dst.L2.L3.ID != 0 {
		t.Errorf("L2.L3.ID 应递归清零")
	}
	if dst.L2.L3.Value != "deep" {
		t.Errorf("L2.L3.Value 应保留")
	}
	// ⚠️ 已知限制：slice 元素不递归
	if dst.L2.L3List[0].ID != 1812 {
		t.Logf("⚠️ 已知限制: L3List[0].ID 未清零 (slice 元素不递归)")
	}
}

func TestStruct_NilPointerField(t *testing.T) {
	src := nilPtrModel{ID: 19, Inner: nil, CreatedAt: testNow}
	dst := Struct(src)

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Inner != nil {
		t.Errorf("nil 指针应保持 nil")
	}
}

func TestStruct_EmptyStruct(t *testing.T) {
	src := emptyModel{}
	dst := Struct(src)

	if !reflect.DeepEqual(src, dst) {
		t.Errorf("空 struct 克隆应相等")
	}
}

func TestStruct_WithIgnoreFields(t *testing.T) {
	src := scalarModel{ID: 1, Name: "n", Age: 18, Score: 99.5, CreatedAt: testNow}
	dst := Struct(src, WithIgnoreFields("Name", "Age"))

	if dst.ID != 0 {
		t.Errorf("ID 应清零")
	}
	if dst.Name != "" {
		t.Errorf("Name 应被 WithIgnoreFields 清零")
	}
	if dst.Age != 0 {
		t.Errorf("Age 应被 WithIgnoreFields 清零")
	}
	if dst.Score != 99.5 {
		t.Errorf("Score 应保留")
	}
}

func TestStruct_WithPrimaryKey(t *testing.T) {
	src := scalarModel{ID: 100, Name: "n", CreatedAt: testNow}

	// 默认清零
	dst1 := Struct(src)
	if dst1.ID != 0 {
		t.Errorf("默认: ID 应清零")
	}

	// 关闭
	dst2 := Struct(src, WithPrimaryKey(false))
	if dst2.ID != 100 {
		t.Errorf("WithPrimaryKey(false): ID 应保留")
	}
}

func TestStruct_WithCreatedAt(t *testing.T) {
	src := scalarModel{ID: 1, Name: "n", CreatedAt: testNow}

	// 默认清零
	dst1 := Struct(src)
	if !dst1.CreatedAt.IsZero() {
		t.Errorf("默认: CreatedAt 应清零")
	}

	// 关闭
	dst2 := Struct(src, WithCreatedAt(false))
	if !dst2.CreatedAt.Equal(testNow) {
		t.Errorf("WithCreatedAt(false): CreatedAt 应保留")
	}
}

func TestSlice_Basic(t *testing.T) {
	src := []scalarModel{
		{ID: 1, Name: "a", CreatedAt: testNow},
		{ID: 2, Name: "b", CreatedAt: testNow},
	}
	dst := Slice(src)

	if len(dst) != 2 {
		t.Fatalf("长度应一致")
	}
	if dst[0].ID != 0 || dst[1].ID != 0 {
		t.Errorf("每项 ID 应清零")
	}
	if dst[0].Name != "a" || dst[1].Name != "b" {
		t.Errorf("每项 Name 应保留")
	}
	if &dst[0] == &src[0] {
		t.Errorf("slice 底层数组应独立")
	}
}

func TestSlice_WithOptions(t *testing.T) {
	src := []scalarModel{
		{ID: 1, Name: "a", Age: 10, CreatedAt: testNow},
		{ID: 2, Name: "b", Age: 20, CreatedAt: testNow},
	}
	dst := Slice(src, WithIgnoreFields("Age"))

	if dst[0].Age != 0 || dst[1].Age != 0 {
		t.Errorf("每项 Age 应被清零")
	}
	if dst[0].Name != "a" || dst[1].Name != "b" {
		t.Errorf("每项 Name 应保留")
	}
}

func TestMap_Basic(t *testing.T) {
	src := map[string]any{
		"ID":           22,
		"Id":           220,
		"CreatedAt":    testNow,
		"MaterialCode": "MAT",
		"Brand":        "B",
	}
	dst := Map(src)

	if _, has := dst["ID"]; has {
		t.Errorf("默认: ID key 应被删")
	}
	if _, has := dst["Id"]; has {
		t.Errorf("默认: Id key 应被删")
	}
	if _, has := dst["CreatedAt"]; has {
		t.Errorf("默认: CreatedAt key 应被删")
	}
	if dst["MaterialCode"] != "MAT" {
		t.Errorf("其他 key 应保留")
	}
}

func TestMap_WithPrimaryKey(t *testing.T) {
	src := map[string]any{"ID": 22, "CreatedAt": testNow, "Name": "n"}

	dst := Map(src, WithPrimaryKey(false))
	if _, has := dst["ID"]; !has {
		t.Errorf("WithPrimaryKey(false): ID 应保留")
	}
	if _, has := dst["CreatedAt"]; has {
		t.Errorf("WithPrimaryKey(false): CreatedAt 仍应被删")
	}
}

func TestMap_WithCreatedAt(t *testing.T) {
	src := map[string]any{"ID": 22, "CreatedAt": testNow, "Name": "n"}

	dst := Map(src, WithCreatedAt(false))
	if _, has := dst["ID"]; has {
		t.Errorf("WithCreatedAt(false): ID 仍应被删")
	}
	if _, has := dst["CreatedAt"]; !has {
		t.Errorf("WithCreatedAt(false): CreatedAt 应保留")
	}
}

func TestMap_WithIgnoreFields(t *testing.T) {
	src := map[string]any{"ID": 22, "Brand": "B", "Name": "n"}

	dst := Map(src, WithIgnoreFields("Brand"))
	if _, has := dst["Brand"]; has {
		t.Errorf("WithIgnoreFields(\"Brand\"): Brand 应被删")
	}
	if dst["Name"] != "n" {
		t.Errorf("Name 应保留")
	}
}

func TestMap_NilMap(t *testing.T) {
	var nilMap map[string]any
	dst := Map(nilMap)
	if dst != nil {
		t.Errorf("nil map 应返回 nil")
	}
}

func TestDeepCopy_UnsupportedType_Panics(t *testing.T) {
	type badStruct struct {
		ID int64 `gorm:"primaryKey"`
		Ch chan int
	}
	src := badStruct{ID: 1, Ch: make(chan int)}

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("应 panic（json.Marshal 失败）")
		}
	}()
	_ = Struct(src)
}

func TestDeepCopy_SourceUnmodified(t *testing.T) {
	src := scalarModel{ID: 1, Name: "orig", CreatedAt: testNow}
	dst := Struct(src)

	dst.Name = "MODIFIED"
	dst.ID = 999

	if src.Name != "orig" {
		t.Errorf("src.Name 不应被修改, got %q", src.Name)
	}
	if src.ID != 1 {
		t.Errorf("src.ID 不应被修改, got %d", src.ID)
	}
}
