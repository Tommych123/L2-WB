package main

import (
	"strconv"
	"testing"
)

// Полный интеграционный тест со всеми флагами
func TestIntegrationAllFlags(t *testing.T) {
	lines := []string{
		"Mar\t2K",
		"Jan\t500",
		"Feb\t1M",
		"Feb\t500",
		"Mar\t1K",
		"Jan\t2M",
		"Apr\t1K  ",
	}

	tests := []struct {
		name                                                   string
		column                                                 int
		numeric, reverse, unique, month, ignoreTrailing, human bool
		want                                                   []string
	}{
		{
			name:           "human readable sort col2",
			column:         2,
			numeric:        false,
			reverse:        false,
			unique:         false,
			month:          false,
			ignoreTrailing: false,
			human:          true,
			want:           []string{"Jan\t500", "Feb\t500", "Mar\t1K", "Apr\t1K  ", "Mar\t2K", "Feb\t1M", "Jan\t2M"},
		},
		{
			name:           "month sort col1",
			column:         1,
			numeric:        false,
			reverse:        false,
			unique:         false,
			month:          true,
			ignoreTrailing: false,
			human:          false,
			want:           []string{"Jan\t500", "Jan\t2M", "Feb\t1M", "Feb\t500", "Mar\t2K", "Mar\t1K", "Apr\t1K  "},
		},
		{
			name:           "ignore trailing blanks",
			column:         2,
			numeric:        false,
			reverse:        false,
			unique:         false,
			month:          false,
			ignoreTrailing: true,
			human:          true,
			want:           []string{"Jan\t500", "Feb\t500", "Mar\t1K", "Apr\t1K  ", "Mar\t2K", "Feb\t1M", "Jan\t2M"},
		},
		{
			name:           "reverse numeric col2 unique",
			column:         2,
			numeric:        false,
			reverse:        true,
			unique:         true,
			month:          false,
			ignoreTrailing: false,
			human:          true,
			want:           []string{"Jan\t2M", "Feb\t1M", "Mar\t2K", "Mar\t1K", "Apr\t1K  ", "Jan\t500", "Feb\t500"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sorted := sortLines(lines, tt.column, tt.numeric, tt.reverse, tt.month, tt.ignoreTrailing, tt.human)
			if tt.unique {
				sorted = removeDuplicates(sorted)
			}
			if len(sorted) != len(tt.want) {
				t.Fatalf("length mismatch: got %d, want %d", len(sorted), len(tt.want))
			}
			for i := range tt.want {
				if sorted[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, sorted[i], tt.want[i])
				}
			}
		})
	}
}

// Тест с большим количеством строк для проверки производительности и корректности
func TestIntegrationLargeFileAllFlags(t *testing.T) {
	lines := make([]string, 100000)
	for i := 0; i < 100000; i++ {
		lines[i] = strconv.Itoa(100000-i) + "\t" + strconv.Itoa(100000-i)
	}

	sorted := sortLines(lines, 1, true, false, false, false, false)
	if sorted[0] != "1\t1" {
		t.Errorf("large file sort failed, first element %q", sorted[0])
	}
	if sorted[len(sorted)-1] != "100000\t100000" {
		t.Errorf("large file sort failed, last element %q", sorted[len(sorted)-1])
	}
}

// Тест функции extractColumn
func TestExtractColumn(t *testing.T) {
	tests := []struct {
		line   string
		column int
		want   string
	}{
		{"a\tb\tc", 0, "a\tb\tc"},
		{"a\tb\tc", 1, "a"},
		{"a\tb\tc", 2, "b"},
		{"a\tb\tc", 3, "c"},
		{"a\tb\tc", 4, ""},
		{"single", 1, "single"},
	}

	for _, tt := range tests {
		got := extractColumn(tt.line, tt.column)
		if got != tt.want {
			t.Errorf("extractColumn(%q, %d) = %q, want %q", tt.line, tt.column, got, tt.want)
		}
	}
}

