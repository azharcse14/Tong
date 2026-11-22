package i18n

import (
	"encoding/json"
	"os"
)

// সব ভাষার messages এক জায়গায় রাখব
type Localizer struct {
	messages map[string]map[string]string
}

// Program শুরুতে JSON file গুলো পড়ে memory তে রাখব
func NewLocalizer(locales map[string]string) (*Localizer, error) {
	l := &Localizer{
		messages: make(map[string]map[string]string),
	}

	for lang, filePath := range locales {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}

		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			return nil, err
		}

		l.messages[lang] = msgs
	}

	return l, nil
}

// Key দিয়ে message আনব, ভাষা না থাকলে English fallback
func (l *Localizer) T(lang, key string) string {
	if msgs, ok := l.messages[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	// Fallback to English
	if msgs, ok := l.messages["en"]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	// একদম না পেলে key ই দেখাও
	return key
}

// Request header থেকে ভাষা বের করব
func DetectLang(acceptLanguage string) string {
	if len(acceptLanguage) >= 2 {
		lang := acceptLanguage[:2]
		if lang == "bn" || lang == "en" {
			return lang
		}
	}
	return "en"
}