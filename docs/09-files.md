# 九、文件 / 文档处理

## excel — Excel / CSV 大工具（1260 LOC）

**类型**：`Excel`、`ExcelPlus`、`Importer`、`Handler`、`DataProcessor`、`Group`、`Action`、`Value`、`Warning`

**子包**：`basic / cell / consts / csv / nexcel / nexcel/data / util / validator`

### 导出

```go
e := excel.NewExcel()
e.SetTitle("data").
  SetHead([]string{"姓名","分数"}).
  AddRecord([]any{"张三", 88}).
  BuildFile("/tmp/x.xlsx")

// 多 Sheet
e.NextSheet("sheet2").SetHead(...).WriteBody(rows)

// 反射自动 head
e.SetHeadByInterface(&User{})
```

| API | 说明 |
|---|---|
| `NewExcel()` | 构造 |
| `SetTitle(s)` | 标题 |
| `SetHead([]string) / SetHeadByInterface(v)` | 表头 |
| `SetBody(rows) / WriteBody(rows) / AddRecord(row)` | 数据 |
| `Write() / BuildFile(path) / BuildTitle(title)` | 输出 |
| `GenUsingDefaultSheet(rows)` | 快速导出 |
| `GetZipBytes()` | 打包为 zip |
| `NextSheet(name)` | 多 Sheet |
| `TableToExcel(table)` | 表格转 Excel |
| `Horizontal / Vertical` | 布局方向 |
| `ColIndexByNum(n) / Cell(x,y)` | 坐标 |

### 导入

```go
imp := excel.NewImporter(
    excel.WithSkipFirstRows(1),
    excel.WithValidator(myV),
    excel.WithRowValidators(rv1, rv2),
    excel.WithCreateProcessor(pCreate),
    excel.WithUpdateProcessor(pUpdate),
    excel.WithDeleteProcessor(pDelete),
    excel.WithModifier(mod),
)
result, err := imp.Import(reader)

// 或直接反射
var rows []MyRow
excel.Import(reader, &rows)

// CSV
excel.ReadCsvFile("/tmp/x.csv")
```

| API | 说明 |
|---|---|
| `NewImporter(opts...)` | 构造导入器 |
| `Import(reader, &v)` | 反射导入 |
| `ParseExcel(reader)` | 解析 |
| `ReadCsvFile(path)` | CSV |
| `GetDataFromActiveSheet()` | 取当前 sheet |
| `LoadRowBySheetIndex(idx)` | 按 sheet |
| `WithSkipFirstRows(n)` | 跳过表头 |
| `WithValidator / WithRowValidators / WithCellValidators` | 校验器 |
| `WithCreateProcessor / WithUpdateProcessor / WithDeleteProcessor` | CUD 处理 |
| `WithModifier(fn)` | 数据改写 |

### 校验 / 单元格

- `RequiredCol(name)` / `RequiredOneCol(names...)` — 必填列
- `IsEmpty(cell)` — 空值判断
- `NewValue(v)` / `ToInt32() / ToString() / ShouldClear()` — 值封装

### 测试

用 `testdata/*.xlsx`：
```bash
go test -v ./excel/...
```

## fileutils — 文件系统

| API | 说明 |
|---|---|
| `FileExists(path)` | 存在？ |
| `IsFile(path)` / `IsDir(path)` | 类型 |
| `IsDirExists(path) / IsPathExists(path)` | 存在 + 类型 |
| `IsRelativePath(p)` | 相对路径？ |
| `ListDir(path)` | 列出目录 |
| `SplitPath(p)` | 拆分路径 |
| `WriteLinesWithBufferFlag(path, lines, flag)` | 缓冲写行 |

```bash
go test -v ./fileutils/...
```
