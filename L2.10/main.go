// Программа sort — упрощённый аналог утилиты UNIX `sort`.
// Поддерживает флаги: -k, -n, -r, -u, -M, -b, -c, -h
// Эффективно обрабатывает большие файлы путём сортировки чанков и их слияния.
package main

import (
	"bufio"
	"container/heap"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

const chunkSize = 100000 // количество строк в одном чанке перед сортировкой и слиянием

var monthMap = map[string]int{
	"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4, "May": 5, "Jun": 6,
	"Jul": 7, "Aug": 8, "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12,
} // map для флага M

func main() {
	// Флаги
	column := pflag.IntP("column", "k", 0, "сортировать по колонке (1-based, разделитель табуляция)")
	numeric := pflag.BoolP("numeric", "n", false, "числовая сортировка")
	reverse := pflag.BoolP("reverse", "r", false, "обратный порядок")
	unique := pflag.BoolP("unique", "u", false, "только уникальные строки")
	month := pflag.BoolP("month", "M", false, "сортировка по месяцам")
	ignoreTrailing := pflag.BoolP("ignore-trailing", "b", false, "игнор хвостовых пробелов")
	checkSorted := pflag.BoolP("check", "c", false, "проверка отсортированности")
	human := pflag.BoolP("human", "h", false, "человекочитаемые числа (1K, 2M, 3G)")
	pflag.Parse()

	// Определяем источник ввода
	var scanner *bufio.Scanner
	if len(pflag.Args()) > 0 {
		file, err := os.Open(pflag.Args()[0])
		if err != nil {
			log.Fatalf("не удалось открыть файл: %v", err)
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // увеличение буфера для длинных строк

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("ошибка при чтении: %v", err)
	}

	// Флаг -c
	if *checkSorted {
		for i := 1; i < len(lines); i++ {
			if !compareValues(lines[i-1], lines[i], *column, *numeric, *reverse, *month, *ignoreTrailing, *human) {
				fmt.Fprintf(os.Stderr, "данные не отсортированы на строке %d\n", i+1)
				os.Exit(1)
			}
		}
		return
	}

	// Разделение на чанки и сортировка
	var sortedChunks [][]string
	for i := 0; i < len(lines); i += chunkSize {
		end := i + chunkSize
		if end > len(lines) {
			end = len(lines)
		}
		chunk := sortLines(lines[i:end], *column, *numeric, *reverse, *month, *ignoreTrailing, *human)
		sortedChunks = append(sortedChunks, chunk)
	}

	// Слияние чанков
	var finalLines []string
	switch len(sortedChunks) {
	case 0:
		return
	case 1:
		finalLines = sortedChunks[0]
	default:
		finalLines = mergeSortedChunks(sortedChunks, *column, *numeric, *reverse, *month, *ignoreTrailing, *human)
	}

	// Флаг u
	if *unique && len(finalLines) > 0 {
		finalLines = removeDuplicates(finalLines)
	}
	// Вывод
	for _, line := range finalLines {
		fmt.Println(line)
	}
}

// compareValues сравнивает две строки с учётом всех флагов
func compareValues(a, b string, column int, numeric, reverse, month, ignoreTrailing, human bool) bool {
	keyA, keyB := extractColumn(a, column), extractColumn(b, column)
	//Флаг b
	if ignoreTrailing {
		keyA = strings.TrimRight(keyA, " \t")
		keyB = strings.TrimRight(keyB, " \t")
	}

	// Флаг M
	if month {
		valA, okA := monthMap[keyA]
		valB, okB := monthMap[keyB]
		if okA && okB {
			if reverse {
				return valA > valB
			}
			return valA < valB
		}
		// Если одна из строк не месяц, сравниваем как строки
		if okA != okB {
			if reverse {
				return !okA
			}
			return okA
		}
	}

	// Флаг h
	if human {
		numA, errA := parseHumanSize(keyA)
		numB, errB := parseHumanSize(keyB)
		if errA == nil && errB == nil {
			if reverse {
				return numA > numB
			}
			return numA < numB
		}
		if (errA == nil) != (errB == nil) {
			if reverse {
				return errA != nil
			}
			return errA == nil
		}
	}

	// Флаг n
	if numeric && !human {
		numA, errA := strconv.ParseFloat(keyA, 64)
		numB, errB := strconv.ParseFloat(keyB, 64)
		if errA == nil && errB == nil {
			if reverse {
				return numA > numB
			}
			return numA < numB
		}
		if (errA == nil) != (errB == nil) {
			if reverse {
				return errA != nil
			}
			return errA == nil
		}
	}

	// Флаг r
	if reverse {
		return keyA > keyB
	}
	return keyA < keyB
}

// extractColumn возвращает нужную колонку
func extractColumn(line string, column int) string {
	if column <= 0 {
		return line
	}
	cols := strings.Split(line, "\t")
	if column <= len(cols) {
		return cols[column-1]
	}
	return ""
}

// parseHumanSize преобразует строку с суффиксом в число
func parseHumanSize(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	mult := 1.0
	var numStr string

	// Проверяем суффиксы в обратном порядке
	switch {
	case strings.HasSuffix(s, "G"):
		mult = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(s, "G")
	case strings.HasSuffix(s, "M"):
		mult = 1024 * 1024
		numStr = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "K"):
		mult = 1024
		numStr = strings.TrimSuffix(s, "K")
	default:
		numStr = s
	}

	numStr = strings.TrimSpace(numStr)
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}
	return val * mult, nil
}

