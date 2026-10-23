// Command backfill-seo-content 为商品/分类补齐多语言 SEO 文案，并清理公告与站点文案中的旧地区表述。
//
// 默认 dry-run，只打印将修改的字段；确认后执行：
//
//	go run ./cmd/backfill-seo-content -apply
//
// 说明：Post 已有可空的 seo_meta 字段，文章 SEO 在后台编辑页维护；
// 本命令只处理商品/分类 SEO、文章分类分配、公告/横幅/站点文案清理。
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	databasemigrations "github.com/dujiao-next/internal/bootstrap/database/migrations"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	categorygormstore "github.com/dujiao-next/internal/modules/catalog/category/infrastructure/gormstore"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	contentcontract "github.com/dujiao-next/internal/modules/content/contract"
	contentdomain "github.com/dujiao-next/internal/modules/content/domain"
	contentgormstore "github.com/dujiao-next/internal/modules/content/infrastructure/gormstore"
	settingsgormstore "github.com/dujiao-next/internal/modules/settings/infrastructure/gormstore"
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

// 在售的 16 个地区 + 英国（GB）。
var regionNames = map[string][3]string{
	"US": {"美国", "美國", "United States"},
	"GB": {"英国", "英國", "United Kingdom"},
	"DE": {"德国", "德國", "Germany"},
	"JP": {"日本", "日本", "Japan"},
	"TW": {"台湾", "台灣", "Taiwan"},
	"PH": {"菲律宾", "菲律賓", "Philippines"},
	"SG": {"新加坡", "新加坡", "Singapore"},
	"TH": {"泰国", "泰國", "Thailand"},
	"MY": {"马来西亚", "馬來西亞", "Malaysia"},
	"VN": {"越南", "越南", "Vietnam"},
	"ID": {"印度尼西亚", "印尼", "Indonesia"},
	"BR": {"巴西", "巴西", "Brazil"},
	"MX": {"墨西哥", "墨西哥", "Mexico"},
	"KR": {"韩国", "韓國", "South Korea"},
	"SA": {"沙特阿拉伯", "沙烏地阿拉伯", "Saudi Arabia"},
	"AE": {"阿联酋", "阿聯酋", "United Arab Emirates"},
}

var (
	oldRegionPhrases = []string{
		"美国、英国、德国、日本、台湾、菲律宾",
		"美國、英國、德國、日本、台灣、菲律賓",
		"US, UK, DE, JP, TW and PH",
		"六个地区",
		"六地区",
		"6 个地区",
	}
	newRegionPhrase = map[string]string{
		"zh-CN": "全球 16 个地区（US/GB/DE/JP/TW/PH/SG/TH/MY/VN/ID/BR/MX/KR/SA/AE）",
		"zh-TW": "全球 16 個地區（US/GB/DE/JP/TW/PH/SG/TH/MY/VN/ID/BR/MX/KR/SA/AE）",
		"en-US": "16 regions worldwide (US/GB/DE/JP/TW/PH/SG/TH/MY/VN/ID/BR/MX/KR/SA/AE)",
	}
	retiredServiceKeywords = []string{"共享节点套餐", "虚拟账号", "网站搭建", "小程序开发", "企业邮局"}
)

