# convert
> copy 包中对时间的转换 （见convert/convert.go）
```go
	_ = copier.CopyWithOption(&resp.Data, &data, copier.Option{
		Converters: []copier.TypeConverter{convert.WithTime2String()},
	})
```
```go

// copier.Copy 辅助转换器
func WithTime2String() copier.TypeConverter {
	converters := copier.TypeConverter{
		SrcType: time.Time{},
		DstType: copier.String,
		Fn: func(src interface{}) (interface{}, error) {
			s, ok := src.(time.Time)

			if !ok {
				return nil, errors.New("src type not matching")
			}

			return s.Format("2006-01-02 15:04:05"), nil
		},
	}
	return converters
}

```
