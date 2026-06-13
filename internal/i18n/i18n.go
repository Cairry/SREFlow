package i18n

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

var (
	bundle    *i18n.Bundle
	localizer *i18n.Localizer
)

// SupportedLanguages 支持的语言列表
var SupportedLanguages = []string{"en", "zh"}

// DefaultLanguage 默认语言
const DefaultLanguage = "en"

// Init 初始化i18n
func Init() error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// 加载英语翻译
	_, err := bundle.LoadMessageFileFS(localeFS, "locales/en.json")
	if err != nil {
		return err
	}

	// 加载中文翻译
	_, err = bundle.LoadMessageFileFS(localeFS, "locales/zh.json")
	if err != nil {
		return err
	}

	// 设置默认localizer
	localizer = i18n.NewLocalizer(bundle, DefaultLanguage)

	return nil
}

// GetLocalizer 获取指定语言的localizer
func GetLocalizer(lang string) *i18n.Localizer {
	if lang == "" {
		lang = DefaultLanguage
	}
	return i18n.NewLocalizer(bundle, lang)
}

// T 翻译文本（使用默认语言）
func T(messageID string, templateData ...map[string]interface{}) string {
	return TWithLang(DefaultLanguage, messageID, templateData...)
}

// TWithLang 使用指定语言翻译文本
func TWithLang(lang, messageID string, templateData ...map[string]interface{}) string {
	loc := GetLocalizer(lang)

	var data map[string]interface{}
	if len(templateData) > 0 {
		data = templateData[0]
	}

	msg, err := loc.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: data,
	})
	if err != nil {
		// 如果翻译失败，返回messageID本身
		return messageID
	}

	return msg
}

// TError 翻译错误消息
func TError(lang, messageID string, templateData ...map[string]interface{}) error {
	return &I18nError{
		Message: TWithLang(lang, messageID, templateData...),
		Code:    messageID,
	}
}

// I18nError 国际化错误类型
type I18nError struct {
	Message string
	Code    string
}

func (e *I18nError) Error() string {
	return e.Message
}