type change struct {
	Kind  string
	Slug  string
	Field string
	Old   string
	New   string
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
	db := gormdb.DB

	settingsStore := settingsgormstore.New(db)
	siteConfig, _, err := settingsStore.GetByKey(constants.SettingKeySiteConfig)
	if err != nil {
		recordOperationError(fmt.Errorf("load site config: %w", err))
		printOperationErrors()
		os.Exit(failureExitCode(*apply, *migrate))
	}
	brandByLocale := map[string]string{}
	for _, loc := range locales {
		brandByLocale[loc] = seoBrand(siteConfig, loc)
	}

	ctx := context.Background()
	operationErrors = nil
	run := func(tx *gorm.DB) []change {
		changes := make([]change, 0, 128)
		changes = append(changes, backfillProducts(tx, brandByLocale, *apply)...)
		changes = append(changes, backfillCategories(tx, brandByLocale, *apply)...)
		changes = append(changes, ensurePostCategories(tx, ctx, *apply)...)
		changes = append(changes, assignPostCategories(tx, ctx, *apply)...)
		changes = append(changes, cleanNotices(tx, ctx, *apply)...)
		changes = append(changes, cleanBanners(tx, ctx, *apply)...)
		changes = append(changes, cleanSiteCopy(settingsgormstore.New(tx), siteConfig, *apply)...)
		return changes
	}

	var changes []change
	if *apply {
		err := db.Transaction(func(tx *gorm.DB) error {
			changes = run(tx)
			if len(operationErrors) > 0 {
				return errors.Join(operationErrors...)
			}
			return nil
		})
		if err != nil && len(operationErrors) == 0 {
			recordOperationError(err)
		}
	} else {
		changes = run(db)
	}

	fmt.Printf("\n共 %d 项变更（apply=%v）\n", len(changes), *apply)
	printChanges(changes)
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

// ---- 回填实现 ----

func backfillProducts(db *gorm.DB, brand map[string]string, apply bool) []change {
	store := productgormstore.NewProductStore(db)
	products, _, err := store.List(productcontract.ListFilter{Page: 1, PageSize: 100000})
	if err != nil {
		recordOperationError(fmt.Errorf("list products: %w", err))
		return nil
	}
	changes := make([]change, 0, len(products)*3)
	for i := range products {
		product := &products[i]
		seo := copyJSONMap(product.SeoMetaJSON)
		titles := stringMap(seo["title"])
		descriptions := stringMap(seo["description"])
		keywords := stringMap(seo["keywords"])
		changed := false
		for _, loc := range locales {
			if strings.TrimSpace(titles[loc]) == "" {
				value := productSEOTitle(product, loc)
				titles[loc] = value
				changes = append(changes, change{"product", product.Slug, "seo_meta.title." + loc, "", value})
				changed = true
			}
			if strings.TrimSpace(descriptions[loc]) == "" {
				value := productSEODescription(product, titles[loc], loc)
				descriptions[loc] = value
				changes = append(changes, change{"product", product.Slug, "seo_meta.description." + loc, "", value})
				changed = true
			}
			if strings.TrimSpace(keywords[loc]) == "" {
				value := productSEOKeywords(product, loc, brand[loc])
				keywords[loc] = value
				changes = append(changes, change{"product", product.Slug, "seo_meta.keywords." + loc, "", value})
				changed = true
			}
		}
		if !changed {
			continue
		}
		seo["title"] = titles
		seo["description"] = descriptions
		seo["keywords"] = keywords
		if apply {
			if err := store.QuickUpdate(strconv.FormatUint(uint64(product.ID), 10), map[string]interface{}{"seo_meta_json": jsonmap.JSON(seo)}); err != nil {
				recordOperationError(fmt.Errorf("update product %s: %w", product.Slug, err))
			}
		}
	}
	return changes
}

func backfillCategories(db *gorm.DB, brand map[string]string, apply bool) []change {
	store := categorygormstore.NewCategoryStore(db)
	categories, err := store.List()
	if err != nil {
		recordOperationError(fmt.Errorf("list categories: %w", err))
		return nil
	}
	changes := make([]change, 0, len(categories)*3)
	for i := range categories {
		category := &categories[i]
		count, err := store.CountActiveProducts(strconv.FormatUint(uint64(category.ID), 10))
		if err != nil {
			recordOperationError(fmt.Errorf("count category products %s: %w", category.Slug, err))
		}
		descriptions := stringMap(category.DescriptionJSON)
		seo := copyJSONMap(category.SeoMetaJSON)
		titles := stringMap(seo["title"])
		seoDescriptions := stringMap(seo["description"])
		keywords := stringMap(seo["keywords"])
		changed := false
		for _, loc := range locales {
			name := localizedText(category.NameJSON, loc)
			if strings.TrimSpace(descriptions[loc]) == "" {
				value := categoryDescription(name, count, loc)
				descriptions[loc] = value
				changes = append(changes, change{"category", category.Slug, "description." + loc, "", value})
				changed = true
			}
			if strings.TrimSpace(titles[loc]) == "" {
				value := categorySEOTitle(name)
				titles[loc] = value
				changes = append(changes, change{"category", category.Slug, "seo_meta.title." + loc, "", value})
				changed = true
			}
			if strings.TrimSpace(seoDescriptions[loc]) == "" {
				value := truncate(descriptions[loc], 150)
				seoDescriptions[loc] = value
				changes = append(changes, change{"category", category.Slug, "seo_meta.description." + loc, "", value})
				changed = true
			}
			if strings.TrimSpace(keywords[loc]) == "" {
				value := categoryKeywords(name, brand[loc], loc)
				keywords[loc] = value
				changes = append(changes, change{"category", category.Slug, "seo_meta.keywords." + loc, "", value})
				changed = true
			}
		}
		if !changed {
			continue
		}
		category.DescriptionJSON = jsonmap.JSON(toAnyMap(descriptions))
		seo["title"] = titles
		seo["description"] = seoDescriptions
		seo["keywords"] = keywords
		category.SeoMetaJSON = jsonmap.JSON(seo)
		if apply {
			if err := store.Update(category); err != nil {
				recordOperationError(fmt.Errorf("update category %s: %w", category.Slug, err))
			}
		}
	}
	return changes
}

type postCategoryDef struct {
	Slug string
	Name map[string]string
	Sort int
}

var postCategoryDefs = []postCategoryDef{
	{Slug: "network-basics", Name: map[string]string{"zh-CN": "网络基础", "zh-TW": "網路基礎", "en-US": "Network Basics"}, Sort: 40},
	{Slug: "live-ops", Name: map[string]string{"zh-CN": "直播运营", "zh-TW": "直播運營", "en-US": "Live Operations"}, Sort: 30},
	{Slug: "cross-border", Name: map[string]string{"zh-CN": "跨境电商", "zh-TW": "跨境電商", "en-US": "Cross-border E-commerce"}, Sort: 20},
	{Slug: "tutorials", Name: map[string]string{"zh-CN": "技术教程", "zh-TW": "技術教學", "en-US": "Tutorials"}, Sort: 10},
}

func ensurePostCategories(db *gorm.DB, ctx context.Context, apply bool) []change {
	store := contentgormstore.NewPostCategoryStore(db)
	existing, err := store.ListAll(ctx, nil)
	if err != nil {
		recordOperationError(fmt.Errorf("list post categories: %w", err))
		return nil
	}
	bySlug := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		bySlug[item.Slug] = struct{}{}
	}
	changes := make([]change, 0, len(postCategoryDefs))
	for _, def := range postCategoryDefs {
		if _, ok := bySlug[def.Slug]; ok {
			continue
		}
		changes = append(changes, change{"post_category", def.Slug, "create", "", def.Name["zh-CN"]})
		if !apply {
			continue
		}
		category := &contentdomain.PostCategory{
			Slug:      def.Slug,
			NameJSON:  jsonmap.JSON{"zh-CN": def.Name["zh-CN"], "zh-TW": def.Name["zh-TW"], "en-US": def.Name["en-US"]},
			IsActive:  true,
			SortOrder: def.Sort,
		}
		if err := store.Create(ctx, category); err != nil {
			recordOperationError(fmt.Errorf("create post category %s: %w", def.Slug, err))
		}
	}
	return changes
}

