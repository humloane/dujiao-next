package productapplication

import (
	"strconv"
	"strings"
	"time"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"

	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"

	cardsecretcontract "github.com/dujiao-next/internal/modules/cardsecret/contract"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
)

// ProductRepository 是读取应用服务所需的最小商品持久化端口。
type ProductRepository interface {
	List(filter productcontract.ListFilter) ([]productdomain.Product, int64, error)
	ListFilterOptions(excludeProductIDs []uint) (productcontract.FilterOptions, error)
	GetBySlug(slug string, onlyActive bool) (*productdomain.Product, error)
	GetAdminByID(id string) (*productdomain.Product, error)
}

// CategoryRepository 是公开分类展开所需的最小分类端口。
type CategoryRepository interface {
	GetByID(id string) (*categorydomain.Category, error)
	List() ([]categorydomain.Category, error)
}

// HiddenProductRepository 是商品读取用例所需的最小分销可见性端口。
type HiddenProductRepository interface {
	ListHiddenProductIDs(resellerID uint) ([]uint, error)
}

// StockCounter 是商品库存聚合所需的最小卡密库存端口。
type StockCounter interface {
	CountStockByProductIDs(productIDs []uint) ([]cardsecretcontract.SKUStockCount, error)
}

// Options 描述 Product 读取应用服务的端口和领域错误。
type Options struct {
	Products   ProductRepository
	Categories CategoryRepository
	Stock      StockCounter
}

// Service 编排商品查询、租户可见性和库存聚合。
type Service struct {
	products   ProductRepository
	categories CategoryRepository
	stock      StockCounter
}

// NewService 创建 Product 读取应用服务。
func NewService(options Options) *Service {
	return &Service{
		products:   options.Products,
		categories: options.Categories,
		stock:      options.Stock,
	}
}

// ListPublic 获取公开商品列表。
func (s *Service) ListPublic(categoryID, search string, page, pageSize int) ([]productdomain.Product, int64, error) {
	return s.ListPublicFiltered(categoryID, search, productcontract.PublicFilter{}, page, pageSize)
}

// ListPublicFiltered 获取带结构化筛选的公开商品列表。
func (s *Service) ListPublicFiltered(categoryID, search string, publicFilter productcontract.PublicFilter, page, pageSize int) ([]productdomain.Product, int64, error) {
	categoryIDs, err := expandPublicCategoryIDs(s.categories, categoryID)
	if err != nil {
		return nil, 0, err
	}

	filter := buildPublicListFilter(publicFilter)
	filter.Page = page
	filter.PageSize = pageSize
	filter.CategoryID = categoryID
	filter.CategoryIDs = categoryIDs
	filter.Search = search
	filter.OnlyActive = true
	filter.WithCategory = true
	return s.products.List(filter)
}

// ListPublicForTenant 获取当前租户上下文的公开商品列表。
func (s *Service) ListPublicForTenant(tenant reseller.TenantContext, resellerRepo HiddenProductRepository, categoryID, search string, page, pageSize int) ([]productdomain.Product, int64, error) {
	return s.ListPublicForTenantFiltered(tenant, resellerRepo, productcontract.PublicFilter{}, categoryID, search, page, pageSize)
}

// ListPublicForTenantFiltered 获取当前租户上下文、带结构化筛选的公开商品列表。
func (s *Service) ListPublicForTenantFiltered(tenant reseller.TenantContext, resellerRepo HiddenProductRepository, publicFilter productcontract.PublicFilter, categoryID, search string, page, pageSize int) ([]productdomain.Product, int64, error) {
	if !tenant.IsReseller() {
		return s.ListPublicFiltered(categoryID, search, publicFilter, page, pageSize)
	}
	if tenant.ResellerID == nil || resellerRepo == nil {
		return nil, 0, productcontract.ErrResellerProductNotListed
	}
	categoryIDs, err := expandPublicCategoryIDs(s.categories, categoryID)
	if err != nil {
		return nil, 0, err
	}
	hiddenIDs, err := resellerRepo.ListHiddenProductIDs(*tenant.ResellerID)
	if err != nil {
		return nil, 0, err
	}
	filter := buildPublicListFilter(publicFilter)
	filter.Page = page
	filter.PageSize = pageSize
	filter.CategoryID = categoryID
	filter.CategoryIDs = categoryIDs
	filter.Search = search
	filter.OnlyActive = true
	filter.WithCategory = true
	filter.ExcludeProductIDs = hiddenIDs
	return s.products.List(filter)
}

// ListFilterOptionsForTenant 返回当前租户可见商品的筛选项。
// 筛选器取值完全来自在售商品，新增国家/地区不需要改前端常量。
func (s *Service) ListFilterOptionsForTenant(tenant reseller.TenantContext, resellerRepo HiddenProductRepository) (productcontract.FilterOptions, error) {
	if !tenant.IsReseller() {
		return s.products.ListFilterOptions(nil)
	}
	if tenant.ResellerID == nil || resellerRepo == nil {
		return productcontract.FilterOptions{}, productcontract.ErrResellerProductNotListed
	}
	hiddenIDs, err := resellerRepo.ListHiddenProductIDs(*tenant.ResellerID)
	if err != nil {
		return productcontract.FilterOptions{}, err
	}
	return s.products.ListFilterOptions(hiddenIDs)
}

