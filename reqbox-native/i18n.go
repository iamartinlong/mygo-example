package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"reqbox-desktop/pkg/store"
)

// appLang is "zh" or "en", resolved from the user's preferred languages
// as mygo matched them against SetLanguages("en", "zh-Hans") — the same
// auto-detection the web does on navigator.language — unless the header
// menu has stored an explicit choice.
var appLang string

func initLanguages() {
	if strings.HasPrefix(mygo.App.Locale(), "zh") {
		appLang = "zh"
	} else {
		appLang = "en"
	}
}

const setLangKey = "lang"

// setLang applies and remembers a menu pick; "en" and "zh" are the only
// ids the header menu offers.
func setLang(st *store.Store, id string) {
	if id != "en" && id != "zh" {
		return
	}
	appLang = id
	if st != nil {
		_ = st.SetSetting(setLangKey, id)
	}
}

// applyLangSetting lets a stored menu pick override the auto-detection.
func applyLangSetting(st *store.Store) {
	if st == nil {
		return
	}
	if v := st.GetSetting(setLangKey, ""); v == "en" || v == "zh" {
		appLang = v
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