func assignPostCategories(db *gorm.DB, ctx context.Context, apply bool) []change {
	categoryStore := contentgormstore.NewPostCategoryStore(db)
	categories, err := categoryStore.ListAll(ctx, nil)
	if err != nil {
		recordOperationError(fmt.Errorf("list post categories: %w", err))
		return nil
	}
	idBySlug := make(map[string]uint, len(categories))
	for _, item := range categories {
		idBySlug[item.Slug] = item.ID
	}

	store := contentgormstore.NewPostStore(db)
	posts, _, err := store.List(ctx, contentcontract.PostQuery{Page: 1, PageSize: 100000})
	if err != nil {
		recordOperationError(fmt.Errorf("list posts: %w", err))
		return nil
	}
	changes := make([]change, 0, len(posts))
	for i := range posts {
		post := &posts[i]
		if post.CategoryID != nil {
			continue
		}
		slug := pickPostCategory(post)
		id, ok := idBySlug[slug]
		if !ok && !isDefinedPostCategory(slug) {
			continue
		}
		changes = append(changes, change{"post", post.Slug, "category_id", "", slug})
		if !apply {
			continue
		}
		if !ok {
			continue
		}
		categoryID := id
		post.CategoryID = &categoryID
		if err := store.Update(ctx, post); err != nil {
			recordOperationError(fmt.Errorf("update post %s: %w", post.Slug, err))
		}
	}
	return changes
}

