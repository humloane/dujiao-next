package lpcontent

import "encoding/json"

// loaded 返回各语言的原始 JSON，用于和前台 locale 做语义比对。
func loaded() map[string]json.RawMessage {
	return rawByLocale
}
