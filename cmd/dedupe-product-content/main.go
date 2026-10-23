// Command dedupe-product-content 清理商品正文里重复出现的「核心亮点/适用场景/交付说明」区块。
//
// 默认 dry-run，只输出 slug、语言、重复区块与预计删除字符数；确认后执行：
//
//	go run ./cmd/dedupe-product-content -apply
//
// 只删除重复的标题区块，不修改标题、SEO、价格、SKU；重复内容应清理数据而不是前端隐藏。
package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	databasemigrations "github.com/dujiao-next/internal/bootstrap/database/migrations"
	"github.com/dujiao-next/internal/config"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	"github.com/dujiao-next/internal/platform/database/gormdb"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"gorm.io/gorm"
)

var locales = []string{"zh-CN", "zh-TW", "en-US"}
var operationErrors []error

func recordOperationError(err error) {
	if err == nil {
		return
	}
	operationErrors = append(operationErrors, err)
	fmt.Fprintf(os.Stderr, "%v\n", err)
}

func migrateIfRequested(apply, migrate bool) error {
	if !apply && !migrate {
		return nil
	}
	return databasemigrations.AutoMigrate()
}

func isMissingSchemaError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") ||
		strings.Contains(message, "no such column") ||
		strings.Contains(message, "does not exist") ||
		strings.Contains(message, "undefined column")
}

func failureExitCode(apply, migrate bool) int {
	if len(operationErrors) == 0 {
		return 0
	}
	if !apply && !migrate {
		for _, err := range operationErrors {
			if isMissingSchemaError(err) {
				return 2
			}
		}
	}
	return 1
}

func printOperationErrors() {
	if len(operationErrors) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "\n共 %d 项操作失败：\n", len(operationErrors))
	for _, err := range operationErrors {
		fmt.Fprintf(os.Stderr, "- %v\n", err)
	}
}

// 只处理这几类被重复写入的区块，避免误删其它正文。
var dedupeSectionLabels = map[string]struct{}{
	"核心亮点": {}, "核心亮點": {}, "套餐亮点": {}, "套餐亮點": {}, "方案亮点": {}, "方案亮點": {},
	"适用场景": {}, "適用場景": {}, "适用范围": {}, "適用範圍": {},
	"交付说明": {}, "交付說明": {},
	"highlights": {}, "use cases": {}, "delivery": {}, "delivery notes": {},
}

var (
	h2Pattern  = regexp.MustCompile(`(?is)<h2[^>]*>.*?</h2>`)
	tagPattern = regexp.MustCompile(`(?s)<[^>]+>`)
)

type change struct {
	Slug    string
	Locale  string
	Labels  string
	Removed int
	OldLen  int
	NewLen  int
}

func main() {
	apply := flag.Bool("apply", false, "写入数据库；默认只做 dry-run")
	migrate := flag.Bool("migrate", false, "允许执行数据库迁移；默认 dry-run 不迁移")
	reportPath := flag.String("report", "", "把变更清单写入 CSV 路径（可选）")
	flag.Parse()

	cfg := config.Load()
	if err := gormdb.InitDB(cfg.Database.Driver, cfg.Database.DSN, gormdb.DBPoolConfig{
		MaxOpenConns:           cfg.Database.Pool.MaxOpenConns,
		MaxIdleConns:           cfg.Database.Pool.MaxIdleConns,
		ConnMaxLifetimeSeconds: cfg.Database.Pool.ConnMaxLifetimeSeconds,
		ConnMaxIdleTimeSeconds: cfg.Database.Pool.ConnMaxIdleTimeSeconds,
	}, cfg.Server.Mode); err != nil {
		fatalf("init db: %v", err)
	}
	if err := migrateIfRequested(*apply, *migrate); err != nil {
		fatalf("migrate db: %v", err)
	}

	operationErrors = nil
	var changes []change
	if *apply {
		err := gormdb.DB.Transaction(func(tx *gorm.DB) error {
			changes = dedupeProducts(tx, true)
			if len(operationErrors) > 0 {
				return errors.Join(operationErrors...)
			}
			return nil
		})
		if err != nil && len(operationErrors) == 0 {
			recordOperationError(err)
		}
	} else {
		changes = dedupeProducts(gormdb.DB, false)
	}
	fmt.Printf("\n共 %d 项变更（apply=%v）\n", len(changes), *apply)
	for i, item := range changes {
		if i >= 40 {
			fmt.Printf("... 其余 %d 项省略（可用 -report 输出完整 CSV）\n", len(changes)-40)
			break
		}
		fmt.Printf("- %s [%s] 删除 %d 字符（%d -> %d），区块：%s\n", item.Slug, item.Locale, item.Removed, item.OldLen, item.NewLen, item.Labels)
	}
	if *reportPath != "" {
		if err := writeReport(*reportPath, changes); err != nil {
			fatalf("write report: %v", err)
		}
		fmt.Printf("变更清单已写入 %s\n", *reportPath)
	}
	printOperationErrors()
	if len(operationErrors) > 0 {
		if !*apply && !*migrate {
			for _, err := range operationErrors {
				if isMissingSchemaError(err) {
					fmt.Fprintln(os.Stderr, "数据库表/字段不存在；dry-run 不自动迁移，请确认备份后加 -migrate 或 -apply。")
					break
				}
			}
		}
		os.Exit(failureExitCode(*apply, *migrate))
	}
	if !*apply {
		fmt.Println("\n这是 dry-run；确认无误后加 -apply 执行。")
	}
}

