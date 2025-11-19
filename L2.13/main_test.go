package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// helper
func captureOutput(f func()) string {
	var buf bytes.Buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old
	buf.ReadFrom(r)
	return buf.String()
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		input    string
		expected map[int]bool
		hasErr   bool
	}{
		{"1", map[int]bool{1: true}, false},
		{"1,3", map[int]bool{1: true, 3: true}, false},
		{"1-3", map[int]bool{1: true, 2: true, 3: true}, false},
		{"1,3-5", map[int]bool{1: true, 3: true, 4: true, 5: true}, false},
		{"2-2", map[int]bool{2: true}, false},
		{"0", nil, true},
		{"3-a", nil, true},
		{"5-1", nil, true},
		{"1--3", nil, true},
	}

	for _, tt := range tests {
		result, err := parseFields(tt.input)

		if tt.hasErr && err == nil {
			t.Errorf("expected error for input %s", tt.input)
		}
		if !tt.hasErr && err != nil {
			t.Errorf("unexpected error for %s: %v", tt.input, err)
		}
		if !tt.hasErr {
			if len(result) != len(tt.expected) {
				t.Errorf("wrong length for %s: got %d, expected %d",
					tt.input, len(result), len(tt.expected))
			}
			for k := range tt.expected {
				if !result[k] {
					t.Errorf("missing field %d for %s", k, tt.input)
				}
			}
		}
	}
}

func TestProcessLine(t *testing.T) {
	flags := Flags{
		Delimiter: ',',
		Separated: false,
	}

	fields := map[int]bool{1: true, 3: true}

	line := "a,b,c,d"

	out := captureOutput(func() {
		processLine(line, flags, fields)
	})

	expected := "a,c\n"
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestProcessLineSeparated(t *testing.T) {
	flags := Flags{
		Delimiter: ',',
		Separated: true,
	}

	fields := map[int]bool{1: true}

	line := "abc"

	out := captureOutput(func() {
		processLine(line, flags, fields)
	})

	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestProcessFile(t *testing.T) {
	content := "a,b,c\n1,2,3"
	tmp, err := os.CreateTemp("", "cut_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	tmp.WriteString(content)
	tmp.Close()

	flags := Flags{
		Delimiter: ',',
		Separated: false,
		File:      tmp.Name(),
	}

	fields := map[int]bool{2: true}

	out := captureOutput(func() {
		err := processFile(tmp.Name(), flags, fields)
		if err != nil {
			t.Fatalf("processFile error: %v", err)
		}
	})

	expected := "b\n2\n"
	if strings.TrimSpace(out) != strings.TrimSpace(expected) {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, out)
	}
}

// --- интеграционные тесты ---

func TestIntegration_FromSTDIN(t *testing.T) {
	input := "a,b,c\n1,2,3"
	expected := "b\n2\n"

	flags := Flags{
		Fields:    "2",
		Delimiter: ',',
		Separated: false,
		File:      "",
	}

	fields, _ := parseFields("2")

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	w.WriteString(input)
	w.Close()

	out := captureOutput(func() {
		processFile("", flags, fields)
	})

	os.Stdin = oldStdin

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_MultipleRanges(t *testing.T) {
	input := "a,b,c,d,e"
	expected := "a,c,e\n"

	flags := Flags{
		Delimiter: ',',
	}

	fields, _ := parseFields("1,3,5")

	out := captureOutput(func() {
		processLine(input, flags, fields)
	})

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_FieldOutOfRange(t *testing.T) {
	input := "x,y"
	expected := "y\n"

	flags := Flags{Delimiter: ','}
	fields, _ := parseFields("2,3")

	out := captureOutput(func() {
		processLine(input, flags, fields)
	})

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_EmptyLines(t *testing.T) {
	input := "\na,b,c\n\n"
	expected := "\na\n\n\n"

	flags := Flags{Delimiter: ','}
	fields, _ := parseFields("1")

	out := captureOutput(func() {
		for _, line := range strings.Split(input, "\n") {
			processLine(line, flags, fields)
		}
	})

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_ConsecutiveDelimiters(t *testing.T) {
	input := "a,,c,,e"
	expected := "a,c,e\n"

	flags := Flags{Delimiter: ','}
	fields, _ := parseFields("1,3,5")

	out := captureOutput(func() {
		processLine(input, flags, fields)
	})

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_SeparatedSkip(t *testing.T) {
	input := "abc\n1,2,3\nxyz"
	expected := "1\n"

	flags := Flags{
		Delimiter: ',',
		Separated: true,
	}

	fields, _ := parseFields("1")

	out := captureOutput(func() {
		for _, line := range strings.Split(input, "\n") {
			processLine(line, flags, fields)
		}
	})

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_UTF8Delimiter(t *testing.T) {
	input := "a☆b☆c"
	expected := "b\n"

	flags := Flags{
		Delimiter: '☆',
	}

	fields, _ := parseFields("2")

	out := captureOutput(func() {
		processLine(input, flags, fields)
	})

	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}

func TestIntegration_NoMatchingFields(t *testing.T) {
	input := "1,2,3"
	expected := "\n"

	flags := Flags{Delimiter: ','}
	fields, _ := parseFields("10")

	out := captureOutput(func() {
		processLine(input, flags, fields)
	})

	if out != expected {
		t.Errorf("expected empty line, got %q", out)
	}
}

func TestIntegration_NoDelimiter(t *testing.T) {
	input := "hello"
	flags := Flags{
		Delimiter: ',',
	}

	fields, _ := parseFields("1")

	out := captureOutput(func() {
		processLine(input, flags, fields)
	})

	expected := "hello\n"
	if out != expected {
		t.Errorf("expected %q, got %q", expected, out)
	}
}
