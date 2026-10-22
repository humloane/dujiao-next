package gormstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"

	"github.com/dujiao-next/internal/persistence/gormutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProductStore 是 Catalog Product 端口的 GORM 实现。
type ProductStore struct {
	db *gorm.DB
}

var _ productcontract.Repository = (*ProductStore)(nil)

func NewProductStore(db *gorm.DB) *ProductStore {
	return &ProductStore{db: db}
}

// BindTx 将 Store 绑定到调用方事务，并仅暴露 Product 端口。
func (r *ProductStore) BindTx(tx *gorm.DB) productcontract.Repository {
	if tx == nil {
		return r
	}
	return NewProductStore(tx)
}

func (r *ProductStore) Transaction(fn func(tx *gorm.DB) error) error {
	if fn == nil {
		return nil
	}
	return r.db.Transaction(fn)
}

// List 商品列表
func (r *ProductStore) List(filter productcontract.ListFilter) ([]productdomain.Product, int64, error) {
	var products []productdomain.Product

	query := r.db.Model(&productdomain.Product{}).Where("products.deleted_at IS NULL")
	if filter.WithCategory {
		query = query.Preload("Category", "deleted_at IS NULL")
	}
	if filter.OnlyActive {
		query = query.Where("products.is_active = ?", true)
		query = query.Where("EXISTS (SELECT 1 FROM categories c WHERE c.id = products.category_id AND c.is_active = ? AND c.deleted_at IS NULL)", true)
		query = query.Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL AND is_active = ?", true).Order("sort_order DESC, id ASC")
		})
	} else {
		query = query.Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("sort_order DESC, id ASC")
		})
	}
	if len(filter.CategoryIDs) > 0 {
		query = query.Where("category_id IN ?", filter.CategoryIDs)
	} else if filter.CategoryID != "" {
		query = query.Where("category_id = ?", filter.CategoryID)
	}
	// 三态上架筛选：OnlyActive 已强制 is_active = true，此时不再叠加
	if filter.IsActive != nil && !filter.OnlyActive {
		query = query.Where("products.is_active = ?", *filter.IsActive)
	}
	if len(filter.ExcludeProductIDs) > 0 {
		query = query.Where("products.id NOT IN ?", filter.ExcludeProductIDs)
	}
	if fulfillmentType := strings.TrimSpace(filter.FulfillmentType); fulfillmentType != "" {
		query = query.Where("fulfillment_type = ?", fulfillmentType)
	}
	if country := strings.TrimSpace(filter.Country); country != "" {
		query = query.Where("products.country = ?", country)
	}
	if ipType := strings.TrimSpace(filter.IPType); ipType != "" {
		query = query.Where("products.ip_type = ?", ipType)
	}
	if lineType := strings.TrimSpace(filter.LineType); lineType != "" {
		query = query.Where("products.line_type = ?", lineType)
	}
	if scenario := strings.TrimSpace(filter.Scenario); scenario != "" {
		// scenario 是 JSON 数组（SQLite TEXT / PostgreSQL jsonb）。
		// 用带引号的 LIKE 做 token 匹配，避免 live 命中 live-stream；
		// PostgreSQL 的 jsonb 需要先转 text 才能 LIKE。
		pattern := "%\"" + scenario + "\"%"
		if r.db.Dialector.Name() == "postgres" {
			query = query.Where("products.scenario::text LIKE ?", pattern)
		} else {
			// SQLite 的 JSON 列可能以 BLOB 存储，先 CAST 成 TEXT 再做 LIKE。
			query = query.Where("CAST(products.scenario AS TEXT) LIKE ?", pattern)
		}
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		condition, argCount := gormutil.BuildLocalizedLikeCondition(r.db, []string{"slug"}, []string{"title_json", "description_json"})
		searchQuery := r.db.Where(condition, gormutil.RepeatLikeArgs(like, argCount)...)

		skuCondition, skuArgCount := gormutil.BuildLocalizedLikeCondition(r.db, []string{"ps.sku_code"}, nil)
		searchQuery = searchQuery.Or(
			"EXISTS (SELECT 1 FROM product_skus ps WHERE ps.product_id = products.id AND ps.deleted_at IS NULL AND ("+skuCondition+"))",
			gormutil.RepeatLikeArgs(like, skuArgCount)...,
		)

		if numericID, err := strconv.ParseUint(search, 10, 64); err == nil && numericID > 0 {
			searchQuery = searchQuery.Or("id = ?", uint(numericID))
		}
		query = query.Where(searchQuery)
	}

	if filter.UpdatedAfter != nil {
		// 库存变化落在 SKU（人工库存预占/售出）与卡密行上，不会改动商品行本身；
		// 下游增量同步依赖此过滤，需一并纳入，否则库存变化要等每日全量同步才可见。
		after := *filter.UpdatedAfter
		query = query.Where("(products.updated_at > ? OR "+
			"EXISTS (SELECT 1 FROM product_skus ps WHERE ps.product_id = products.id AND ps.updated_at > ?) OR "+
			"EXISTS (SELECT 1 FROM card_secrets cs WHERE cs.product_id = products.id AND cs.updated_at > ?))",
			after, after, after)
	}

	stockStatus := strings.ToLower(strings.TrimSpace(filter.StockStatus))
	query = applyStockStatusFilter(query, stockStatus, filter.LowStockThreshold)
	if filter.HasWholesalePrices != nil {
		expr := gormutil.JSONArrayLengthExpr(r.db, "wholesale_prices")
		if *filter.HasWholesalePrices {
			query = query.Where(expr + " > 0")
		} else {
			query = query.Where(expr + " = 0")
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = gormutil.ApplyPagination(query, filter.Page, filter.PageSize)
	query = query.Order(productListOrder(filter.Sort))

	if err := query.Find(&products).Error; err != nil {
		return nil, 0, err
	}

	return products, total, nil
}

// productListOrder 把公开列表排序参数映射为固定 SQL，禁止直接拼接用户输入。
func productListOrder(sort string) string {
	switch sort {
	case "price_asc":
		return "products.price_amount ASC, products.sort_order DESC, products.id DESC"
	case "price_desc":
		return "products.price_amount DESC, products.sort_order DESC, products.id DESC"
	case "newest":
		return "products.created_at DESC, products.id DESC"
	default:
		return "products.sort_order DESC, products.created_at DESC, products.id DESC"
	}
}

func applyStockStatusFilter(query *gorm.DB, status string, lowStockThreshold int) *gorm.DB {
	if query == nil || status == "" {
		return query
	}
	if lowStockThreshold < 0 {
		lowStockThreshold = 0
	}

	// manual 库存子查询
	const manualActiveSKUExists = "EXISTS (SELECT 1 FROM product_skus ps WHERE ps.product_id = products.id AND ps.is_active = true AND ps.deleted_at IS NULL)"
	const manualUnlimitedSKUExists = "EXISTS (SELECT 1 FROM product_skus ps WHERE ps.product_id = products.id AND ps.is_active = true AND ps.deleted_at IS NULL AND ps.manual_stock_total = -1)"
	const manualSKURemaining = "COALESCE((SELECT SUM(CASE WHEN ps.manual_stock_total > 0 THEN ps.manual_stock_total ELSE 0 END) FROM product_skus ps WHERE ps.product_id = products.id AND ps.is_active = true AND ps.deleted_at IS NULL), 0)"

	// auto 库存子查询（可用卡密数）
	const autoStockCount = "COALESCE((SELECT COUNT(*) FROM card_secrets cs WHERE cs.product_id = products.id AND cs.status = 'available' AND cs.deleted_at IS NULL), 0)"

	// upstream 库存子查询（通过 product_mappings + sku_mappings）
	const upstreamUnlimitedExists = "EXISTS (SELECT 1 FROM product_mappings pm JOIN sku_mappings sm ON sm.product_mapping_id = pm.id AND sm.deleted_at IS NULL WHERE pm.local_product_id = products.id AND pm.deleted_at IS NULL AND sm.upstream_stock = -1)"
	const upstreamStockSum = "COALESCE((SELECT SUM(CASE WHEN sm.upstream_stock > 0 THEN sm.upstream_stock ELSE 0 END) FROM product_mappings pm JOIN sku_mappings sm ON sm.product_mapping_id = pm.id AND sm.deleted_at IS NULL WHERE pm.local_product_id = products.id AND pm.deleted_at IS NULL), 0)"

	switch status {
	case "low":
		// manual: 非无限且剩余 <= 0 | auto: 可用卡密位于 [0, 低库存阈值] | upstream: 非无限且库存和 = 0
		condition := fmt.Sprintf("("+
			"(fulfillment_type = 'manual' AND (((%s) AND NOT (%s) AND (%s) <= 0) OR (NOT (%s) AND manual_stock_total = 0)))"+
			" OR (fulfillment_type = 'auto' AND (%s) >= 0 AND (%s) <= ?)"+
			" OR (fulfillment_type = 'upstream' AND NOT (%s) AND (%s) = 0)"+
			")",
			manualActiveSKUExists, manualUnlimitedSKUExists, manualSKURemaining, manualActiveSKUExists,
			autoStockCount, autoStockCount,
			upstreamUnlimitedExists, upstreamStockSum,
		)
		return query.Where(condition, lowStockThreshold)
	case "normal":
		// manual: 非无限且剩余 > 0 | auto: 可用卡密 > 低库存阈值 | upstream: 非无限且库存和 > 0
		condition := fmt.Sprintf("("+
			"(fulfillment_type = 'manual' AND (((%s) AND NOT (%s) AND (%s) > 0) OR (NOT (%s) AND manual_stock_total > 0)))"+
			" OR (fulfillment_type = 'auto' AND (%s) > ?)"+
			" OR (fulfillment_type = 'upstream' AND NOT (%s) AND (%s) > 0)"+
			")",
			manualActiveSKUExists, manualUnlimitedSKUExists, manualSKURemaining, manualActiveSKUExists,
			autoStockCount,
			upstreamUnlimitedExists, upstreamStockSum,
		)
		return query.Where(condition, lowStockThreshold)
	case "unlimited":
		// manual: 有无限 SKU | upstream: 有无限库存的映射
		condition := fmt.Sprintf("("+
			"(fulfillment_type = 'manual' AND ((%s) OR (NOT (%s) AND manual_stock_total = -1)))"+
			" OR (fulfillment_type = 'upstream' AND (%s))"+
			")",
			manualUnlimitedSKUExists, manualActiveSKUExists,
			upstreamUnlimitedExists,
		)
		return query.Where(condition)
	default:
		return query
	}
}

// GetBySlug 根据 slug 获取商品
func (r *ProductStore) GetBySlug(slug string, onlyActive bool) (*productdomain.Product, error) {
	query := r.db.Preload("Category", "deleted_at IS NULL").Where("products.deleted_at IS NULL AND products.slug = ?", slug)
	if onlyActive {
		query = query.Where("products.is_active = ?", true)
		query = query.Where("EXISTS (SELECT 1 FROM categories c WHERE c.id = products.category_id AND c.is_active = ? AND c.deleted_at IS NULL)", true)
		query = query.Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL AND is_active = ?", true).Order("sort_order DESC, id ASC")
		})
	} else {
		query = query.Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("sort_order DESC, id ASC")
		})
	}

	var product productdomain.Product
	if err := query.First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

