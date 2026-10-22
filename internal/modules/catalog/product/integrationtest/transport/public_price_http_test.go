package integrationtest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	promotiondomain "github.com/dujiao-next/internal/modules/promotion/domain"

	"github.com/dujiao-next/internal/constants"
	productcontract "github.com/dujiao-next/internal/modules/catalog/product/contract"
	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	producthttp "github.com/dujiao-next/internal/modules/catalog/product/transport/http"
	productpresenter "github.com/dujiao-next/internal/modules/catalog/product/transport/presenter"
	contentcontract "github.com/dujiao-next/internal/modules/content/contract"
	promotionapp "github.com/dujiao-next/internal/modules/promotion/application"
	promotiongormstore "github.com/dujiao-next/internal/modules/promotion/infrastructure/gormstore"
	reseller "github.com/dujiao-next/internal/modules/reseller/contract"
	"github.com/dujiao-next/internal/shared/money"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type staticPublicProductQueries struct {
	product productdomain.Product
}

func (s staticPublicProductQueries) ListPublicForTenant(reseller.TenantContext, string, string, int, int) ([]productdomain.Product, int64, error) {
	return []productdomain.Product{s.product}, 1, nil
}

func (s staticPublicProductQueries) ListPublicForTenantFiltered(reseller.TenantContext, productcontract.PublicFilter, string, string, int, int) ([]productdomain.Product, int64, error) {
	return []productdomain.Product{s.product}, 1, nil
}

func (s staticPublicProductQueries) GetPublicBySlugForTenant(reseller.TenantContext, string) (*productdomain.Product, error) {
	product := s.product
	return &product, nil
}

func (s staticPublicProductQueries) ListFilterOptionsForTenant(reseller.TenantContext) (productcontract.FilterOptions, error) {
	return productcontract.FilterOptions{
		Countries: []string{s.product.Country},
		IPTypes:   []string{s.product.IPType},
		LineTypes: []string{s.product.LineType},
	}, nil
}

func (staticPublicProductQueries) ApplyAutoStockCounts([]productdomain.Product) error {
	return nil
}

type emptyRelatedPostReader struct{}

func (emptyRelatedPostReader) ListPostsForProduct(context.Context, uint, int) ([]contentcontract.RelatedPost, error) {
	return nil, nil
}

func TestPublicProductHTTPPromotionUsesDisplayPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dsn := fmt.Sprintf("file:catalog_public_product_display_price_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&promotiondomain.Promotion{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	promotion := promotiondomain.Promotion{
		Name:       "fixed-10",
		ScopeType:  constants.ScopeTypeProduct,
		ScopeRefID: 1,
		Type:       constants.PromotionTypeFixed,
		Value:      money.FromDecimal(decimal.NewFromInt(10)),
		MinAmount:  money.FromDecimal(decimal.Zero),
		IsActive:   true,
	}
	if err := db.Create(&promotion).Error; err != nil {
		t.Fatalf("create promotion failed: %v", err)
	}

	queries := staticPublicProductQueries{product: productdomain.Product{
		ID:          1,
		Slug:        "display-price",
		PriceAmount: money.FromDecimal(decimal.RequireFromString("59.90")),
		SKUs: []productdomain.ProductSKU{
			{
				ID:          21,
				IsActive:    true,
				SortOrder:   100,
				PriceAmount: money.FromDecimal(decimal.RequireFromString("89.90")),
			},
			{
				ID:          22,
				IsActive:    true,
				SortOrder:   10,
				PriceAmount: money.FromDecimal(decimal.RequireFromString("49.90")),
			},
		},
	}}
	promotions := promotionapp.NewService(promotiongormstore.New(db))
	handler := producthttp.NewPublicHandler(queries, nil, promotions, nil, nil, nil, emptyRelatedPostReader{})

	router := gin.New()
	producthttp.RegisterPublicRoutes(router, handler)
	request := httptest.NewRequest(http.MethodGet, "/products/display-price", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var envelope struct {
		Data productpresenter.Product `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode product response: %v", err)
	}
	if envelope.Data.PromotionPriceAmount == nil {
		t.Fatal("expected promotion price amount")
	}

	expectedDisplay := decimal.RequireFromString("89.90")
	expectedPromotion := decimal.RequireFromString("79.90")
	if !envelope.Data.PriceAmount.Decimal.Equal(expectedDisplay) {
		t.Fatalf("expected display price %s, got %s", expectedDisplay, envelope.Data.PriceAmount.String())
	}
	if !envelope.Data.PromotionPriceAmount.Decimal.Equal(expectedPromotion) {
		t.Fatalf("expected promotion display price %s, got %s", expectedPromotion, envelope.Data.PromotionPriceAmount.String())
	}
}

// 公开筛选项接口：国家/地区等取值直接来自在售商品，新增地区不需要改前端。
func TestPublicProductFilterOptionsUsesProductData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queries := staticPublicProductQueries{product: productdomain.Product{
		ID:       1,
		Slug:     "sg-product",
		Country:  "SG",
		IPType:   "residential",
		LineType: "iepl",
		Scenario: []string{"live", "ads"},
	}}
	handler := producthttp.NewPublicHandler(queries, nil, nil, nil, nil, nil, emptyRelatedPostReader{})

	router := gin.New()
	producthttp.RegisterPublicRoutes(router, handler)
	request := httptest.NewRequest(http.MethodGet, "/product-filter-options", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Data struct {
			Countries []string `json:"countries"`
			IPTypes   []string `json:"ip_types"`
			LineTypes []string `json:"line_types"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.Countries) != 1 || body.Data.Countries[0] != "SG" {
		t.Fatalf("unexpected countries: %v", body.Data.Countries)
	}
	if len(body.Data.IPTypes) != 1 || body.Data.IPTypes[0] != "residential" {
		t.Fatalf("unexpected ip types: %v", body.Data.IPTypes)
	}
}
