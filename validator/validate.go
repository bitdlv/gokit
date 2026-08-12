package validator

import (
	"context"
	"net/http"
	"time"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	tagTimestring = "timestring" //校验时间格式
)

// grpc 添加tag https://www.cnblogs.com/oxspirt/p/15949195.html
// 校验库 https://github.com/go-playground/validator/
// validator 使用方法 https://www.cnblogs.com/jiujuan/p/13823864.html

// 参数校验
func Validate(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	validate := validator.New()
	uni := ut.New(en.New(), zh.New())
	trans, _ := uni.GetTranslator("zh")
	_ = validate.RegisterValidation(tagTimestring, timestring(validate, trans))

	err = validate.Struct(req)
	if err == nil {
		return handler(ctx, req)
	}

	_ = zhTranslations.RegisterDefaultTranslations(validate, trans)
	verrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return resp, status.New(http.StatusBadRequest, err.Error()).Err()
	}
	var msg string
	for _, value := range verrs.Translate(trans) {
		msg = value
		break
	}

	return resp, status.New(http.StatusBadRequest, msg).Err()
}

func timestring(valid *validator.Validate, trans ut.Translator) func(fn validator.FieldLevel) bool {
	_ = valid.RegisterTranslation(tagTimestring, trans, func(ut ut.Translator) error {
		return ut.Add(tagTimestring, "{0} 时间格式不正确", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tagTimestring, fe.Field())
		return t
	})
	return func(fn validator.FieldLevel) bool {
		value := fn.Field().String()
		_, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
		return err == nil
	}
}