// GetByID 根据 ID 获取商品
func (r *ProductStore) GetByID(id string) (*productdomain.Product, error) {
	pk, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, nil
	}
	var product productdomain.Product
	if err := r.db.Preload("Category", "deleted_at IS NULL").
		Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL AND is_active = ?", true).Order("sort_order DESC, id ASC")
		}).
		Where("products.deleted_at IS NULL").
		First(&product, pk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

// GetAdminByID 根据 ID 获取后台商品详情，包含全部 SKU
func (r *ProductStore) GetAdminByID(id string) (*productdomain.Product, error) {
	pk, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return nil, nil
	}
	var product productdomain.Product
	if err := r.db.Preload("Category", "deleted_at IS NULL").
		Preload("SKUs", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("sort_order DESC, id ASC")
		}).
		Where("products.deleted_at IS NULL").
		First(&product, pk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &product, nil
}

// ListFilterOptions 汇总公开筛选器实际可用的取值。
//
// 国家/地区是开放取值，新增地区不应要求改前端常量，所以这里按在售商品实时聚合。
// 场景以 JSON 数组保存，SQLite / PostgreSQL 无法共用一条 DISTINCT SQL，
// 因此读取该列后在 Go 里去重；商品目录量级下成本可接受。
func (r *ProductStore) ListFilterOptions(excludeProductIDs []uint) (productcontract.FilterOptions, error) {
	options := productcontract.FilterOptions{
		Countries: []string{},
		Scenarios: []string{},
		IPTypes:   []string{},
		LineTypes: []string{},
	}

	base := func() *gorm.DB {
		query := r.db.Model(&productdomain.Product{}).
			Where("products.deleted_at IS NULL").
			Where("products.is_active = ?", true).
			Where("EXISTS (SELECT 1 FROM categories c WHERE c.id = products.category_id AND c.is_active = ? AND c.deleted_at IS NULL)", true)
		if len(excludeProductIDs) > 0 {
			query = query.Where("products.id NOT IN ?", excludeProductIDs)
		}
		return query
	}

	if err := base().Where("products.country <> ''").
		Distinct().
		Pluck("products.country", &options.Countries).Error; err != nil {
		return productcontract.FilterOptions{}, err
	}
	if err := base().Where("products.ip_type <> ''").
		Distinct().
		Pluck("products.ip_type", &options.IPTypes).Error; err != nil {
		return productcontract.FilterOptions{}, err
	}
	if err := base().Where("products.line_type <> ''").
		Distinct().
		Pluck("products.line_type", &options.LineTypes).Error; err != nil {
		return productcontract.FilterOptions{}, err
	}

	scenarioColumn := scenarioTextColumn(r.db.Dialector.Name())
	rawScenarios := make([]string, 0, len(options.Countries)+4)
	if err := base().Where("products.scenario IS NOT NULL").
		Pluck(scenarioColumn, &rawScenarios).Error; err != nil {
		return productcontract.FilterOptions{}, err
	}
	rawScenarioValues := make([]string, 0, len(rawScenarios))
	for _, raw := range rawScenarios {
		var values []string
		if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
			continue
		}
		rawScenarioValues = append(rawScenarioValues, values...)
	}

	// 统一清洗大小写与别名：历史数据里的 "us" / "Residential" / "STORE_GROUP"
	// 不能在筛选器里出现第二份取值。
	options.Countries = normalizeFilterValues(options.Countries, productdomain.NormalizeCountryCode)
	// SHARED 是历史遗留的“共享节点”占位值，不是国家/地区代码。
	// 数据里即使还有存量值，也不要在公开筛选器里暴露成国家选项。
	if len(options.Countries) > 0 {
		countries := make([]string, 0, len(options.Countries))
		for _, code := range options.Countries {
			if code == productdomain.CountryShared {
				continue
			}
			countries = append(countries, code)
		}
		options.Countries = countries
	}
	options.IPTypes = normalizeFilterValues(options.IPTypes, productdomain.NormalizeIPType)
	options.LineTypes = normalizeFilterValues(options.LineTypes, productdomain.NormalizeLineType)
	// 场景保留领域定义的固定顺序，未知取值丢弃（后端筛选同样会归一化掉）。
	options.Scenarios = append([]string{}, productdomain.NormalizeScenarios(rawScenarioValues)...)
	return options, nil
}