func cleanNotices(db *gorm.DB, ctx context.Context, apply bool) []change {
	store := contentgormstore.NewPostStore(db)
	posts, _, err := store.List(ctx, contentcontract.PostQuery{Page: 1, PageSize: 100000, Type: constants.PostTypeNotice})
	if err != nil {
		recordOperationError(fmt.Errorf("list notice posts: %w", err))
		return nil
	}
	changes := make([]change, 0, len(posts))
	for i := range posts {
		post := &posts[i]
		changed := false
		summaries := copyJSONMap(post.SummaryJSON)
		contents := copyJSONMap(post.ContentJSON)
		for _, loc := range locales {
			oldSummary := stringValue(summaries[loc])
			newSummary := cleanNoticeText(oldSummary, loc)
			if newSummary != oldSummary {
				summaries[loc] = newSummary
				changes = append(changes, change{"notice", post.Slug, "summary." + loc, truncate(oldSummary, 40), truncate(newSummary, 40)})
				changed = true
			}
			oldContent := stringValue(contents[loc])
			newContent := cleanNoticeText(oldContent, loc)
			if newContent != oldContent {
				contents[loc] = newContent
				changes = append(changes, change{"notice", post.Slug, "content." + loc, truncate(oldContent, 40), truncate(newContent, 40)})
				changed = true
			}
		}
		if !changed {
			continue
		}
		post.SummaryJSON = jsonmap.JSON(summaries)
		post.ContentJSON = jsonmap.JSON(contents)
		if apply {
			if err := store.Update(ctx, post); err != nil {
				recordOperationError(fmt.Errorf("update notice %s: %w", post.Slug, err))
			}
		}
	}
	return changes
}

func cleanBanners(db *gorm.DB, ctx context.Context, apply bool) []change {
	store := contentgormstore.NewBannerStore(db)
	banners, _, err := store.List(ctx, contentcontract.BannerQuery{Page: 1, PageSize: 10000})
	if err != nil {
		recordOperationError(fmt.Errorf("list banners: %w", err))
		return nil
	}
	changes := make([]change, 0, len(banners))
	for i := range banners {
		banner := &banners[i]
		titles := copyJSONMap(banner.TitleJSON)
		subtitles := copyJSONMap(banner.SubtitleJSON)
		changed := false
		for _, loc := range locales {
			oldTitle := stringValue(titles[loc])
			newTitle := replaceRegionPhrases(oldTitle, "banner.title."+loc)
			if newTitle != oldTitle {
				titles[loc] = newTitle
				changes = append(changes, change{"banner", banner.Name, "title." + loc, truncate(oldTitle, 40), truncate(newTitle, 40)})
				changed = true
			}
			oldSubtitle := stringValue(subtitles[loc])
			newSubtitle := replaceRegionPhrases(oldSubtitle, "banner.subtitle."+loc)
			if newSubtitle != oldSubtitle {
				subtitles[loc] = newSubtitle
				changes = append(changes, change{"banner", banner.Name, "subtitle." + loc, truncate(oldSubtitle, 40), truncate(newSubtitle, 40)})
				changed = true
			}
		}
		if !changed {
			continue
		}
		banner.TitleJSON = jsonmap.JSON(titles)
		banner.SubtitleJSON = jsonmap.JSON(subtitles)
		if apply {
			if err := store.Update(ctx, banner); err != nil {
				recordOperationError(fmt.Errorf("update banner %s: %w", banner.Name, err))
			}
		}
	}
	return changes
}

