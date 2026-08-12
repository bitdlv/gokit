package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestJsonStr_UnmarshalJSON(t *testing.T) {
	type a struct {
		A JsonStr `json:"a"`
	}

	var aa a
	j := `{"a": {"b": "c"}}`
	err := json.Unmarshal([]byte(j), &aa)
	if err != nil {
		t.Fatal(err)
	}
	if aa.A != "{\"b\": \"c\"}" {
		t.Fatal(errors.New("should be {\"b\": \"c\"}"))
	}
}

func TestJsonStr_MarshalJSON(t *testing.T) {
	type a struct {
		A JsonStr `json:"a"`
	}

	aa := a{
		A: "{\"b\": \"c\"}",
	}

	bytes, err := json.Marshal(aa)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "{\"a\":{\"b\":\"c\"}}" {
		t.Fatal(errors.New(fmt.Sprintf("actually: %s", string(bytes))))
	}
}

func TestJsonStr_MarshalJSONInterface(t *testing.T) {
	type a struct {
		A interface{} `json:"a"`
	}

	aa := a{
		A: JsonStr("{\"b\": \"c\"}"),
	}

	bytes, err := json.Marshal(aa)
	if err != nil {
		t.Fatal(err)
	}

	if string(bytes) != "{\"a\":{\"b\":\"c\"}}" {
		t.Fatal(errors.New(fmt.Sprintf("actually: %s", string(bytes))))
	}
}