// scenarioTextColumn 返回把 JSON 数组列读成文本的方言表达式。
// SQLite 的 JSON 列可能以 BLOB 存放，需要 CAST；PostgreSQL 的 jsonb 用 ::text。
func scenarioTextColumn(dialect string) string {
	if strings.EqualFold(strings.TrimSpace(dialect), "postgres") {
		return "products.scenario::text"
	}
	return "CAST(products.scenario AS TEXT)"
}

// normalizeFilterValues 对聚合出的取值做归一化、去空并按字母序去重。
func normalizeFilterValues(values []string, normalize func(string) string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := normalize(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// ListByIDs 批量获取商品
func (r *ProductStore) ListByIDs(ids []uint) ([]productdomain.Product, error) {
	if len(ids) == 0 {
		return []productdomain.Product{}, nil
	}
	var products []productdomain.Product
	if err := r.db.Where("deleted_at IS NULL AND id IN ?", ids).Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

// Create 创建商品
func (r *ProductStore) Create(product *productdomain.Product) error {
	return r.db.Create(product).Error
}

// Update 更新商品自身字段。
//
// 不级联保存关联：调用方拿到的 product 往往带着读取时预加载的 Category/SKUs 快照，
// 而 SKU 删除是硬删除，级联 upsert 会把已删除的规格按原 ID 重新插回（issue #344）。
// SKU 一律由 SKURepository 独占管理。
func (r *ProductStore) Update(product *productdomain.Product) error {
	return r.db.Omit(clause.Associations).Save(product).Error
}

// QuickUpdate 快速更新商品指定字段
func (r *ProductStore) QuickUpdate(id string, fields map[string]interface{}) error {
	return r.db.Model(&productdomain.Product{}).Where("id = ? AND deleted_at IS NULL", id).Updates(fields).Error
}

// Delete 删除商品
func (r *ProductStore) Delete(id string) error {
	return r.db.Model(&productdomain.Product{}).Where("id = ? AND deleted_at IS NULL", id).Update("deleted_at", time.Now()).Error
}

// CountBySlug 统计 slug 数量（含软删除记录：products.slug 唯一索引同样约束已删除商品）
func (r *ProductStore) CountBySlug(slug string, excludeID *string) (int64, error) {
	var count int64
	query := r.db.Model(&productdomain.Product{}).Where("slug = ?", slug)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ReserveManualStock 预占手动库存
func (r *ProductStore) ReserveManualStock(productID uint, quantity int) (int64, error) {
	return gormutil.ReserveManualStock(r.db, &productdomain.Product{}, productID, quantity)
}

// ReleaseManualStock 释放手动库存占用
func (r *ProductStore) ReleaseManualStock(productID uint, quantity int) (int64, error) {
	return gormutil.ReleaseManualStock(r.db, &productdomain.Product{}, productID, quantity)
}

// ConsumeManualStock 消耗手动库存（支付成功后占用转已售）
func (r *ProductStore) ConsumeManualStock(productID uint, quantity int) (int64, error) {
	return gormutil.ConsumeManualStock(r.db, &productdomain.Product{}, productID, quantity)
}
