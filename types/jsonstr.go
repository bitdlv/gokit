package types

// JsonStr 将json字段转为字符串字段
type JsonStr string

func (c *JsonStr) UnmarshalJSON(b []byte) (err error) {
	if string(*c) == "null" {
		*c = ""
	} else {
		*c = JsonStr(b)
	}
	return nil
}

func (c JsonStr) MarshalJSON() ([]byte, error) {
	if string(c) == "" {
		return []byte("null"), nil
	}

	return []byte(c), nil
}
