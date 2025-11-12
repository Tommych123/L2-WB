package main

import (
	"fmt"
	"sort"
	"strings"
)

// findAnagrams - находит и формирует map-множество анаграмм
func findAnagrams(words []string) map[string][]string {
	dictionary := make(map[string][]string)       // промежуточный неотсортированный словарь
	resultDictionary := make(map[string][]string) // итоговый словарь
	for _, v := range words {
		v = strings.ToLower(v)                                                            // приводим к нижнему регистру слово
		inputWord := []rune(v)                                                            // переведом string -> слайс rune
		sort.Slice(inputWord, func(i, j int) bool { return inputWord[i] < inputWord[j] }) // сортируем слайс rune
		sorted := string(inputWord)                                                       // переводим слайс rune -> string
		dictionary[sorted] = append(dictionary[sorted], v)                                // добавляем в слайс по ключу анаграмму
	}
	for key := range dictionary {
		if len(dictionary[key]) > 1 { // отбираем слова у которых есть анаграммы
			realKey := dictionary[key][0]               // берем первое слово встреченное как ключ для итоговой map
			sort.Strings(dictionary[key])               // сортируем слова в массиве анаграмм по возрастанию
			resultDictionary[realKey] = dictionary[key] // заносим по realKey сортированный массив
		}
	}
	return resultDictionary
}

func main() {
	words := []string{"Пятак", "пяТка", "тяпка", "слиток", "листок", "столик", "стол"}
	resultMap := findAnagrams(words)
	for key := range resultMap {
		fmt.Println(key, resultMap[key]) // вывод
	}
}
