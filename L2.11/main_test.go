package main

import (
	"reflect"
	"testing"
)

// Тест 1
func TestFindAnagrams_Basic(t *testing.T) {
	input := []string{"Пятак", "пяТка", "тяпка", "листок", "слиток", "столик", "стол"}
	expected := map[string][]string{
		"пятак":  {"пятак", "пятка", "тяпка"},
		"листок": {"листок", "слиток", "столик"},
	}

	got := findAnagrams(input)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Ожидалось %+v, получено %+v", expected, got)
	}
}

// Тест 2: только одно слово
func TestFindAnagrams_SingleWord(t *testing.T) {
	input := []string{"дом"}
	expected := map[string][]string{}

	got := findAnagrams(input)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Ожидалось %+v, получено %+v", expected, got)
	}
}

// Тест 3: слова без анаграмм
func TestFindAnagrams_NoAnagrams(t *testing.T) {
	input := []string{"кот", "собака", "дерево"}
	expected := map[string][]string{}

	got := findAnagrams(input)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Ожидалось %+v, получено %+v", expected, got)
	}
}

// Тест 4: дублирующиеся слова
func TestFindAnagrams_Duplicates(t *testing.T) {
	input := []string{"Пятак", "пятак", "ПЯТКА", "Тяпка"}
	expected := map[string][]string{
		"пятак": {"пятак", "пятак", "пятка", "тяпка"},
	}

	got := findAnagrams(input)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Ожидалось %+v, получено %+v", expected, got)
	}
}

// Тест 5: несколько множеств анаграмм
func TestFindAnagrams_MultipleGroups(t *testing.T) {
	input := []string{"кот", "ток", "окт", "нос", "сон", "сено", "неос"}
	expected := map[string][]string{
		"кот":  {"кот", "окт", "ток"},
		"нос":  {"нос", "сон"},
		"сено": {"неос", "сено"},
	}

	got := findAnagrams(input)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Ожидалось %+v, получено %+v", expected, got)
	}
}
