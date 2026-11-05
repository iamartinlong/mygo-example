package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
)

// appLang is "zh" or "en", resolved from the user's preferred languages
// as mygo matched them against SetLanguages("en", "zh-Hans") — the same
// auto-detection the web does on navigator.language.
var appLang string

func initLanguages() {
	if strings.HasPrefix(mygo.App.Locale(), "zh") {
		appLang = "zh"
	} else {
		appLang = "en"
	}
}

// tr returns the message for key in the app language, filling {name}
// placeholders from args pairs: tr("toast.duplicated", "name", n).
func tr(key string, args ...any) string {
	s, ok := msgEN[key]
	if !ok {
		return key
	}
	if appLang == "zh" {
		if zh, ok := msgZH[key]; ok {
			s = zh
		}
	}
	for i := 0; i+1 < len(args); i += 2 {
		s = strings.ReplaceAll(s, "{"+fmt.Sprint(args[i])+"}", fmt.Sprint(args[i+1]))
	}
	return s
}
