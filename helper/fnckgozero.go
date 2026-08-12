package helper

import (
	"github.com/zeromicro/go-zero/core/mapping"
	"github.com/zeromicro/go-zero/core/validation"
	"github.com/zeromicro/go-zero/rest/httpx"
	"io"
	"net/http"
	"sync/atomic"
	_ "unsafe"
)

//go:linkname validator github.com/zeromicro/go-zero/rest/httpx.validator
var validator atomic.Value

var formUnmarshaler = mapping.NewUnmarshaler(formKey, mapping.WithStringValues(), mapping.WithOpaqueKeys())

const maxMemory = 32 << 20
const formKey = "form"

// Parse gozero的Parse方法，添加解析上传文件的功能。
//
//	type Example struct {
//		SomeFile []byte `form:"file"` //通过定义 []byte 类型的成员变量来接收文件
//	}
func Parse(r *http.Request, v any) error {
	if err := httpx.ParsePath(r, v); err != nil {
		return err
	}

	if err := ParseForm(r, v); err != nil {
		return err
	}

	if err := httpx.ParseHeaders(r, v); err != nil {
		return err
	}

	if err := httpx.ParseJsonBody(r, v); err != nil {
		return err
	}

	if valid, ok := v.(validation.Validator); ok {
		return valid.Validate()
	} else if val := validator.Load(); val != nil {
		return val.(httpx.Validator).Validate(r, v)
	}

	return nil
}

// ParseForm parses the form request.
func ParseForm(r *http.Request, v any) error {
	params, err := GetFormValues(r)
	if err != nil {
		return err
	}

	return formUnmarshaler.Unmarshal(params, v)
}

// GetFormValues returns the form values.
func GetFormValues(r *http.Request) (map[string]any, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		if err != http.ErrNotMultipart {
			return nil, err
		}
	}

	params := make(map[string]any, len(r.Form))
	for name := range r.Form {
		formValue := r.Form.Get(name)
		if len(formValue) > 0 {
			params[name] = formValue
		}
	}
	for key, files := range r.MultipartForm.File {
		if len(files) > 0 {
			f, err := files[0].Open()
			if err != nil {
				return nil, err
			}
			params[key], err = io.ReadAll(f)
			if err != nil {
				return nil, err
			}
			f.Close()
		}
	}

	return params, nil
}