func cleanSiteCopy(store *settingsgormstore.Store, siteConfig jsonmap.JSON, apply bool) []change {
	if len(siteConfig) == 0 {
		return nil
	}
	changes := make([]change, 0, 8)
	updated := copyJSONMap(siteConfig)

	var walk func(value interface{}, path string) interface{}
	walk = func(value interface{}, path string) interface{} {
		switch typed := value.(type) {
		case string:
			next := replaceRegionPhrases(typed, path)
			if next != typed {
				changes = append(changes, change{"site_config", path, "text", truncate(typed, 40), truncate(next, 40)})
			}
			return next
		case map[string]interface{}:
			for key, item := range typed {
				typed[key] = walk(item, path+"."+key)
			}
			return typed
		case []interface{}:
			for i, item := range typed {
				typed[i] = walk(item, path+"["+strconv.Itoa(i)+"]")
			}
			return typed
		default:
			return value
		}
	}
	updated = walk(updated, "site_config").(map[string]interface{})

	// 关于页服务列表里已下架的服务项直接移除。
	about := asMap(updated["about"])
	services := asMap(about["services"])
	if items, ok := services["items"].([]interface{}); ok {
		filtered := make([]interface{}, 0, len(items))
		for _, item := range items {
			if containsRetiredService(item) {
				changes = append(changes, change{"site_config", "about.services.items", "remove", truncate(fmt.Sprint(item), 40), ""})
				continue
			}
			filtered = append(filtered, item)
		}
		services["items"] = filtered
	}
	if len(changes) == 0 {
		return nil
	}
	if apply {
		if _, err := store.Upsert(constants.SettingKeySiteConfig, jsonmap.JSON(updated)); err != nil {
			recordOperationError(fmt.Errorf("update site config: %w", err))
		}
	}
	return changes
}

func seoBrand(siteConfig jsonmap.JSON, locale string) string {
	seo := asMap(siteConfig["seo"])
	brand := asMap(siteConfig["brand"])
	return firstNonEmpty(localizedText(seo["title"], locale), stringValue(brand["site_name"]), "Dujiao-Next")
}

func productSEOTitle(product *productdomain.Product, locale string) string {
	region := regionLabel(product.Country, locale)
	line := lineLabel(product.LineType, locale)
	if product.IPType == productdomain.IPTypeShared || strings.HasPrefix(product.Slug, "shared-ip") {
		spec := sharedSpec(product.Slug)
		switch locale {
		case "zh-TW":
			return fmt.Sprintf("%sGB 不限時流量包｜共享 IP", spec)
		case "en-US":
			return fmt.Sprintf("%sGB No-Expiry Pack | Shared IP", spec)
		default:
			return fmt.Sprintf("%sGB 不限时流量包｜共享 IP", spec)
		}
	}
	kind := ipTypeLabel(product.IPType, locale)
	switch locale {
	case "zh-TW":
		return fmt.Sprintf("%s%s IP｜%s", region, kind, line)
	case "en-US":
		return fmt.Sprintf("%s %s IP | %s", region, kind, line)
	default:
		return fmt.Sprintf("%s%s IP｜%s", region, kind, line)
	}
}

