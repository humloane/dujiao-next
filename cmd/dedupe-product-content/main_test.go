package main

import (
	"strings"
	"testing"
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