func dedupeProducts(db *gorm.DB, apply bool) []change {
	store := productgormstore.NewProductStore(db)
	products, _, err := store.List(productcontract.ListFilter{Page: 1, PageSize: 100000})
	if err != nil {
		recordOperationError(fmt.Errorf("list products: %w", err))
		return nil
	}
	changes := make([]change, 0, len(products))
	for i := range products {
		product := &products[i]
		contents := stringMap(product.ContentJSON)
		changed := false
		for _, locale := range locales {
			oldContent := contents[locale]
			newContent, removed, labels := dedupeContent(oldContent)
			if removed == 0 || newContent == oldContent {
				continue
			}
			contents[locale] = newContent
			changes = append(changes, change{
				Slug: product.Slug, Locale: locale, Labels: strings.Join(labels, ","),
				Removed: removed, OldLen: len([]rune(oldContent)), NewLen: len([]rune(newContent)),
			})
			changed = true
		}
		if !changed || !apply {
			continue
		}
		if err := store.QuickUpdate(strconv.FormatUint(uint64(product.ID), 10), map[string]interface{}{"content_json": jsonmap.JSON(toAnyMap(contents))}); err != nil {
			recordOperationError(fmt.Errorf("update product %s: %w", product.Slug, err))
		}
	}
	return changes
}

// dedupeContent 按 h2 区块切分正文：同一个白名单标题第二次出现时整段删除。
func dedupeContent(html string) (string, int, []string) {
	matches := h2Pattern.FindAllStringIndex(html, -1)
	if len(matches) == 0 {
		// 没有标题时，退化为“整段重复”检测（A+A）。
		trimmed := strings.TrimSpace(html)
		half := len(trimmed) / 2
		if half > 0 && len(trimmed)%2 == 0 && trimmed[:half] == trimmed[half:] {
			return trimmed[:half], len([]rune(trimmed[:half])), []string{"duplicate-half"}
		}
		return html, 0, nil
	}

	type section struct {
		start, end int
		label      string
	}
	sections := make([]section, 0, len(matches))
	for i, match := range matches {
		end := len(html)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		label := normalizeLabel(html[match[0]:match[1]])
		sections = append(sections, section{start: match[0], end: end, label: label})
	}

	counts := make(map[string]int)
	for _, item := range sections {
		if _, ok := dedupeSectionLabels[item.label]; ok {
			counts[item.label]++
		}
	}
	hasDuplicate := false
	for _, count := range counts {
		if count > 1 {
			hasDuplicate = true
			break
		}
	}
	if !hasDuplicate {
		return html, 0, nil
	}

	seen := map[string]struct{}{}
	removedLabels := make([]string, 0, 4)
	var builder strings.Builder
	builder.WriteString(html[:matches[0][0]])
	removed := 0
	for _, item := range sections {
		if _, ok := dedupeSectionLabels[item.label]; ok {
			if _, already := seen[item.label]; already {
				removed += len([]rune(html[item.start:item.end]))
				removedLabels = append(removedLabels, item.label)
				continue
			}
			seen[item.label] = struct{}{}
		}
		builder.WriteString(html[item.start:item.end])
	}
	return builder.String(), removed, removedLabels
}

func normalizeLabel(raw string) string {
	text := tagPattern.ReplaceAllString(raw, "")
	return strings.ToLower(strings.TrimSpace(text))
}

func toAnyMap(source map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func stringMap(value interface{}) map[string]string {
	result := map[string]string{}
	if source, ok := value.(map[string]interface{}); ok {
		for key, item := range source {
			if text, ok := item.(string); ok {
				result[key] = text
			}
		}
	}
	if source, ok := value.(jsonmap.JSON); ok {
		for key, item := range source {
			if text, ok := item.(string); ok {
				result[key] = text
			}
		}
	}
	for _, loc := range locales {
		if _, ok := result[loc]; !ok {
			result[loc] = ""
		}
	}
	return result
}

func writeReport(path string, changes []change) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"slug", "locale", "labels", "removed", "old_len", "new_len"}); err != nil {
		return err
	}
	for _, item := range changes {
		if err := writer.Write([]string{item.Slug, item.Locale, item.Labels, strconv.Itoa(item.Removed), strconv.Itoa(item.OldLen), strconv.Itoa(item.NewLen)}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
