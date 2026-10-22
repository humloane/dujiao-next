package productdomain

import (
	"strings"
	"time"

	categorydomain "github.com/dujiao-next/internal/modules/catalog/category/domain"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/dujiao-next/internal/shared/jsonslice"
	"github.com/dujiao-next/internal/shared/money"
)

// Product 商品表
type Product struct {
	ID               uint                `gorm:"primarykey" json:"id"`                                           // 主键
	CategoryID       uint                `gorm:"not null;index" json:"category_id"`                              // 分类ID
	Slug             string              `gorm:"uniqueIndex;not null" json:"slug"`                               // 唯一标识
	SeoMetaJSON      jsonmap.JSON        `gorm:"type:json" json:"seo_meta"`                                      // SEO 元数据
	TitleJSON        jsonmap.JSON        `gorm:"type:json;not null" json:"title"`                                // 多语言标题
	DescriptionJSON  jsonmap.JSON        `gorm:"type:json" json:"description"`                                   // 多语言描述
	ContentJSON      jsonmap.JSON        `gorm:"type:json" json:"content"`                                       // 多语言详情（Markdown）
	InstructionsJSON jsonmap.JSON        `gorm:"type:json" json:"instructions"`                                  // 多语言交付后使用说明（仅订单详情可见）
	PriceAmount      money.Amount        `gorm:"type:decimal(20,2);not null;default:0" json:"price_amount"`      // 价格金额
	CostPriceAmount  money.Amount        `gorm:"type:decimal(20,2);not null;default:0" json:"cost_price_amount"` // 成本价（取最低活跃SKU成本价）
	WholesalePrices  WholesalePriceTiers `gorm:"type:json" json:"wholesale_prices"`                              // 批发价阶梯
	Images           jsonslice.Strings   `gorm:"type:json" json:"images"`                                        // 图片数组
	Tags             jsonslice.Strings   `gorm:"type:json" json:"tags"`                                          // 标签数组
	// 结构化元数据：用于门户级筛选、标题模板与 JSON-LD。
	// scenario 使用 JSON 数组保存，SQLite / PostgreSQL 均通过 LIKE 做轻量筛选
	//（数据量在商品目录级别，避免引入只有单一数据库支持的 JSON 查询语法）。
	Country              string            `gorm:"type:varchar(8);index" json:"country,omitempty"`                      // 国家/地区代码（US/GB/DE/JP/TW/PH/SHARED 等）
	Scenario             jsonslice.Strings `gorm:"type:json" json:"scenario"`                                           // 适用场景（live/ecommerce/store-group/ads/enterprise/test）
	IPType               string            `gorm:"type:varchar(24);index" json:"ip_type,omitempty"`                     // IP 类型（residential/datacenter/isp/shared）
	LineType             string            `gorm:"type:varchar(16);index" json:"line_type,omitempty"`                   // 线路类型（direct/iepl/shared）
	PurchaseType         string            `gorm:"type:varchar(20);not null;default:'member'" json:"purchase_type"`     // 购买身份（guest/member）
	MinPurchaseQuantity  int               `gorm:"not null;default:0" json:"min_purchase_quantity"`                     // 单次最小购买数量（0 表示不限制）
	MaxPurchaseQuantity  int               `gorm:"not null;default:0" json:"max_purchase_quantity"`                     // 单次最大购买数量（0 表示不限制）
	StockDisplayMode     string            `gorm:"type:varchar(20);not null;default:'exact'" json:"stock_display_mode"` // 公开库存展示模式（exact/status/range/hidden）
	FulfillmentType      string            `gorm:"type:varchar(20);not null;default:'manual'" json:"fulfillment_type"`  // 交付类型（auto/manual）
	ManualFormSchemaJSON jsonmap.JSON      `gorm:"type:json" json:"manual_form_schema"`                                 // 人工交付表单 schema
	ManualStockTotal     int               `gorm:"not null;default:0" json:"manual_stock_total"`                        // 手动剩余库存（-1 表示无限库存，>=0 表示当前可售数量）
	ManualStockLocked    int               `gorm:"not null;default:0" json:"manual_stock_locked"`                       // 手动库存占用量（待支付）
	ManualStockSold      int               `gorm:"not null;default:0" json:"manual_stock_sold"`                         // 手动库存已售量（支付成功后累加）
	PaymentChannelIDs    string            `gorm:"type:text" json:"payment_channel_ids"`                                // 允许的支付渠道ID（jsonmap.JSON数组字符串，空表示不限制）
	IsAffiliateEnabled   bool              `gorm:"not null;default:false;index" json:"is_affiliate_enabled"`            // 是否参与推广返利
	AutoStockAvailable   int64             `gorm:"-" json:"auto_stock_available"`                                       // 自动发货库存可用量（仅结构，不写入数据库）
	AutoStockTotal       int64             `gorm:"-" json:"auto_stock_total"`                                           // 自动发货库存总量（仅结构，不写入数据库）
	AutoStockLocked      int64             `gorm:"-" json:"auto_stock_locked"`                                          // 自动发货库存占用量（仅结构，不写入数据库）
	AutoStockSold        int64             `gorm:"-" json:"auto_stock_sold"`                                            // 自动发货库存已售量（仅结构，不写入数据库）
	IsMapped             bool              `gorm:"not null;default:false;index" json:"is_mapped"`                       // 是否为对接商品
	IsActive             bool              `gorm:"default:false;index" json:"is_active"`                                // 是否上架
	SortOrder            int               `gorm:"default:0;index" json:"sort_order"`                                   // 排序权重
	CreatedAt            time.Time         `gorm:"index" json:"created_at"`                                             // 创建时间
	UpdatedAt            time.Time         `json:"updated_at"`                                                          // 更新时间
	DeletedAt            *time.Time        `gorm:"index" json:"-"`                                                      // 软删除时间

	// 关联
	Category categorydomain.Category `gorm:"foreignKey:CategoryID" json:"category,omitempty"` // 分类信息
	SKUs     []ProductSKU            `gorm:"foreignKey:ProductID" json:"skus,omitempty"`      // SKU 列表
}

