package productwrite

import (
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	"github.com/dujiao-next/internal/shared/jsonslice"
)

// productMetaInput 用指针区分「字段未提交」与「显式清空」。
type productMetaInput struct {
	Country  *string
	Scenario *[]string
	IPType   *string
	LineType *string
}

// normalizeMetaInput 合并请求值与现有值，返回规范化后的结构化元数据。
func normalizeMetaInput(input productMetaInput, existing productdomain.Product) productdomain.ProductMeta {
	country := existing.Country
	if input.Country != nil {
		country = *input.Country
	}
	scenarios := []string(existing.Scenario)
	if input.Scenario != nil {
		scenarios = *input.Scenario
	}
	ipType := existing.IPType
	if input.IPType != nil {
		ipType = *input.IPType
	}
	lineType := existing.LineType
	if input.LineType != nil {
		lineType = *input.LineType
	}
	return productdomain.NormalizeProductMeta(country, scenarios, ipType, lineType)
}

func applyMetaToProduct(product *productdomain.Product, meta productdomain.ProductMeta) {
	if product == nil {
		return
	}
	product.Country = meta.Country
	product.Scenario = jsonslice.Strings(meta.Scenario)
	product.IPType = meta.IPType
	product.LineType = meta.LineType
}