func productSEODescription(product *productdomain.Product, title, locale string) string {
	// 只使用该语言自己的描述；缺失时用该语言的模板，避免把简体描述塞进 zh-TW/en-US。
	if desc := stringValue(asMap(product.DescriptionJSON)[locale]); strings.TrimSpace(desc) != "" {
		return truncate(desc, 160)
	}
	switch locale {
	case "zh-TW":
		return title + "。查看價格、交付方式與適用場景。"
	case "en-US":
		return title + ". See pricing, delivery and use cases."
	default:
		return title + "。查看价格、交付方式与适用场景。"
	}
}

func productSEOKeywords(product *productdomain.Product, locale, brand string) string {
	region := regionLabel(product.Country, locale)
	kind := ipTypeLabel(product.IPType, locale)
	line := lineLabel(product.LineType, locale)
	return strings.Join([]string{region, kind + " IP", line, brand}, ",")
}

func categoryDescription(name string, count int64, locale string) string {
	switch locale {
	case "zh-TW":
		return fmt.Sprintf("%s：共 %d 個在售商品，支援按地區、IP 類型與線路類型篩選；頁面提供價格、交付方式與適用場景說明，方便按業務需求比較選擇。如需批量採購或定製線路，可透過頁面聯絡方式諮詢。", name, count)
	case "en-US":
		return fmt.Sprintf("%s: %d products on sale, filterable by region, IP type and line type, with pricing, delivery and use-case details. Contact us for bulk orders or custom routes.", name, count)
	default:
		return fmt.Sprintf("%s：共 %d 个在售商品，支持按地区、IP 类型与线路类型筛选；页面提供价格、交付方式与适用场景说明，方便按业务需求对比选择。如需批量采购或定制线路，可通过页面联系方式咨询。", name, count)
	}
}

func categorySEOTitle(name string) string {
	return name
}

func categoryKeywords(name, brand, locale string) string {
	extra := "静态住宅 IP"
	if locale == "zh-TW" {
		extra = "靜態住宅 IP"
	}
	if locale == "en-US" {
		extra = "static residential IP"
	}
	return strings.Join([]string{name, extra, brand}, ",")
}

func pickPostCategory(post *contentdomain.Post) string {
	text := localizedText(post.TitleJSON, "zh-CN") + localizedText(post.SummaryJSON, "zh-CN") + localizedText(post.ContentJSON, "zh-CN")
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "直播") || strings.Contains(lower, "tiktok") || strings.Contains(lower, "stream"):
		return "live-ops"
	case strings.Contains(lower, "跨境") || strings.Contains(lower, "电商") || strings.Contains(lower, "店群") || strings.Contains(lower, "store"):
		return "cross-border"
	case strings.Contains(lower, "教程") || strings.Contains(lower, "配置") || strings.Contains(lower, "客户端") || strings.Contains(lower, "设置"):
		return "tutorials"
	default:
		return "network-basics"
	}
}

func isDefinedPostCategory(slug string) bool {
	for _, def := range postCategoryDefs {
		if def.Slug == slug {
			return true
		}
	}
	return false
}

func cleanNoticeText(value, locale string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	result := value
	for _, keyword := range retiredServiceKeywords {
		// Go 的 RE2 不支持负向先行断言，先匹配整个 <li> 再按关键词过滤。
		result = liItemPattern.ReplaceAllStringFunc(result, func(item string) string {
			if strings.Contains(item, keyword) {
				return ""
			}
			return item
		})
	}
	if phrase := newRegionPhrase[locale]; phrase != "" {
		for _, old := range oldRegionPhrases {
			result = strings.ReplaceAll(result, old, phrase)
		}
	}
	return strings.TrimSpace(result)
}

func replaceRegionPhrases(value, path string) string {
	locale := ""
	for _, loc := range locales {
		if strings.Contains(path, loc) {
			locale = loc
			break
		}
	}
	phrase := newRegionPhrase[locale]
	if phrase == "" {
		phrase = newRegionPhrase["zh-CN"]
	}
	result := value
	for _, old := range oldRegionPhrases {
		result = strings.ReplaceAll(result, old, phrase)
	}
	return result
}

