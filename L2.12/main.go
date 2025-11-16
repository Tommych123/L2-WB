package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// crash выводит сообщение об ошибке и завершает программу
func crash(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// Flags хранит значения флагов и информации о файле или шаблоне
type Flags struct {
	After   int    // количество строк после совпадения (-A)
	Before  int    // количество строк до совпадения (-B)
	Count   bool   // выводить только количество совпадений (-c)
	Ignore  bool   // игнорировать регистр (-i)
	Invert  bool   // инвертировать совпадение (-v)
	Fixed   bool   // поиск точной строки (-F)
	Num     bool   // показывать номера строк (-n)
	Pattern string // шаблон поиска
	File    string // входной файл (пусто = stdin)
}

// parseFlags парсит флаги командной строки и возвращает структуру Flags
func parseFlags() Flags {
	after := flag.Int("A", 0, "количество строк после совпадения")
	before := flag.Int("B", 0, "количество строк до совпадения")
	context := flag.Int("C", 0, "контекст до и после")
	count := flag.Bool("c", false, "выводить только количество совпадений")
	ignore := flag.Bool("i", false, "игнорировать регистр")
	invert := flag.Bool("v", false, "инвертировать совпадение")
	fixed := flag.Bool("F", false, "поиск точной строки")
	num := flag.Bool("n", false, "показывать номера строк")

	flag.Parse()

	if *context > 0 {
		*after = *context
		*before = *context
	}

	if flag.NArg() < 1 {
		crash(errors.New("usage: grep [flags] pattern [file]"))
	}

	pattern := flag.Arg(0)
	file := ""
	if flag.NArg() > 1 {
		file = flag.Arg(1)
	}

	return Flags{
		After:   *after,
		Before:  *before,
		Count:   *count,
		Ignore:  *ignore,
		Invert:  *invert,
		Fixed:   *fixed,
		Num:     *num,
		Pattern: pattern,
		File:    file,
	}
}

// readLines считывает все строки из файла или stdin и возвращает их срезом
func readLines(filename string) ([]string, error) {
	var reader io.Reader
	if filename != "" {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	scanner := bufio.NewScanner(reader)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// makeMatcher возвращает функцию которая проверяет соответствует ли строка шаблону
func makeMatcher(pattern string, fixed, ignore bool) func(string) bool {
	var re *regexp.Regexp
	var err error

	if !fixed {
		if ignore {
			pattern = "(?i)" + pattern
		}
		re, err = regexp.Compile(pattern)
		if err != nil {
			crash(err)
		}
	} else if ignore {
		pattern = strings.ToLower(pattern)
	}

	return func(s string) bool {
		if fixed {
			if ignore {
				s = strings.ToLower(s)
			}
			return strings.Contains(s, pattern)
		}
		return re.MatchString(s)
	}
}

// findMatches возвращает карту индексов строк которые соответствуют шаблону
func findMatches(lines []string, matcher func(string) bool, invert bool) map[int]bool {
	matchIdx := make(map[int]bool)
	for i, line := range lines {
		m := matcher(line)
		if invert {
			m = !m
		}
		if m {
			matchIdx[i] = true
		}
	}
	return matchIdx
}

// printMatches выводит совпавшие строки с контекстом и номерами строк при необходимости
func printMatches(lines []string, matchIdx map[int]bool, before, after int, showNum bool) {
	printed := make(map[int]bool)
	for i := 0; i < len(lines); i++ {
		if !matchIdx[i] {
			continue
		}
		start := i - before
		if start < 0 {
			start = 0
		}
		end := i + after
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			if printed[j] {
				continue
			}
			printed[j] = true
			if showNum {
				fmt.Printf("%d:%s\n", j+1, lines[j])
			} else {
				fmt.Println(lines[j])
			}
		}
	}
}

func main() {
	flags := parseFlags()

	lines, err := readLines(flags.File)
	if err != nil {
		crash(err)
	}

	matcher := makeMatcher(flags.Pattern, flags.Fixed, flags.Ignore)
	matchIdx := findMatches(lines, matcher, flags.Invert)

	if flags.Count {
		fmt.Println(len(matchIdx))
		return
	}

	printMatches(lines, matchIdx, flags.Before, flags.After, flags.Num)
}
