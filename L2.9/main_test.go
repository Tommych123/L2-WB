package main

import "testing"

func TestUnpack(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"a4bc2d5e", "aaaabccddddde", false}, // тесты из описания задания
		{"abcd", "abcd", false},
		{"", "", false},
		{"45", "", true},
		{"qwe\\4\\5", "qwe45", false},
		{"qwe\\45", "qwe44444", false},
		{"qwe\\\\5", "qwe\\\\\\\\\\", false}, // экранированный обратный слеш
		{"\\", "", true},                     // некорректно: слеш в конце
	}

	for _, tt := range tests {
		got, err := unpackString(tt.input)
		if tt.wantErr && err == nil {
			t.Errorf("Unpack(%q) = nil error, want error", tt.input)
			continue
		}
		if !tt.wantErr && err != nil {
			t.Errorf("Unpack(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("Unpack(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