func containsRetiredService(value interface{}) bool {
	text := fmt.Sprint(value)
	for _, keyword := range retiredServiceKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func localizedText(value interface{}, locale string) string {
	m := asMap(value)
	if len(m) == 0 {
		return stringValue(value)
	}
	for _, key := range []string{locale, "zh-CN", "en-US"} {
		if text := strings.TrimSpace(stringValue(m[key])); text != "" {
			return text
		}
	}
	return ""
}

func asMap(value interface{}) map[string]interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return typed
	case jsonmap.JSON:
		return map[string]interface{}(typed)
	default:
		return map[string]interface{}{}
	}
}

func stringMap(value interface{}) map[string]string {
	result := map[string]string{}
	for key, item := range asMap(value) {
		result[key] = stringValue(item)
	}
	for _, loc := range locales {
		if _, ok := result[loc]; !ok {
			result[loc] = ""
		}
	}
	return result
}

func toAnyMap(source map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func copyJSONMap(value interface{}) map[string]interface{} {
	source := asMap(value)
	if len(source) == 0 {
		return map[string]interface{}{}
	}
	raw, err := json.Marshal(source)
	if err != nil {
		return map[string]interface{}{}
	}
	var cloned map[string]interface{}
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return map[string]interface{}{}
	}
	return cloned
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func regionLabel(code, locale string) string {
	item, ok := regionNames[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return strings.ToUpper(strings.TrimSpace(code))
	}
	switch locale {
	case "zh-TW":
		return item[1]
	case "en-US":
		return item[2]
	default:
		return item[0]
	}
}

func ipTypeLabel(ipType, locale string) string {
	switch ipType {
	case productdomain.IPTypeDatacenter:
		return map[string]string{"zh-CN": "机房", "zh-TW": "機房", "en-US": "Datacenter"}[locale]
	case productdomain.IPTypeISP:
		return map[string]string{"zh-CN": "ISP 住宅", "zh-TW": "ISP 住宅", "en-US": "ISP Residential"}[locale]
	case productdomain.IPTypeShared:
		return map[string]string{"zh-CN": "共享", "zh-TW": "共享", "en-US": "Shared"}[locale]
	default:
		return map[string]string{"zh-CN": "静态住宅", "zh-TW": "靜態住宅", "en-US": "Static Residential"}[locale]
	}
}

func lineLabel(lineType, locale string) string {
	switch lineType {
	case productdomain.LineTypeIEPL:
		return "IEPL"
	case productdomain.LineTypeShared:
		return map[string]string{"zh-CN": "共享", "zh-TW": "共享", "en-US": "Shared"}[locale]
	default:
		return map[string]string{"zh-CN": "直连", "zh-TW": "直連", "en-US": "Direct"}[locale]
	}
}

var sharedSpecPattern = regexp.MustCompile(`shared-ip-(\d+)g`)
var liItemPattern = regexp.MustCompile(`(?is)<li[^>]*>.*?</li>`)

func sharedSpec(slug string) string {
	match := sharedSpecPattern.FindStringSubmatch(slug)
	if len(match) == 2 {
		return match[1]
	}
	return "流量"
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}

func printChanges(changes []change) {
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Kind != changes[j].Kind {
			return changes[i].Kind < changes[j].Kind
		}
		return changes[i].Slug < changes[j].Slug
	})
	limit := len(changes)
	if limit > 40 {
		limit = 40
	}
	for _, item := range changes[:limit] {
		fmt.Printf("- [%s] %s %s: %q -> %q\n", item.Kind, item.Slug, item.Field, item.Old, item.New)
	}
	if len(changes) > limit {
		fmt.Printf("... 其余 %d 项省略（可用 -report 输出完整 CSV）\n", len(changes)-limit)
	}
}

func writeReport(path string, changes []change) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if err := writer.Write([]string{"kind", "slug", "field", "old", "new"}); err != nil {
		return err
	}
	for _, item := range changes {
		if err := writer.Write([]string{item.Kind, item.Slug, item.Field, item.Old, item.New}); err != nil {
			return err
		}
	}
	return writer.Error()
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
