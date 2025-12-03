package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestNormalizeURL проверяет нормализацию URL
func TestNormalizeURL(t *testing.T) {
	tests := map[string]string{
		"http://example.com/#section": "http://example.com/",
		"https://example.com":         "https://example.com",
		"  http://test.com/path ":     "http://test.com/path",
		"":                            "",
	}
	for in, want := range tests {
		got := normalizeURL(in)
		if got != want {
			t.Errorf("normalizeURL(%q) = %q; want %q", in, got, want)
		}
	}
}

// TestURLToLocalPath проверяет создание локального пути
func TestURLToLocalPath(t *testing.T) {
	baseDir := "./downloaded"
	tests := []struct {
		urlStr  string
		resType string
		wantEnd string
	}{
		{"http://example.com", "html", "example.com/index.html"},
		{"https://example.com/path/", "html", "example.com/path/index.html"},
		{"https://example.com/file.js", "js", "example.com/file.js"},
		{"https://example.com/dir", "html", "example.com/dir.html"},
	}

	for _, tt := range tests {
		got, err := urlToLocalPath(tt.urlStr, baseDir, tt.resType)
		if err != nil {
			t.Errorf("urlToLocalPath(%q) returned error: %v", tt.urlStr, err)
		}
		gotSlash := filepath.ToSlash(got)
		if !strings.HasSuffix(gotSlash, tt.wantEnd) {
			t.Errorf("urlToLocalPath(%q) = %q; want suffix %q", tt.urlStr, gotSlash, tt.wantEnd)
		}
	}
}

// TestGetResourceType проверяет определение типа ресурса
func TestGetResourceType(t *testing.T) {
	tests := map[string]string{
		"http://example.com":              "html",
		"http://example.com/index.html":   "html",
		"https://site.com/style.css":      "css",
		"https://site.com/app.js":         "js",
		"https://site.com/image.png":      "image",
		"https://site.com/folder/":        "html",
		"https://site.com/download":       "html",
		"https://site.com/archive.tar.gz": "resource",
	}
	for urlStr, want := range tests {
		got := getResourceType(urlStr)
		if got != want {
			t.Errorf("getResourceType(%q) = %q; want %q", urlStr, got, want)
		}
	}
}

// TestUniqueStrings проверяет удаление дубликатов
func TestUniqueStrings(t *testing.T) {
	input := []string{"a", "b", "a", "", "c", "b"}
	want := []string{"a", "b", "c"}
	got := uniqueStrings(input)
	if len(got) != len(want) {
		t.Errorf("uniqueStrings length = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uniqueStrings[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

// TestFilepathToSlash проверяет замену обратных слэшей
func TestFilepathToSlash(t *testing.T) {
	input := "folder\\sub\\file.html"
	want := "folder/sub/file.html"
	got := filepathToSlash(input)
	if got != want {
		t.Errorf("filepathToSlash(%q) = %q; want %q", input, got, want)
	}
}

// TestURLSet проверяет добавление и уникальность
func TestURLSet(t *testing.T) {
	s := NewURLSet()
	if !s.Add("http://example.com") {
		t.Error("expected Add to return true for new URL")
	}
	if s.Add("http://example.com") {
		t.Error("expected Add to return false for duplicate URL")
	}
	if s.Size() != 1 {
		t.Errorf("Size = %d; want 1", s.Size())
	}
}