// sortLines сортирует срез строк с учётом всех флагов
func sortLines(lines []string, column int, numeric, reverse, month, ignoreTrailing, human bool) []string {
	sorted := make([]string, len(lines))
	copy(sorted, lines)
	sort.SliceStable(sorted, func(i, j int) bool {
		return compareValues(sorted[i], sorted[j], column, numeric, reverse, month, ignoreTrailing, human)
	})
	return sorted
}

// removeDuplicates убирает соседние одинаковые строки
func removeDuplicates(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	out := lines[:1]
	for i := 1; i < len(lines); i++ {
		if lines[i] != lines[i-1] {
			out = append(out, lines[i])
		}
	}
	return out
}

// mergeSortedChunks сливает отсортированные чанки с помощью кучи
func mergeSortedChunks(chunks [][]string, column int, numeric, reverse, month, ignoreTrailing, human bool) []string {
	h := &lineHeap{
		items:          make([]lineItem, 0),
		column:         column,
		numeric:        numeric,
		reverse:        reverse,
		month:          month,
		ignoreTrailing: ignoreTrailing,
		human:          human,
	}
	heap.Init(h)

	for i, ch := range chunks {
		if len(ch) > 0 {
			heap.Push(h, lineItem{value: ch[0], chunk: i, index: 0})
		}
	}

	var result []string
	for h.Len() > 0 {
		item := heap.Pop(h).(lineItem)
		result = append(result, item.value)
		nextIdx := item.index + 1
		if nextIdx < len(chunks[item.chunk]) {
			heap.Push(h, lineItem{value: chunks[item.chunk][nextIdx], chunk: item.chunk, index: nextIdx})
		}
	}
	return result
}

// lineItem — элемент кучи
type lineItem struct {
	value string
	chunk int
	index int
}

// lineHeap — структура кучи
type lineHeap struct {
	items          []lineItem
	column         int
	numeric        bool
	reverse        bool
	month          bool
	ignoreTrailing bool
	human          bool
}

// Методы для работы с кучей

func (h lineHeap) Len() int { return len(h.items) }

func (h lineHeap) Less(i, j int) bool {
	return compareValues(h.items[i].value, h.items[j].value, h.column, h.numeric, h.reverse, h.month, h.ignoreTrailing, h.human)
}

func (h lineHeap) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *lineHeap) Push(x any) { h.items = append(h.items, x.(lineItem)) }

func (h *lineHeap) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[:n-1]
	return item
}
