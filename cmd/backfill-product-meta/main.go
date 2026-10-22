// Command backfill-product-meta 按 slug 规则回填商品结构化字段。
//
// 默认 dry-run，只打印将要写入的值；确认后执行：
//
//	go run ./cmd/backfill-product-meta -dry-run=false
//
// 规则与 openclaw/code-change-plan.md 对齐，并保持现有 slug/URL 不变：
//   - shared-ip-*                -> SHARED / shared / shared
//   - XX-Residential-IEPL        -> XX / residential / iepl
//   - XX-data-IP                 -> XX / datacenter / direct
//   - XX-Residential-IP          -> XX / residential / direct
//   - XX-IP                      -> XX / residential / direct
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	databasemigrations "github.com/dujiao-next/internal/bootstrap/database/migrations"
	"github.com/dujiao-next/internal/config"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	productgormstore "github.com/dujiao-next/internal/modules/catalog/product/store/gormstore"
	"github.com/dujiao-next/internal/platform/database/gormdb"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "只打印将要写入的值，不修改数据库")
	flag.Parse()

	cfg := config.Load()
	if err := gormdb.InitDB(cfg.Database.Driver, cfg.Database.DSN, gormdb.DBPoolConfig{
		MaxOpenConns:           cfg.Database.Pool.MaxOpenConns,
		MaxIdleConns:           cfg.Database.Pool.MaxIdleConns,
		ConnMaxLifetimeSeconds: cfg.Database.Pool.ConnMaxLifetimeSeconds,
		ConnMaxIdleTimeSeconds: cfg.Database.Pool.ConnMaxIdleTimeSeconds,
	}, cfg.Server.Mode); err != nil {
		fmt.Fprintf(os.Stderr, "init db: %v\n", err)
		os.Exit(1)
	}
	// 回填依赖新增列，命令必须与正常启动一样先跑 AutoMigrate。
	if err := databasemigrations.AutoMigrate(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate db: %v\n", err)
		os.Exit(1)
	}

	store := productgormstore.NewProductStore(gormdb.DB)
	products, total, err := store.List(productcontract.ListFilter{Page: 1, PageSize: 100000})
	if err != nil {
		fmt.Fprintf(os.Stderr, "list products: %v\n", err)
		os.Exit(1)
	}

	updated := 0
	skipped := 0
	for i := range products {
		product := products[i]
		meta := inferMeta(product.Slug)
		if meta.Country == "" && len(meta.Scenario) == 0 && meta.IPType == "" && meta.LineType == "" {
			skipped++
			continue
		}
		if product.Country == meta.Country &&
			equalStrings(product.ScenarioValues(), []string(meta.Scenario)) &&
			product.IPType == meta.IPType &&
			product.LineType == meta.LineType {
			skipped++
			continue
		}

		fmt.Printf("%s %s -> country=%s scenario=%v ip_type=%s line_type=%s\n",
			product.Slug, product.TitleJSON, meta.Country, meta.Scenario, meta.IPType, meta.LineType)
		if *dryRun {
			continue
		}
		scenarioJSON, err := json.Marshal([]string(meta.Scenario))
		if err != nil {
			fmt.Fprintf(os.Stderr, "encode scenario %s: %v\n", product.Slug, err)
			os.Exit(1)
		}
		if err := store.QuickUpdate(strconv.FormatUint(uint64(product.ID), 10), map[string]interface{}{
			"country":   meta.Country,
			"scenario":  string(scenarioJSON),
			"ip_type":   meta.IPType,
			"line_type": meta.LineType,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "update %s: %v\n", product.Slug, err)
			os.Exit(1)
		}
		updated++
	}

	mode := "dry-run"
	if !*dryRun {
		mode = "applied"
	}
	fmt.Printf("mode=%s total=%d updated=%d skipped=%d\n", mode, total, updated, skipped)
}

func inferMeta(slug string) productdomain.ProductMeta {
	lower := strings.ToLower(strings.TrimSpace(slug))
	if lower == "" {
		return productdomain.ProductMeta{}
	}

	var country string
	var ipType string
	var lineType string
	var scenarios []string

	if strings.HasPrefix(lower, "shared-ip") {
		country = productdomain.CountryShared
		ipType = productdomain.IPTypeShared
		lineType = productdomain.LineTypeShared
		scenarios = []string{productdomain.ScenarioEcommerce, productdomain.ScenarioTest}
	} else {
		country = countryFromSlug(lower)
		switch {
		case strings.Contains(lower, "residential-iepl"):
			ipType = productdomain.IPTypeResidential
			lineType = productdomain.LineTypeIEPL
			scenarios = []string{
				productdomain.ScenarioLive,
				productdomain.ScenarioEcommerce,
				productdomain.ScenarioStoreGroup,
				productdomain.ScenarioAds,
			}
		case strings.Contains(lower, "data-ip"):
			ipType = productdomain.IPTypeDatacenter
			lineType = productdomain.LineTypeDirect
			scenarios = []string{productdomain.ScenarioEcommerce, productdomain.ScenarioAds, productdomain.ScenarioTest}
		case strings.Contains(lower, "residential-ip"):
			ipType = productdomain.IPTypeResidential
			lineType = productdomain.LineTypeDirect
			scenarios = []string{productdomain.ScenarioLive, productdomain.ScenarioEcommerce}
		case strings.HasSuffix(lower, "-ip"):
			ipType = productdomain.IPTypeResidential
			lineType = productdomain.LineTypeDirect
			scenarios = []string{
				productdomain.ScenarioLive,
				productdomain.ScenarioEcommerce,
				productdomain.ScenarioStoreGroup,
			}
		}
	}

	return productdomain.NormalizeProductMeta(country, scenarios, ipType, lineType)
}

func countryFromSlug(slug string) string {
	parts := strings.Split(slug, "-")
	if len(parts) == 0 {
		return ""
	}
	code := strings.ToUpper(strings.TrimSpace(parts[0]))
	if len(code) != 2 {
		return ""
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return code
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	left := append([]string(nil), a...)
	right := append([]string(nil), b...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