// buildPublicListFilter 把 HTTP 层传入的原始筛选值规范化为领域取值。
// 未识别的值会被丢弃，避免筛选条件把整个商品列表清空。
func buildPublicListFilter(input productcontract.PublicFilter) productcontract.ListFilter {
	filter := productcontract.ListFilter{
		Country:  productdomain.NormalizeCountryCode(input.Country),
		IPType:   productdomain.NormalizeIPType(input.IPType),
		LineType: productdomain.NormalizeLineType(input.LineType),
	}
	// scenario 仅为旧客户端兼容保留；前台已移除场景筛选，不要在新代码中使用。
	if scenarios := productdomain.NormalizeScenarios([]string{input.Scenario}); len(scenarios) > 0 {
		filter.Scenario = scenarios[0]
	}
	switch strings.ToLower(strings.TrimSpace(input.Sort)) {
	case "price_asc", "price-asc":
		filter.Sort = "price_asc"
	case "price_desc", "price-desc":
		filter.Sort = "price_desc"
	case "newest", "latest":
		filter.Sort = "newest"
	default:
		filter.Sort = "recommended"
	}
	return filter
}

// ListForUpstreamSync 上游同步专用：可选包含已下架商品，便于下游识别下架状态
// includeInactive=true 时返回所有未软删商品（含 is_active=false）
func (s *Service) ListForUpstreamSync(updatedAfter *time.Time, includeInactive bool, page, pageSize int) ([]productdomain.Product, int64, error) {
	filter := productcontract.ListFilter{
		Page:         page,
		PageSize:     pageSize,
		OnlyActive:   !includeInactive,
		WithCategory: true,
		UpdatedAfter: updatedAfter,
	}
	return s.products.List(filter)
}

// ListPublicExact 获取公开商品列表（精确匹配分类，不展开父分类）
func (s *Service) ListPublicExact(categoryID string, page, pageSize int) ([]productdomain.Product, int64, error) {
	filter := productcontract.ListFilter{
		Page:         page,
		PageSize:     pageSize,
		CategoryID:   categoryID,
		OnlyActive:   true,
		WithCategory: true,
	}
	return s.products.List(filter)
}

// GetPublicBySlug 获取公开商品详情
func (s *Service) GetPublicBySlug(slug string) (*productdomain.Product, error) {
	item, err := s.products.GetBySlug(slug, true)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, productcontract.ErrNotFound
	}
	return item, nil
}

// GetPublicBySlugForTenant 获取当前租户上下文的公开商品详情。
func (s *Service) GetPublicBySlugForTenant(tenant reseller.TenantContext, resellerRepo HiddenProductRepository, slug string) (*productdomain.Product, error) {
	item, err := s.GetPublicBySlug(slug)
	if err != nil {
		return nil, err
	}
	if !tenant.IsReseller() {
		return item, nil
	}
	if tenant.ResellerID == nil || resellerRepo == nil {
		return nil, productcontract.ErrNotFound
	}
	hiddenIDs, err := resellerRepo.ListHiddenProductIDs(*tenant.ResellerID)
	if err != nil {
		return nil, err
	}
	for _, id := range hiddenIDs {
		if id == item.ID {
			return nil, productcontract.ErrNotFound
		}
	}
	return item, nil
}

// ListAdmin 获取后台商品列表
// isActive 为三态上架筛选：nil=不限，true=仅已上架，false=仅已下架。
func (s *Service) ListAdmin(categoryID, search, fulfillmentType, stockStatus string, hasWholesalePrices, isActive *bool, lowStockThreshold int, page, pageSize int) ([]productdomain.Product, int64, error) {
	filter := productcontract.ListFilter{
		Page:               page,
		PageSize:           pageSize,
		CategoryID:         categoryID,
		Search:             search,
		FulfillmentType:    strings.TrimSpace(fulfillmentType),
		StockStatus:        normalizeStockStatus(stockStatus),
		HasWholesalePrices: hasWholesalePrices,
		IsActive:           isActive,
		LowStockThreshold:  lowStockThreshold,
		OnlyActive:         false,
		WithCategory:       true,
	}
	return s.products.List(filter)
}

// GetAdminByID 获取后台商品详情
func (s *Service) GetAdminByID(id string) (*productdomain.Product, error) {
	item, err := s.products.GetAdminByID(id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, productcontract.ErrNotFound
	}
	return item, nil
}

func normalizeStockStatus(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", "all":
		return ""
	case "low", "normal", "unlimited":
		return value
	default:
		return ""
	}
}

func expandPublicCategoryIDs(categoryRepo CategoryRepository, categoryID string) ([]uint, error) {
	normalizedCategoryID := strings.TrimSpace(categoryID)
	if normalizedCategoryID == "" {
		return nil, nil
	}

	parsedCategoryID, err := strconv.ParseUint(normalizedCategoryID, 10, 64)
	if err != nil || parsedCategoryID == 0 {
		return nil, nil
	}
	if categoryRepo == nil {
		return []uint{uint(parsedCategoryID)}, nil
	}

	category, err := categoryRepo.GetByID(normalizedCategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return []uint{uint(parsedCategoryID)}, nil
	}
	if !category.IsActive {
		return []uint{}, nil
	}
	if category.ParentID > 0 {
		return []uint{category.ID}, nil
	}

	categories, err := categoryRepo.List()
	if err != nil {
		return nil, err
	}

	categoryIDs := []uint{category.ID}
	for _, item := range categories {
		if item.ParentID == category.ID && item.IsActive {
			categoryIDs = append(categoryIDs, item.ID)
		}
	}
	return categoryIDs, nil
}
