package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	productdomain "github.com/dujiao-next/internal/modules/catalog/product/domain"
	"github.com/dujiao-next/internal/platform/database/gormdb"
	"github.com/dujiao-next/internal/shared/jsonmap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDedupeContentRemovesRepeatedSections(t *testing.T) {
	html := `<p>intro</p>` +
		`<h2>核心亮点</h2><ul><li>A1</li></ul>` +
		`<h2>适用场景</h2><ul><li>B1</li></ul>` +
		`<h2>核心亮点</h2><ul><li>A2</li></ul>` +
		`<h2>交付说明</h2><p>C1</p>`
	got, removed, labels := dedupeContent(html)
	if removed == 0 {
		t.Fatalf("expected duplicate sections to be removed")
	}
	if strings.Contains(got, "A2") {
		t.Fatalf("duplicate block should be removed: %q", got)
	}
	for _, want := range []string{"intro", "A1", "B1", "C1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("content %q missing %q", got, want)
		}
	}
	if len(labels) != 1 || labels[0] != "核心亮点" {
		t.Fatalf("unexpected removed labels: %v", labels)
	}
}

func TestDedupeContentKeepsUniqueSections(t *testing.T) {
	html := `<h2>核心亮点</h2><p>A</p><h2>适用场景</h2><p>B</p><h2>交付说明</h2><p>C</p>`
	got, removed, _ := dedupeContent(html)
	if removed != 0 || got != html {
		t.Fatalf("unique content must stay unchanged")
	}
}

func TestDedupeContentDetectsDuplicatedHalf(t *testing.T) {
	half := `<p>同一段正文</p>`
	got, removed, labels := dedupeContent(half + half)
	if removed == 0 || got != half {
		t.Fatalf("expected duplicated half to be removed, got %q", got)
	}
	if len(labels) != 1 || labels[0] != "duplicate-half" {
		t.Fatalf("unexpected labels: %v", labels)
	}
}

func TestMigrateOnlyWhenRequested(t *testing.T) {
	oldDB := gormdb.DB
	defer func() { gormdb.DB = oldDB }()

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "schema.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	gormdb.DB = db

	if err := migrateIfRequested(false, false); err != nil {
		t.Fatalf("dry-run should not migrate: %v", err)
	}
	if db.Migrator().HasTable("products") {
		t.Fatal("dry-run created products table")
	}

	if err := migrateIfRequested(true, false); err != nil {
		t.Fatalf("apply migration failed: %v", err)
	}
	if !db.Migrator().HasTable("products") {
		t.Fatal("apply did not create products table")
	}
}

func TestFailureExitCode(t *testing.T) {
	operationErrors = []error{errors.New("update failed")}
	if got := failureExitCode(true, false); got != 1 {
		t.Fatalf("apply failure exit = %d, want 1", got)
	}
	if got := failureExitCode(false, false); got != 1 {
		t.Fatalf("ordinary dry-run failure exit = %d, want 1", got)
	}

	operationErrors = []error{errors.New("no such table: products")}
	if got := failureExitCode(false, false); got != 2 {
		t.Fatalf("missing schema dry-run exit = %d, want 2", got)
	}
	if got := failureExitCode(false, true); got != 1 {
		t.Fatalf("migrate dry-run failure exit = %d, want 1", got)
	}
	operationErrors = nil
}

func TestUpdateFailureIsCollectedAndReported(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "failure.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&productdomain.Product{}, &productdomain.ProductSKU{}); err != nil {
		t.Fatalf("migrate product tables: %v", err)
	}
	if err := db.Create(&productdomain.Product{
		Slug:        "dupe-test",
		TitleJSON:   jsonmap.JSON{"zh-CN": "重复正文"},
		ContentJSON: jsonmap.JSON{"zh-CN": `<h2>核心亮点</h2><p>A</p><h2>核心亮点</h2><p>B</p>`},
		IsActive:    true,
	}).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER fail_product_update BEFORE UPDATE ON products BEGIN SELECT RAISE(ABORT, 'injected update failure'); END;`).Error; err != nil {
		t.Fatalf("create trigger: %v", err)
	}

	operationErrors = nil
	changes := dedupeProducts(db, true)
	if len(changes) == 0 {
		t.Fatal("expected a planned change even when update fails")
	}
	if len(operationErrors) == 0 {
		t.Fatal("update failure was not collected")
	}
	if got := failureExitCode(true, false); got != 1 {
		t.Fatalf("failure exit = %d, want 1", got)
	}

	report := filepath.Join(t.TempDir(), "dedupe-failure.csv")
	if err := writeReport(report, changes); err != nil {
		t.Fatalf("write report: %v", err)
	}
	body, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "dupe-test") {
		t.Fatalf("report missing failed product: %s", body)
	}
	operationErrors = nil
}
