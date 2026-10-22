// Command gen-lp-content 把前台三语 landing 文案导出成后端可 embed 的 JSON。
//
// 落地页正文既要给 Vue 渲染，也要让不执行 JS 的爬虫在原始 HTML 里看到，
// 所以后端需要同一份内容。前台 locale 仍是唯一事实来源，这个命令只做导出；
// internal/web/lpcontent 的测试会在内容漂移时失败。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var locales = []string{"zh-CN", "zh-TW", "en-US"}

func main() {
	root, err := repoRoot()
	if err != nil {
		fail(err)
	}
	output := map[string]json.RawMessage{}
	for _, locale := range locales {
		path := filepath.Join(root, "frontend", "user", "src", "i18n", "locales", locale+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		var doc struct {
			Landing json.RawMessage `json:"landing"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			fail(fmt.Errorf("%s: %w", path, err))
		}
		if len(doc.Landing) == 0 {
			fail(fmt.Errorf("%s: missing landing block", path))
		}
		output[locale] = doc.Landing
	}

	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fail(err)
	}
	target := filepath.Join(root, "internal", "web", "lpcontent", "landing.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fail(err)
	}
	if err := os.WriteFile(target, append(encoded, '\n'), 0o644); err != nil {
		fail(err)
	}
	keys := make([]string, 0, len(output))
	for key := range output {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("wrote %s (%v)\n", target, keys)
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "gen-lp-content:", err)
	os.Exit(1)
}
