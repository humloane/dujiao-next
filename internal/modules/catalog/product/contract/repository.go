package productcontract

import (
	"time"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
)

// ListFilter 描述 Catalog 商品列表的持久化筛选条件。
//
// IsActive 与 OnlyActive 都是上架状态条件，分工不同：OnlyActive 是公开侧强制的
// 「必须上架」开关；IsActive 是后台可选的三态筛选，nil 不限、true 仅已上架、false 仅已下架。
type ListFilter struct {
	Page               int
	PageSize           int
	CategoryID         string
	CategoryIDs        []uint
	ExcludeProductIDs  []uint
	Search             string
	FulfillmentType    string
	StockStatus        string
	Country            string
	Scenario           string
	IPType             string
	LineType           string
	Sort               string
	HasWholesalePrices *bool
	LowStockThreshold  int
	OnlyActive         bool
	IsActive           *bool
	WithCategory       bool
	UpdatedAfter       *time.Time
}

// PublicFilter 是公开商品列表的结构化筛选条件。
// 字段为空表示不筛选；调用方负责把用户输入规范化为领域取值。
type PublicFilter struct {
	Country  string
	Scenario string
	IPType   string
	LineType string
	Sort     string
}

// FilterOptions 是公开筛选器当前实际可用的取值。
//
// 国家/地区是开放取值（NormalizeCountryCode 允许任意国家代码与 SHARED），
// 不能在代码里写死；这里由在售商品聚合得到，新增地区无需改前端。
type FilterOptions struct {
	Countries []string
	Scenarios []string
	IPTypes   []string
	LineTypes []string
}

// Repository 定义商品领域需要的持久化能力；事务适配保留在 GORM Store。
type Repository interface {
	List(filter ListFilter) ([]productdomain.Product, int64, error)
	GetBySlug(slug string, onlyActive bool) (*productdomain.Product, error)
	GetByID(id string) (*productdomain.Product, error)
	GetAdminByID(id string) (*productdomain.Product, error)
	ListByIDs(ids []uint) ([]productdomain.Product, error)
	Create(item *productdomain.Product) error
	Update(item *productdomain.Product) error
	Delete(id string) error
	CountBySlug(slug string, excludeID *string) (int64, error)
	// ListFilterOptions 汇总公开筛选器的可用取值；excludeProductIDs 用于剔除
	// 当前分销租户不可见的商品，避免筛选项暴露未开放商品。
	ListFilterOptions(excludeProductIDs []uint) (FilterOptions, error)
	ReserveManualStock(productID uint, quantity int) (int64, error)
	ReleaseManualStock(productID uint, quantity int) (int64, error)
	ConsumeManualStock(productID uint, quantity int) (int64, error)
	QuickUpdate(id string, fields map[string]interface{}) error
}

// SKURepository 定义商品 SKU 领域需要的持久化能力。
type SKURepository interface {
	ListByProduct(productID uint, onlyActive bool) ([]productdomain.ProductSKU, error)
	GetByID(id uint) (*productdomain.ProductSKU, error)
	GetByProductAndCode(productID uint, skuCode string) (*productdomain.ProductSKU, error)
	ListByIDs(ids []uint) ([]productdomain.ProductSKU, error)
	Create(item *productdomain.ProductSKU) error
	CreateBatch(items []productdomain.ProductSKU) error
	Update(item *productdomain.ProductSKU) error
	Delete(id uint) error
	DeleteByProduct(productID uint) error
	PurgeSoftDeletedByProductAndCode(productID uint, skuCode string) error
	ReserveManualStock(skuID uint, quantity int) (int64, error)
	ReleaseManualStock(skuID uint, quantity int) (int64, error)
	ConsumeManualStock(skuID uint, quantity int) (int64, error)
}