// Тест функции compareValues
func TestCompareValues(t *testing.T) {
	tests := []struct {
		a, b           string
		column         int
		numeric        bool
		reverse        bool
		month          bool
		ignoreTrailing bool
		human          bool
		want           bool
	}{
		{"andrey", "bob", 0, false, false, false, false, false, true},
		{"bob", "andrey", 0, false, false, false, false, false, false},

		{"13", "5", 0, true, false, false, false, false, false},
		{"5", "13", 0, true, false, false, false, false, true},

		{"apple", "banana", 0, false, true, false, false, false, false},
		{"banana", "apple", 0, false, true, false, false, false, true},

		{"Jan", "Feb", 0, false, false, true, false, false, true},
		{"Feb", "Jan", 0, false, false, true, false, false, false},

		{"1K", "500", 0, false, false, false, false, true, false},
		{"500", "1K", 0, false, false, false, false, true, true},
		{"1M", "1K", 0, false, false, false, false, true, false},
	}

	for _, tt := range tests {
		got := compareValues(tt.a, tt.b, tt.column, tt.numeric, tt.reverse, tt.month, tt.ignoreTrailing, tt.human)
		if got != tt.want {
			t.Errorf("compareValues(%q, %q, col=%d, num=%v, rev=%v, month=%v, ignoreTrailing=%v, human=%v) = %v, want %v",
				tt.a, tt.b, tt.column, tt.numeric, tt.reverse, tt.month, tt.ignoreTrailing, tt.human, got, tt.want)
		}
	}
}

// Тест функции sortLines
func TestSortLines(t *testing.T) {
	lines := []string{"c", "a", "b"}
	sorted := sortLines(lines, 0, false, false, false, false, false)
	expected := []string{"a", "b", "c"}

	for i := range expected {
		if sorted[i] != expected[i] {
			t.Errorf("sortLines: got %v, want %v", sorted, expected)
			break
		}
	}
}

// Тест функции removeDuplicates
func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{[]string{"a", "a", "b", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a"}, []string{"a"}},
		{[]string{}, []string{}},
	}

	for _, tt := range tests {
		got := removeDuplicates(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("removeDuplicates(%v) length: got %d, want %d", tt.input, len(got), len(tt.want))
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("removeDuplicates(%v): got %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

// Тест функции mergeSortedChunks
func TestMergeSortedChunks(t *testing.T) {
	chunks := [][]string{
		{"a", "c", "e"},
		{"b", "d", "f"},
	}
	merged := mergeSortedChunks(chunks, 0, false, false, false, false, false)
	expected := []string{"a", "b", "c", "d", "e", "f"}

	for i := range expected {
		if merged[i] != expected[i] {
			t.Errorf("mergeSortedChunks: got %v, want %v", merged, expected)
			break
		}
	}
}

// Тест сортировки по месяцам
func TestMonthSort(t *testing.T) {
	lines := []string{"Mar", "Jan", "Feb", "Dec"}
	sorted := sortLines(lines, 0, false, false, true, false, false)
	expected := []string{"Jan", "Feb", "Mar", "Dec"}

	for i := range expected {
		if sorted[i] != expected[i] {
			t.Errorf("MonthSort: got %v, want %v", sorted, expected)
			break
		}
	}
}

// Тест игнорирования хвостовых пробелов
func TestIgnoreTrailing(t *testing.T) {
	lines := []string{"a  ", "a", "b  "}
	sorted := sortLines(lines, 0, false, false, false, true, false)
	if sorted[0] != "a  " || sorted[1] != "a" {
		t.Errorf("IgnoreTrailing sort: got %v, want a and a together", sorted)
	}
}

// Тест человекочитаемых чисел
func TestHumanSort(t *testing.T) {
	lines := []string{"1K", "500", "2M", "1.5K"}
	sorted := sortLines(lines, 0, false, false, false, false, true)

	expected := []string{"500", "1K", "1.5K", "2M"}

	for i := range expected {
		if sorted[i] != expected[i] {
			t.Errorf("HumanSort: got %v, want %v", sorted, expected)
			break
		}
	}
}

// Тест проверки сортированности
func TestCheckSorted(t *testing.T) {
	// Этот тест проверяет, что compareValues работает корректно для проверки сортированности
	sortedLines := []string{"a", "b", "c"}
	unsortedLines := []string{"b", "a", "c"}

	// Проверяем отсортированный список
	for i := 1; i < len(sortedLines); i++ {
		if !compareValues(sortedLines[i-1], sortedLines[i], 0, false, false, false, false, false) {
			t.Error("CheckSorted: sorted lines should be considered sorted")
		}
	}

	// Проверяем неотсортированный список
	isSorted := true
	for i := 1; i < len(unsortedLines); i++ {
		if !compareValues(unsortedLines[i-1], unsortedLines[i], 0, false, false, false, false, false) {
			isSorted = false
			break
		}
	}
	if isSorted {
		t.Error("CheckSorted: unsorted lines should not be considered sorted")
	}
}