// TableName 指定表名
func (Product) TableName() string {
	return "products"
}

// 商品结构化元数据取值。前台筛选、后台表单与回填命令共用同一套规范化逻辑，
// 避免出现 "IEPL" / "iepl" / "store_group" 这类同义值把筛选切碎。
const (
	IPTypeResidential = "residential"
	IPTypeDatacenter  = "datacenter"
	IPTypeISP         = "isp"
	IPTypeShared      = "shared"

	LineTypeDirect = "direct"
	LineTypeIEPL   = "iepl"
	LineTypeShared = "shared"

	ScenarioLive       = "live"
	ScenarioEcommerce  = "ecommerce"
	ScenarioStoreGroup = "store-group"
	ScenarioAds        = "ads"
	ScenarioEnterprise = "enterprise"
	ScenarioTest       = "test"

	// CountryShared 用于没有单一国家归属的共享节点套餐。
	CountryShared = "SHARED"
)

var scenarioOrder = []string{
	ScenarioLive,
	ScenarioEcommerce,
	ScenarioStoreGroup,
	ScenarioAds,
	ScenarioEnterprise,
	ScenarioTest,
}

var scenarioAliases = map[string]string{
	"live":        ScenarioLive,
	"stream":      ScenarioLive,
	"streaming":   ScenarioLive,
	"ecommerce":   ScenarioEcommerce,
	"e-commerce":  ScenarioEcommerce,
	"shop":        ScenarioEcommerce,
	"store-group": ScenarioStoreGroup,
	"store_group": ScenarioStoreGroup,
	"storegroup":  ScenarioStoreGroup,
	"ads":         ScenarioAds,
	"ad":          ScenarioAds,
	"enterprise":  ScenarioEnterprise,
	"business":    ScenarioEnterprise,
	"test":        ScenarioTest,
	"testing":     ScenarioTest,
}

// ProductMeta 是商品结构化字段的规范化结果。
type ProductMeta struct {
	Country  string
	Scenario jsonslice.Strings
	IPType   string
	LineType string
}

// NormalizeProductMeta 统一清洗国家、场景、IP 类型和线路类型。
// 非法或未知值会被丢弃而不是报错：这些字段只影响展示与筛选，
// 不应让一个历史拼写错误阻止商品保存。
func NormalizeProductMeta(country string, scenario []string, ipType, lineType string) ProductMeta {
	return ProductMeta{
		Country:  NormalizeCountryCode(country),
		Scenario: NormalizeScenarios(scenario),
		IPType:   NormalizeIPType(ipType),
		LineType: NormalizeLineType(lineType),
	}
}

// NormalizeCountryCode 将国家代码规范为大写；允许 SHARED 与两位 ISO 代码。
func NormalizeCountryCode(raw string) string {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	if len(value) > 8 {
		return ""
	}
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return ""
		}
	}
	return value
}

// NormalizeIPType 将常见别名映射到固定取值。
func NormalizeIPType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case IPTypeResidential, "residental", "resi":
		return IPTypeResidential
	case IPTypeDatacenter, "data-center", "data_center", "data", "dc":
		return IPTypeDatacenter
	case IPTypeISP:
		return IPTypeISP
	case IPTypeShared, "share":
		return IPTypeShared
	default:
		return ""
	}
}

// NormalizeLineType 将线路类型映射为 direct / iepl / shared。
func NormalizeLineType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case LineTypeDirect, "public", "internet":
		return LineTypeDirect
	case LineTypeIEPL, "iepl-line", "iepl_line", "iplc":
		return LineTypeIEPL
	case LineTypeShared, "share":
		return LineTypeShared
	default:
		return ""
	}
}

// NormalizeScenarios 清洗场景数组：去重、转小写、按固定顺序输出。
func NormalizeScenarios(raw []string) jsonslice.Strings {
	if len(raw) == 0 {
		return jsonslice.Strings{}
	}
	selected := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		value := strings.ToLower(strings.TrimSpace(item))
		if canonical, ok := scenarioAliases[value]; ok {
			selected[canonical] = struct{}{}
		}
	}
	result := make(jsonslice.Strings, 0, len(selected))
	for _, scenario := range scenarioOrder {
		if _, ok := selected[scenario]; ok {
			result = append(result, scenario)
		}
	}
	return result
}

// ScenarioValues 返回场景数组的副本，避免调用方修改领域对象内部切片。
func (p Product) ScenarioValues() []string {
	if len(p.Scenario) == 0 {
		return nil
	}
	return append([]string(nil), p.Scenario...)
}
