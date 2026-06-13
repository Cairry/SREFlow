package i18n

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// LanguageHeader Accept-Language header name
	LanguageHeader = "Accept-Language"
	// LanguageContextKey language context key
	LanguageContextKey = "language"
)

// LanguageMiddleware extracts language from request header
func LanguageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get language from header
		lang := c.GetHeader(LanguageHeader)

		// Validate language
		if lang == "" || !isValidLanguage(lang) {
			lang = DefaultLanguage
		}

		// Simplify language code (zh-CN -> zh, en-US -> en)
		if strings.Contains(lang, "-") {
			lang = strings.Split(lang, "-")[0]
		}

		// Set to context
		c.Set(LanguageContextKey, lang)

		c.Next()
	}
}

// GetLanguageFromContext gets language from gin context
func GetLanguageFromContext(c *gin.Context) string {
	lang, exists := c.Get(LanguageContextKey)
	if !exists {
		return DefaultLanguage
	}
	return lang.(string)
}

// isValidLanguage checks if language is supported
func isValidLanguage(lang string) bool {
	lang = strings.ToLower(lang)
	if strings.Contains(lang, "-") {
		lang = strings.Split(lang, "-")[0]
	}

	for _, supported := range SupportedLanguages {
		if supported == lang {
			return true
		}
	}
	return false
}
