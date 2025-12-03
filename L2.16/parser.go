package main

import (
	"bytes"
	"net/url"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// ExtractAndRewriteLinks - Парсит HTML, находит все ссылки (href, src, srcset, link rel=stylesheet)
// Возвращает slice абсолютных URL для очереди и HTML с переписанными локальными ссылками.
func ExtractAndRewriteLinks(htmlBytes []byte, pageURL, outputDir string) ([]string, []byte, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes)) // парсинг HTML в DOM
	if err != nil {
		return nil, nil, err
	}

	base, err := url.Parse(pageURL) // базовый URL страницы для абсолютных ссылок
	if err != nil {
		return nil, nil, err
	}

	var found []string // все найденные ссылки для очереди

	// локальный путь текущей страницы
	pageLocal, err := urlToLocalPath(pageURL, outputDir, "html")
	if err != nil {
		return nil, nil, err
	}
	pageDir := filepath.Dir(pageLocal)

	// walk - рекурсивный обход DOM дерева
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				for i, a := range n.Attr {
					if a.Key == "href" {
						abs := resolveRef(base, a.Val)
						if abs != "" {
							found = append(found, abs)
							local, err := urlToLocalPath(abs, outputDir, getResourceType(abs))
							if err == nil {
								rel, err := filepath.Rel(pageDir, local)
								if err == nil {
									n.Attr[i].Val = filepathToSlash(rel)
								}
							}
						}
					}
				}
			case "img", "script", "iframe", "embed", "source":
				for i, a := range n.Attr {
					if a.Key == "src" {
						abs := resolveRef(base, a.Val)
						if abs != "" {
							found = append(found, abs)
							local, err := urlToLocalPath(abs, outputDir, getResourceType(abs))
							if err == nil {
								rel, err := filepath.Rel(pageDir, local)
								if err == nil {
									n.Attr[i].Val = filepathToSlash(rel)
								}
							}
						}
					}
					if a.Key == "srcset" {
						parts := strings.Split(a.Val, ",")
						var outParts []string
						for _, p := range parts {
							p = strings.TrimSpace(p)
							if p == "" {
								continue
							}
							parts2 := strings.Fields(p)
							u0 := parts2[0]
							abs := resolveRef(base, u0)
							if abs == "" {
								outParts = append(outParts, p)
								continue
							}
							found = append(found, abs)
							local, err := urlToLocalPath(abs, outputDir, getResourceType(abs))
							if err != nil {
								outParts = append(outParts, p)
								continue
							}
							rel, err := filepath.Rel(pageDir, local)
							if err != nil {
								outParts = append(outParts, p)
								continue
							}
							if len(parts2) > 1 {
								outParts = append(outParts, filepathToSlash(rel)+" "+strings.Join(parts2[1:], " "))
							} else {
								outParts = append(outParts, filepathToSlash(rel))
							}
						}
						n.Attr[i].Val = strings.Join(outParts, ", ")
					}
				}
			case "link":
				hrefIdx := -1
				var relVal string
				for i, a := range n.Attr {
					if a.Key == "href" {
						hrefIdx = i
					}
					if a.Key == "rel" {
						relVal = strings.ToLower(a.Val)
					}
				}
				if hrefIdx >= 0 {
					abs := resolveRef(base, n.Attr[hrefIdx].Val)
					if abs != "" {
						found = append(found, abs)
						local, err := urlToLocalPath(abs, outputDir, getResourceType(abs))
						if err == nil {
							rel, err := filepath.Rel(pageDir, local)
							if err == nil {
								n.Attr[hrefIdx].Val = filepathToSlash(rel)
							}
						}
					}
				}
				_ = relVal
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return nil, nil, err
	}
	return uniqueStrings(found), buf.Bytes(), nil
}

// resolveRef - Преобразует ссылку на странице в абсолютный URL, игнорирует фрагменты и mailto:, javascript: и tel:
func resolveRef(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "#") || strings.HasPrefix(strings.ToLower(ref), "mailto:") ||
		strings.HasPrefix(strings.ToLower(ref), "tel:") || strings.HasPrefix(strings.ToLower(ref), "javascript:") {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	res := base.ResolveReference(u)
	res.Fragment = ""
	return res.String()
}

// filepathToSlash - заменяет \ на / для совместимости с URL
func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// uniqueStrings - убирает дубликаты строк
func uniqueStrings(in []string) []string {
	m := make(map[string]struct{}, len(in))
	var out []string
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := m[s]; ok {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
