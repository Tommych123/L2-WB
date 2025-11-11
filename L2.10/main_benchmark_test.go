package main

import (
	"strconv"
	"testing"
)

// Benchmark для больших файлов с разными флагами
func BenchmarkSortLargeFileNumeric(b *testing.B) {
	lines := make([]string, 100000)
	for i := 0; i < len(lines); i++ {
		lines[i] = strconv.Itoa(len(lines)-i) + "\t" + strconv.Itoa(len(lines)-i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortLines(lines, 1, true, false, false, false, false)
	}
}

func BenchmarkSortLargeFileReverse(b *testing.B) {
	lines := make([]string, 100000)
	for i := 0; i < len(lines); i++ {
		lines[i] = strconv.Itoa(i) + "\t" + strconv.Itoa(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortLines(lines, 1, true, true, false, false, false)
	}
}

func BenchmarkSortLargeFileHuman(b *testing.B) {
	lines := []string{"1K", "2M", "500", "3G", "4K", "5M"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortLines(lines, 0, false, false, false, false, true)
	}
}

func BenchmarkMergeSortedChunks(b *testing.B) {
	chunks := [][]string{
		{"1", "3", "5"},
		{"2", "4", "6"},
		{"0", "7", "8"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mergeSortedChunks(chunks, 0, true, false, false, false, false)
	}
}

func BenchmarkMonthSort(b *testing.B) {
	lines := []string{"Mar", "Jan", "Feb", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sortLines(lines, 0, false, false, true, false, false)
	}
}
