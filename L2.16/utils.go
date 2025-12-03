package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// URLSet - потокобезопасный set для хранения посещённых URL
type URLSet struct {
	mu sync.Mutex
	m  map[string]struct{}
}

// NewURLSet - создаёт новый set
func NewURLSet() *URLSet {
	return &URLSet{m: make(map[string]struct{})}
}

// Add - добавляет URL, возвращает true если URL новый
func (s *URLSet) Add(u string) bool {
	u = normalizeURL(u)
	if u == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.m[u]; ok {
		return false
	}
	s.m[u] = struct{}{}
	return true
}

func (s *URLSet) Size() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.m)
}

// normalizeURL - убирает пробелы и фрагменты (#), приводит URL к нормальной форме
func normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	u.Fragment = ""
	if u.RawQuery == "" {
		u.RawQuery = ""
	}
	return u.String()
}

// ensureDir - создаёт директорию
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// writeFileAtomic - сохраняет файл атомарно: сначала tmp, потом rename
func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := ioutil.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// urlToLocalPath - преобразует URL в локальный путь, учитывает тип ресурса
func urlToLocalPath(rawURL, outputDir, resType string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	p := u.Path
	if p == "" || p == "/" {
		if resType == "html" {
			p = "/index.html"
		} else {
			p = "/resource"
		}
	} else {
		if strings.HasSuffix(p, "/") {
			if resType == "html" {
				p = p + "index.html"
			} else {
				p = p + "resource"
			}
		} else {
			if filepath.Ext(p) == "" && resType == "html" {
				p = p + ".html"
			}
		}
	}
	full := filepath.Join(outputDir, u.Host, filepath.FromSlash(p))
	return filepath.Clean(full), nil
}

// getResourceType - определяет тип ресурса по расширению URL
func getResourceType(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "resource"
	}
	ext := strings.ToLower(filepath.Ext(u.Path))
	switch ext {
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".js":
		return "js"
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".webp", ".bmp":
		return "image"
	default:
		if u.Path == "" || strings.HasSuffix(u.Path, "/") || ext == "" {
			return "html"
		}
		return "resource"
	}
}

// isHTMLByType - определяет HTML по Content-Type и по телу документа
func isHTMLByType(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml+xml") {
		return true
	}
	head := string(body)
	if len(head) > 4096 {
		head = head[:4096]
	}
	head = strings.ToLower(head)
	return strings.Contains(head, "<!doctype html") || strings.Contains(head, "<html")
}

// readAllLimit - читает не больше limit байт из io.Reader
func readAllLimit(r io.Reader, limit int64) ([]byte, error) {
	var buf bytes.Buffer
	n, err := buf.ReadFrom(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if n > limit {
		return nil, errors.New("resource too large")
	}
	return buf.Bytes(), nil
}
