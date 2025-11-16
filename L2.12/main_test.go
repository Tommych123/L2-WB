package main

import (
	"fmt"
	"strings"
	"testing"
)

var testLines = []string{
	"Hello World",
	"hello world",
	"HELLO WORLD",
	"Goodbye World",
	"Something else",
	"Another line",
	"Match this line",
	"match This Line",
	"MATCH THIS LINE",
	"End of file",
}

// helper для получения совпадений по matcher
func getMatches(lines []string, matcher func(string) bool, invert bool) []string {
	matchIdx := findMatches(lines, matcher, invert)
	result := []string{}
	for i := 0; i < len(lines); i++ {
		if matchIdx[i] {
			result = append(result, lines[i])
		}
	}
	return result
}

// Тест поиска точной строки с игнорированием регистра (-F -i)
func TestFixedIgnoreCase(t *testing.T) {
	matcher := makeMatcher("hello", true, true)
	matches := getMatches(testLines, matcher, false)
	expected := []string{"Hello World", "hello world", "HELLO WORLD"}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d", len(expected), len(matches))
	}
	for i := range expected {
		if matches[i] != expected[i] {
			t.Errorf("expected %q, got %q", expected[i], matches[i])
		}
	}
}

// Тест поиска с регулярным выражением, игнорирование регистра (-i)
func TestRegexCaseInsensitive(t *testing.T) {
	matcher := makeMatcher("Match.*Line", false, true)
	matches := getMatches(testLines, matcher, false)
	expected := []string{"Match this line", "match This Line", "MATCH THIS LINE"}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d", len(expected), len(matches))
	}
	for i := range expected {
		if matches[i] != expected[i] {
			t.Errorf("expected %q, got %q", expected[i], matches[i])
		}
	}
}

// Тест поиска с регекспом, чувствительный к регистру
func TestRegexCaseSensitive(t *testing.T) {
	matcher := makeMatcher("Match this line", false, false)
	matches := getMatches(testLines, matcher, false)
	expected := []string{"Match this line"}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d", len(expected), len(matches))
	}
	if matches[0] != expected[0] {
		t.Errorf("expected %q, got %q", expected[0], matches[0])
	}
}

// Тест инверсии совпадений (-v)
func TestInvertMatch(t *testing.T) {
	matcher := makeMatcher("hello", true, true)
	matches := getMatches(testLines, matcher, true)
	for _, m := range matches {
		if strings.Contains(strings.ToLower(m), "hello") {
			t.Errorf("invert match failed, found %q", m)
		}
	}
}

// Тест подсчета совпадений (-c)
func TestCountMatches(t *testing.T) {
	matcher := makeMatcher("match.*line", false, true)
	matchIdx := findMatches(testLines, matcher, false)
	if len(matchIdx) != 3 {
		t.Errorf("expected 3 matches, got %d", len(matchIdx))
	}
}

// Тест контекста (-A, -B, -C)
func TestPrintMatchesContext(t *testing.T) {
	lines := testLines
	matchIdx := findMatches(lines, makeMatcher("match.*line", false, true), false)
	printed := make(map[int]bool)
	result := []string{}
	for i := 0; i < len(lines); i++ {
		if !matchIdx[i] {
			continue
		}
		start := i - 1
		if start < 0 {
			start = 0
		}
		end := i + 1
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			if printed[j] {
				continue
			}
			printed[j] = true
			result = append(result, lines[j])
		}
	}
	expectedCount := 5
	if len(result) != expectedCount {
		t.Errorf("context test failed: expected %d lines, got %d", expectedCount, len(result))
	}
}

// Пустой файл
func TestEmptyFile(t *testing.T) {
	lines := []string{}
	matcher := makeMatcher("something", true, true)
	matchIdx := findMatches(lines, matcher, false)
	if len(matchIdx) != 0 {
		t.Errorf("expected 0 matches for empty file")
	}
}

// Совпадение в начале файла
func TestMatchAtStart(t *testing.T) {
	lines := []string{"match first line", "second line"}
	matcher := makeMatcher("match", true, true)
	matchIdx := findMatches(lines, matcher, false)
	if len(matchIdx) != 1 || !matchIdx[0] {
		t.Errorf("expected match at line 0")
	}
}

// Совпадение в конце файла
func TestMatchAtEnd(t *testing.T) {
	lines := []string{"first line", "last match"}
	matcher := makeMatcher("match", true, true)
	matchIdx := findMatches(lines, matcher, false)
	if len(matchIdx) != 1 || !matchIdx[1] {
		t.Errorf("expected match at last line")
	}
}

// Перекрывающиеся контексты (-A, -B)
func TestOverlappingContext(t *testing.T) {
	lines := []string{"a", "b", "match", "c", "match", "d", "e"}
	matcher := makeMatcher("match", true, true)
	matchIdx := findMatches(lines, matcher, false)

	printed := make(map[int]bool)
	result := []string{}
	for i := 0; i < len(lines); i++ {
		if !matchIdx[i] {
			continue
		}
		start := i - 1
		if start < 0 {
			start = 0
		}
		end := i + 1
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			if printed[j] {
				continue
			}
			printed[j] = true
			result = append(result, lines[j])
		}
	}
	expected := []string{"b", "match", "c", "match", "d"}
	if len(result) != len(expected) {
		t.Errorf("overlapping context failed: expected %v, got %v", expected, result)
	}
}

// Номера строк (-n)
func TestShowLineNumbers(t *testing.T) {
	lines := []string{"first match", "second match"}
	matcher := makeMatcher("match", true, true)
	matchIdx := findMatches(lines, matcher, false)

	printed := make(map[int]bool)
	result := []string{}
	for i := 0; i < len(lines); i++ {
		if !matchIdx[i] {
			continue
		}
		start := i
		end := i
		for j := start; j <= end; j++ {
			if printed[j] {
				continue
			}
			printed[j] = true
			result = append(result, fmt.Sprintf("%d:%s", j+1, lines[j]))
		}
	}
	expected := []string{"1:first match", "2:second match"}
	if len(result) != len(expected) {
		t.Errorf("line numbers test failed")
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("expected %q, got %q", expected[i], result[i])
		}
	}
}

// Проверка комбинации флагов (-v и -A)
func TestCombinedFlags(t *testing.T) {
	lines := []string{"Hello", "hello", "world", "HELLO", "other"}
	matcher := makeMatcher("hello", true, true)
	matchIdx := findMatches(lines, matcher, true) // -v
	printed := make(map[int]bool)
	result := []string{}
	for i := 0; i < len(lines); i++ {
		if !matchIdx[i] {
			continue
		}
		start := i
		end := i + 1 // -A1
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			if printed[j] {
				continue
			}
			printed[j] = true
			result = append(result, fmt.Sprintf("%d:%s", j+1, lines[j]))
		}
	}
	expected := []string{"3:world", "4:HELLO", "5:other"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d lines, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("expected %q, got %q", expected[i], result[i])
		}
	}
}
