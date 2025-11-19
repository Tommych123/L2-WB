package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// crash выводит сообщение об ошибке и завершает программу
func crash(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// Flags хранит значения флагов
type Flags struct {
	Fields    string // строка -f: "1,3-5"
	Delimiter rune   // разделитель
	Separated bool   // флаг -s
	File      string // входной файл
}

// parseFlags парсит флаги командной строки
func parseFlags() Flags {
	fields := flag.String("f", "", "номера колонок, которые нужно вывести, например 1,3-5 (обязательный)")
	delimiter := flag.String("d", "\t", "разделитель")
	separated := flag.Bool("s", false, "только строки содержащие разделитель")

	flag.Parse()

	if *fields == "" {
		crash(errors.New("usage: cut -f list [-s] [-d delim] [file ...]"))
	}

	runes := []rune(*delimiter)
	if len(runes) != 1 {
		crash(errors.New("delimiter must be a single character"))
	}

	file := ""
	if flag.NArg() > 0 {
		file = flag.Arg(0)
	}

	return Flags{
		Fields:    *fields,
		Delimiter: runes[0],
		Separated: *separated,
		File:      file,
	}
}

// parseFields преобразует "-f 1,3-5" в map[int]bool
func parseFields(spec string) (map[int]bool, error) {
	result := make(map[int]bool)

	parts := strings.Split(spec, ",")
	for _, part := range parts {

		if strings.Contains(part, "-") { // диапазон
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range: %s", part)
			}
			start, err1 := strconv.Atoi(bounds[0])
			end, err2 := strconv.Atoi(bounds[1])

			if err1 != nil || err2 != nil || start <= 0 || end <= 0 || start > end {
				return nil, fmt.Errorf("invalid range: %s", part)
			}

			for i := start; i <= end; i++ {
				result[i] = true
			}
		} else { // одиночное поле
			n, err := strconv.Atoi(part)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid field: %s", part)
			}
			result[n] = true
		}
	}
	return result, nil
}

// processLine обрабатывает одну строку
func processLine(line string, flags Flags, fields map[int]bool) {
	if flags.Separated && !strings.ContainsRune(line, flags.Delimiter) { // если -s и разделителя нет то игнорируем
		return
	}

	cols := strings.Split(line, string(flags.Delimiter))

	var out []string

	for i := 1; i <= len(cols); i++ {
		if fields[i] {
			out = append(out, cols[i-1])
		}
	}

	fmt.Println(strings.Join(out, string(flags.Delimiter))) // если нет подходящих полей — выводим пустую строку
}

// processFile определение ввода из файла или из stdin и считывание данных
func processFile(filename string, flags Flags, fields map[int]bool) error {
	var reader io.Reader
	if filename != "" {
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		processLine(scanner.Text(), flags, fields)
	}
	return scanner.Err()
}

func main() {
	flags := parseFlags()

	fields, err := parseFields(flags.Fields)
	if err != nil {
		crash(err)
	}

	if err := processFile(flags.File, flags, fields); err != nil {
		crash(err)
	}
}
