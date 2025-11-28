package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureOutput(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestTokenizerSimple(t *testing.T) {
	line := "echo hello world"
	toks, err := tokenize(line)
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(toks))
	}
	if toks[0].Val != "echo" || toks[1].Val != "hello" || toks[2].Val != "world" {
		t.Fatalf("unexpected tokens: %+v", toks)
	}
}

func TestTokenizerPipeAndAndOr(t *testing.T) {
	line := "ps | grep ssh && echo ok || echo fail"
	toks, err := tokenize(line)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ps", "|", "grep", "ssh", "&&", "echo", "ok", "||", "echo", "fail"}
	if len(toks) != len(want) {
		t.Fatalf("unexpected number of tokens: %d", len(toks))
	}
}

func TestTokenizerQuotes(t *testing.T) {
	line := `echo "hello world"`
	toks, err := tokenize(line)
	if err != nil {
		t.Fatal(err)
	}
	if toks[1].Val != "hello world" {
		t.Fatalf("expected quoted string preserved, got %q", toks[1].Val)
	}
}

func TestExpandEnv(t *testing.T) {
	os.Setenv("TESTVAR", "VALUE123")
	out := expandEnv("X=$TESTVAR Y=${TESTVAR}")
	if out != "X=VALUE123 Y=VALUE123" {
		t.Fatalf("unexpected env expansion: %q", out)
	}
}

func TestParseSimpleCommand(t *testing.T) {
	toks, _ := tokenize("echo hello")
	segs, err := parseSegments(toks)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment")
	}
	if len(segs[0].Pipeline) != 1 {
		t.Fatalf("expected 1 pipeline element")
	}
	if segs[0].Pipeline[0].Args[0] != "echo" ||
		segs[0].Pipeline[0].Args[1] != "hello" {
		t.Fatalf("unexpected args: %+v", segs[0].Pipeline[0].Args)
	}
}

func TestParsePipeline(t *testing.T) {
	toks, _ := tokenize("ps | grep ssh | wc -l")
	segs, err := parseSegments(toks)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs[0].Pipeline) != 3 {
		t.Fatalf("expected 3 pipeline cmds, got %d", len(segs[0].Pipeline))
	}
}

func TestParseRedirects(t *testing.T) {
	toks, _ := tokenize("cat < input.txt > output.txt")
	segs, _ := parseSegments(toks)
	unit := segs[0].Pipeline[0]
	if unit.Stdin != "input.txt" || unit.Stdout != "output.txt" {
		t.Fatalf("redirects parsed incorrectly: %+v", unit)
	}
}

//
// ---------------- BUILTIN TESTS ----------------
//

func TestBuiltinEcho(t *testing.T) {
	var out bytes.Buffer
	code, err := runBuiltin("echo", []string{"hello", "world"}, nil, &out)
	if err != nil || code != 0 {
		t.Fatal("builtin echo failed")
	}
	if strings.TrimSpace(out.String()) != "hello world" {
		t.Fatalf("unexpected echo output: %q", out.String())
	}
}

func TestBuiltinPwd(t *testing.T) {
	var out bytes.Buffer
	code, err := runBuiltin("pwd", nil, nil, &out)
	if err != nil || code != 0 {
		t.Fatal("pwd failed:", err)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("pwd returned empty")
	}
}

func TestBuiltinCd(t *testing.T) {
	dir := t.TempDir()
	code, err := runBuiltin("cd", []string{dir}, nil, nil)
	if err != nil || code != 0 {
		t.Fatalf("cd failed: %v", err)
	}
	cwd, _ := os.Getwd()
	cwdEval, _ := filepath.EvalSymlinks(cwd)
	dirEval, _ := filepath.EvalSymlinks(dir)
	cwdAbs, _ := filepath.Abs(cwdEval)
	dirAbs, _ := filepath.Abs(dirEval)
	if cwdAbs != dirAbs {
		t.Fatalf("cd did not change directory: %s != %s", cwdAbs, dirAbs)
	}
}


func TestRunSingleExternal(t *testing.T) {
	unit := &CmdUnit{Args: []string{"echo", "OK"}}

	out := captureOutput(func() {
		_, err := runSingle(unit)
		if err != nil {
			t.Fatalf("runSingle error: %v", err)
		}
	})

	if !strings.Contains(out, "OK") {
		t.Fatalf("unexpected output: %q", out)
	}
}


func TestPipelineBuiltins(t *testing.T) {
	units := []*CmdUnit{
		{Args: []string{"echo", "HELLO"}},
		{Args: []string{"cat"}},
	}

	out := captureOutput(func() {
		_, err := runPipeline(units)
		if err != nil {
			t.Fatalf("runPipeline error: %v", err)
		}
	})

	if !strings.Contains(out, "HELLO") {
		t.Fatalf("pipeline output invalid: %q", out)
	}
}


func TestConditionalAnd(t *testing.T) {
	toks, _ := tokenize("echo ok && echo success")
	segs, _ := parseSegments(toks)

	out := captureOutput(func() {
		for _, seg := range segs {
			_, _ = runSegment(seg)
		}
	})

	if !strings.Contains(out, "ok") || !strings.Contains(out, "success") {
		t.Fatalf("AND operator failed: %q", out)
	}
}

func TestConditionalOr(t *testing.T) {
	toks, _ := tokenize("badcmd || echo fallback")
	segs, _ := parseSegments(toks)

	out := captureOutput(func() {
		for _, seg := range segs {
			_, _ = runSegment(seg)
		}
	})

	if !strings.Contains(out, "fallback") {
		t.Fatalf("OR operator failed: %q", out)
	}
}


func TestRedirectOutput(t *testing.T) {
	tmp := t.TempDir()
	file := tmp + "/out.txt"

	unit := &CmdUnit{
		Args:   []string{"echo", "HELLO"},
		Stdout: file,
	}

	_, err := runSingle(unit)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(file)
	if !strings.Contains(string(data), "HELLO") {
		t.Fatalf("redirect failed: %q", string(data))
	}
}

func TestRedirectInput(t *testing.T) {
	tmp := t.TempDir()
	file := tmp + "/in.txt"

	os.WriteFile(file, []byte("WORLD"), 0644)

	unit := &CmdUnit{
		Args:  []string{"cat"},
		Stdin: file,
	}

	out := captureOutput(func() {
		_, err := runSingle(unit)
		if err != nil {
			t.Fatalf("runSingle error: %v", err)
		}
	})

	if strings.TrimSpace(out) != "WORLD" {
		t.Fatalf("stdin redirect failed: %q", out)
	}
}
