package environmentvariable

import (
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

type languageDef struct {
	Chinese string
	English string
}

var LanguageDef = languageDef{
	Chinese: "zh-cn",
	English: "en",
}

var SystemLanguage string = LanguageDef.Chinese

func SystemLanguageIsEnglish() bool {
	return SystemLanguage == LanguageDef.English
}

func SetSystemLanguage(language string) {
	functionName := "[environmentvariable.SetSystemLanguage]"

	switch strings.ToLower(language) {
	case LanguageDef.Chinese:
		SystemLanguage = LanguageDef.Chinese
	case LanguageDef.English:
		SystemLanguage = LanguageDef.English
	default:
		logx.Errorf("%s unknow language: %s", functionName, language)
	}
}
