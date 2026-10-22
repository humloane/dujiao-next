package application

import (
	"strings"
	"testing"
)

func TestNormalizeUTMKeepsOnlyWhitelistedKeys(t *testing.T) {
	got := normalizeUTM(map[string]string{
		"source":   " google ",
		"medium":   "cpc",
		"campaign": strings.Repeat("x", 250),
		"evil":     "drop-me",
		"term":     "",
	})
	if got["source"] != "google" || got["medium"] != "cpc" {
		t.Fatalf("unexpected utm: %#v", got)
	}
	if _, ok := got["evil"]; ok {
		t.Fatalf("unknown key must be dropped: %#v", got)
	}
	if _, ok := got["term"]; ok {
		t.Fatalf("empty value must be dropped: %#v", got)
	}
	if len([]rune(got["campaign"].(string))) != utmValueMaxLength {
		t.Fatalf("campaign length = %d, want %d", len([]rune(got["campaign"].(string))), utmValueMaxLength)
	}
}

func TestNormalizeUTMEmptyInput(t *testing.T) {
	if got := normalizeUTM(nil); len(got) != 0 {
		t.Fatalf("nil input should produce empty map, got %#v", got)
	}
}
