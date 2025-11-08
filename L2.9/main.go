package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"
)

// unpackString выполняет "распаковку" строки с поддержкой экранирования
func unpackString(s string) (string, error) {
	if s == "" { // проверка на пустую строку
		return "", nil
	}

	var result strings.Builder
	var prev rune
	var escaped bool

	for i, r := range s {
		if escaped { // обработка случая с экранированием
			result.WriteRune(r)
			prev = r
			escaped = false
			continue
		}

		if r == '\\' {
			if i == len(s)-1 { // проверка на последний символ (не / в конце строки)
				return "", errors.New("input error: string ends with escape symbol")
			}
			escaped = true // экранируем
			continue
		}

		if unicode.IsDigit(r) { // проверка на цифру
			if prev == 0 { // если строка начинается с цифры
				return "", errors.New("input error: string starts with digit")
			}
			count := int(r - '0')
			for j := 1; j < count; j++ { // запись символа столько раз сколько указано в цифре перед ним
				result.WriteRune(prev)
			}
		} else {
			result.WriteRune(r)
			prev = r
		}
	}

	return result.String(), nil
}

func main() {
	var inputString string
	fmt.Scan(&inputString)

	log.SetOutput(os.Stderr)
	log.SetFlags(0)

	result, err := unpackString(inputString)
	if err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}
	fmt.Println(result)
}
