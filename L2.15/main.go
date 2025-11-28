package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// Тип токена — слово, &&, ||, |
type TokenKind int

const (
	TK_WORD TokenKind = iota // обычное слово (команда или аргумент)
	TK_AND                   // &&
	TK_OR                    // ||
	TK_PIPE                  // |
)

// Структура токена
type Token struct {
	Kind TokenKind
	Val  string
}

// Одна команда (ls, echo ...) с возможным редиректом ввода/вывода
type CmdUnit struct {
	Args   []string // аргументы, включая имя команды
	Stdin  string   // файл для < (если пусто — stdin)
	Stdout string   // файл для > (если пусто — stdout)
	Append bool     // (не используется, под "+" redirection)
}

// Сегмент — это команда или pipeline, плюс оператор && или ||
type CmdSegment struct {
	Pipeline []*CmdUnit // список Unit, разделённых |
	NextOp   TokenKind  // оператор после сегмента
}

// Группа процессов, которой нужно слать SIGINT
var (
	pidMutex    sync.Mutex
	currentPgrp int = 0
)

// main — основной цикл командной строки shell
func main() {

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT)

	go func() {
		for sig := range sigch {
			pidMutex.Lock()
			pgrp := currentPgrp
			pidMutex.Unlock()

			if pgrp != 0 {
				// посылаем сигнал всей группе процессов отрицательный pid = process group
				_ = syscall.Kill(-pgrp, sig.(syscall.Signal))
			}
		}
	}()

	r := bufio.NewReader(os.Stdin)

	for {
		cwd, _ := os.Getwd()
		fmt.Printf("%s$ ", cwd)

		line, err := r.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) { // Ctrl+D
				fmt.Println()
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, "read error:", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Разбить строку на токены
		toks, terr := tokenize(line)
		if terr != nil {
			fmt.Fprintln(os.Stderr, "tokenize:", terr)
			continue
		}

		// Разобрать токены в сегменты (pipeline + && || )
		segs, perr := parseSegments(toks)
		if perr != nil {
			fmt.Fprintln(os.Stderr, "parse:", perr)
			continue
		}

		// Выполнение команд с учётом && и ||
		lastOk := true
		for i, seg := range segs {

			if i > 0 {
				prev := segs[i-1].NextOp

				// && — выполнять только если предыдущая была успешной
				if prev == TK_AND && !lastOk {
					lastOk = false
					continue
				}

				// || — выполнять только если предыдущая провалилась
				if prev == TK_OR && lastOk {
					lastOk = true
					continue
				}
			}

			code, err := runSegment(seg)
			lastOk = (err == nil && code == 0)

			if err != nil {
				fmt.Fprintln(os.Stderr, "exec error:", err)
			}
		}
	}
}

// isSpace — проверка пробела
func isSpace(b byte) bool {
	return b == ' ' || b == '\t'
}

// tokenize — разбивает строку на токены: слова, |, ||, &&
func tokenize(line string) ([]Token, error) {
	var toks []Token
	i := 0
	n := len(line)

	for i < n {

		if isSpace(line[i]) {
			i++
			continue
		}

		// && оператор
		if i+1 < n && line[i] == '&' && line[i+1] == '&' {
			toks = append(toks, Token{Kind: TK_AND, Val: "&&"})
			i += 2
			continue
		}

		// || оператор
		if i+1 < n && line[i] == '|' && line[i+1] == '|' {
			toks = append(toks, Token{Kind: TK_OR, Val: "||"})
			i += 2
			continue
		}

		// | оператор
		if line[i] == '|' {
			toks = append(toks, Token{Kind: TK_PIPE, Val: "|"})
			i++
			continue
		}

		// Кавычки "..." или '...'
		if line[i] == '"' || line[i] == '\'' {
			q := line[i]
			i++
			start := i
			for i < n && line[i] != q {
				i++
			}
			if i >= n {
				return nil, fmt.Errorf("unterminated quote")
			}
			toks = append(toks, Token{Kind: TK_WORD, Val: line[start:i]})
			i++
			continue
		}

		// Обычное слово
		start := i
		for i < n && !isSpace(line[i]) {
			if line[i] == '|' {
				break
			}
			if line[i] == '&' && i+1 < n && line[i+1] == '&' {
				break
			}
			i++
		}
		toks = append(toks, Token{Kind: TK_WORD, Val: line[start:i]})
	}
	return toks, nil
}

// parseSegments — делит токены на сегменты по && и ||
func parseSegments(toks []Token) ([]*CmdSegment, error) {
	var segs []*CmdSegment
	i := 0
	n := len(toks)

	for i < n {
		j := i

		// период до && или ||
		for j < n && toks[j].Kind != TK_AND && toks[j].Kind != TK_OR {
			j++
		}

		segmentTokens := toks[i:j]
		seg, err := parsePipelineSegment(segmentTokens)
		if err != nil {
			return nil, err
		}

		// оператор после сегмента
		if j < n {
			seg.NextOp = toks[j].Kind
			j++
		} else {
			seg.NextOp = TK_WORD
		}

		segs = append(segs, seg)
		i = j
	}

	return segs, nil
}

// parsePipelineSegment — разбивает сегмент на части через |
func parsePipelineSegment(toks []Token) (*CmdSegment, error) {
	if len(toks) == 0 {
		return nil, fmt.Errorf("empty segment")
	}

	var units []*CmdUnit
	start := 0

	for i := 0; i <= len(toks); i++ {
		if i == len(toks) || toks[i].Kind == TK_PIPE {
			sub := toks[start:i]
			if len(sub) == 0 {
				return nil, fmt.Errorf("empty command in pipeline")
			}

			unit, err := parseCmdUnit(sub)
			if err != nil {
				return nil, err
			}

			units = append(units, unit)
			start = i + 1
		}
	}

	return &CmdSegment{Pipeline: units, NextOp: TK_WORD}, nil
}

// parseCmdUnit — парсит одну команду с аргументами и редиректами
func parseCmdUnit(toks []Token) (*CmdUnit, error) {
	args := []string{}
	var stdinFile, stdoutFile string

	for i := 0; i < len(toks); {
		if toks[i].Kind != TK_WORD {
			return nil, fmt.Errorf("unexpected token")
		}
		w := toks[i].Val

		// формы >file и <file
		if strings.HasPrefix(w, ">") && len(w) > 1 {
			stdoutFile = w[1:]
			i++
			continue
		}
		if strings.HasPrefix(w, "<") && len(w) > 1 {
			stdinFile = w[1:]
			i++
			continue
		}

		// формы > file  / < file
		if w == ">" {
			i++
			if i >= len(toks) {
				return nil, fmt.Errorf("expected filename after >")
			}
			stdoutFile = toks[i].Val
			i++
			continue
		}
		if w == "<" {
			i++
			if i >= len(toks) {
				return nil, fmt.Errorf("expected filename after <")
			}
			stdinFile = toks[i].Val
			i++
			continue
		}

		// Подстановка переменных $VAR
		w = expandEnv(w)

		args = append(args, w)
		i++
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return &CmdUnit{Args: args, Stdin: stdinFile, Stdout: stdoutFile}, nil
}

// isAlnum — буква или цифра
func isAlnum(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}

// expandEnv — подставляет $VAR и ${VAR}
func expandEnv(s string) string {
	var buf bytes.Buffer
	i := 0
	n := len(s)

	for i < n {
		if s[i] == '$' {

			// ${VAR} формат
			if i+1 < n && s[i+1] == '{' {
				j := i + 2
				for j < n && s[j] != '}' {
					j++
				}
				if j < n {
					name := s[i+2 : j]
					buf.WriteString(os.Getenv(name))
					i = j + 1
					continue
				}
			}

			// $VAR формат
			j := i + 1
			for j < n && (isAlnum(s[j]) || s[j] == '_') {
				j++
			}
			if j > i+1 {
				name := s[i+1 : j]
				buf.WriteString(os.Getenv(name))
				i = j
				continue
			}

			buf.WriteByte('$')
			i++
			continue
		}

		buf.WriteByte(s[i])
		i++
	}
	return buf.String()
}

// runSegment — выполняет один сегмент: либо одну команду, либо pipeline
func runSegment(seg *CmdSegment) (int, error) {
	if len(seg.Pipeline) == 1 {
		return runSingle(seg.Pipeline[0])
	}
	return runPipeline(seg.Pipeline)
}

// runSingle — выполнение одиночной команды (builtin или внешняя)
func runSingle(unit *CmdUnit) (int, error) {
	name := unit.Args[0]

	if isBuiltin(name) {

		var in io.Reader = os.Stdin
		var out io.Writer = os.Stdout

		// редирект <
		if unit.Stdin != "" {
			f, err := os.Open(unit.Stdin)
			if err != nil {
				return 1, err
			}
			defer f.Close()
			in = f
		}

		// редирект >
		if unit.Stdout != "" {
			f, err := os.Create(unit.Stdout)
			if err != nil {
				return 1, err
			}
			defer f.Close()
			out = f
		}

		return runBuiltin(name, unit.Args[1:], in, out)
	}

	cmd := exec.Command(name, unit.Args[1:]...)

	// stdin
	if unit.Stdin != "" {
		f, err := os.Open(unit.Stdin)
		if err != nil {
			return 1, err
		}
		defer f.Close()
		cmd.Stdin = f
	} else {
		cmd.Stdin = os.Stdin
	}

	// stdout
	if unit.Stdout != "" {
		f, err := os.Create(unit.Stdout)
		if err != nil {
			return 1, err
		}
		defer f.Close()
		cmd.Stdout = f
	} else {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr

	// Каждую команду запускаем в независимой группе процессов чтобы Ctrl+C убивал только её
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		return 1, err
	}

	// сохраняем группу процессов
	pidMutex.Lock()
	currentPgrp = cmd.Process.Pid
	pidMutex.Unlock()

	err := cmd.Wait()

	// очищаем pgrp
	pidMutex.Lock()
	currentPgrp = 0
	pidMutex.Unlock()

	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			ws := ee.Sys().(syscall.WaitStatus)
			return ws.ExitStatus(), nil
		}
		return 1, err
	}

	return 0, nil
}

// runPipeline — выполнение конвейера команд "A | B | C"
func runPipeline(units []*CmdUnit) (int, error) {
	n := len(units)

	// создаём пайпы
	rdrs := make([]*os.File, n-1)
	wtrs := make([]*os.File, n-1)

	for i := 0; i < n-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return 1, err
		}
		rdrs[i] = r
		wtrs[i] = w
	}

	var wg sync.WaitGroup
	cmds := make([]*exec.Cmd, n)
	var leaderPid int

	for i, unit := range units {
		isB := isBuiltin(unit.Args[0])

		// stdin
		var in io.Reader = os.Stdin
		if i == 0 { // первая команда
			if unit.Stdin != "" {
				f, err := os.Open(unit.Stdin)
				if err != nil {
					return 1, err
				}
				defer f.Close()
				in = f
			}
		} else {
			in = rdrs[i-1]
		}

		// stdout
		var out io.Writer = os.Stdout
		if i == n-1 { // последняя команда
			if unit.Stdout != "" {
				f, err := os.Create(unit.Stdout)
				if err != nil {
					return 1, err
				}
				defer f.Close()
				out = f
			}
		} else {
			out = wtrs[i]
		}
		if isB {
			wg.Add(1)
			go func(u *CmdUnit, r io.Reader, w io.Writer) {
				defer wg.Done()
				_, _ = runBuiltin(u.Args[0], u.Args[1:], r, w)
			}(unit, in, out)
			continue
		}
		cmd := exec.Command(unit.Args[0], unit.Args[1:]...)
		cmd.Stdin = toReadCloser(in)
		cmd.Stdout = toWriteCloser(out)
		cmd.Stderr = os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			return 1, err
		}

		if leaderPid == 0 {
			leaderPid = cmd.Process.Pid
			pidMutex.Lock()
			currentPgrp = leaderPid
			pidMutex.Unlock()
		}

		cmds[i] = cmd
	}

	// закрываем записи pipe у родителя
	for _, w := range wtrs {
		if w != nil {
			w.Close()
		}
	}

	var finalErr error

	// ждём внешние команды
	for _, c := range cmds {
		if c == nil {
			continue
		}
		if err := c.Wait(); err != nil {
			finalErr = err
		}
	}

	// ждём builtin
	wg.Wait()

	// сбрасываем pgrp
	pidMutex.Lock()
	currentPgrp = 0
	pidMutex.Unlock()

	if finalErr != nil {
		if ee, ok := finalErr.(*exec.ExitError); ok {
			ws := ee.Sys().(syscall.WaitStatus)
			return ws.ExitStatus(), nil
		}
		return 1, finalErr
	}

	return 0, nil
}

// toReadCloser — превращает io.Reader в io.ReadCloser
func toReadCloser(r io.Reader) io.ReadCloser {
	if rc, ok := r.(io.ReadCloser); ok {
		return rc
	}
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.Copy(pw, r)
		_ = pw.Close()
	}()
	return pr
}

// toWriteCloser — превращает io.Writer в io.WriteCloser
func toWriteCloser(w io.Writer) io.WriteCloser {
	if wc, ok := w.(io.WriteCloser); ok {
		return wc
	}
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.Copy(w, pr)
		_ = pr.Close()
	}()
	return pw
}

// isBuiltin — проверяет встроенные команды
func isBuiltin(name string) bool {
	switch name {
	case "cd", "pwd", "echo", "kill", "ps", "exit":
		return true
	default:
		return false
	}
}

// runBuiltin — запуск встроенных команд
func runBuiltin(name string, args []string, in io.Reader, out io.Writer) (int, error) {
	switch name {

	case "cd":
		return builtinCd(args)

	case "pwd":
		return builtinPwd(out)

	case "echo":
		return builtinEcho(args, out)

	case "kill":
		return builtinKill(args)

	case "ps":
		return builtinPs(out)

	case "exit":
		os.Exit(0)
	}

	return 1, fmt.Errorf("unknown builtin %s", name)
}

// builtinCd — смена директории
func builtinCd(args []string) (int, error) {
	target := ""

	if len(args) == 0 {
		target = os.Getenv("HOME")
		if target == "" {
			target = "/"
		}
	} else {
		target = args[0]

		// обработка  ~/
		if strings.HasPrefix(target, "~") {
			home := os.Getenv("HOME")
			if home == "" {
				return 1, fmt.Errorf("HOME not set")
			}
			target = filepath.Join(home, target[1:])
		}
	}

	if err := os.Chdir(target); err != nil {
		return 1, err
	}
	return 0, nil
}

// builtinPwd — вывод текущей директории
func builtinPwd(out io.Writer) (int, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return 1, err
	}
	fmt.Fprintln(out, cwd)
	return 0, nil
}

// builtinEcho — печатает аргументы
func builtinEcho(args []string, out io.Writer) (int, error) {
	fmt.Fprintln(out, strings.Join(args, " "))
	return 0, nil
}

// builtinKill — kill PID или kill -9 PID
func builtinKill(args []string) (int, error) {
	if len(args) == 0 {
		return 1, fmt.Errorf("kill: missing pid")
	}

	pidArg := args[0]
	var sig syscall.Signal = syscall.SIGTERM
	var pid int
	var err error

	// kill -9 PID
	if strings.HasPrefix(pidArg, "-") && len(args) >= 2 {
		sn, e := strconv.Atoi(pidArg[1:])
		if e == nil {
			sig = syscall.Signal(sn)
			pid, err = strconv.Atoi(args[1])
			if err != nil {
				return 1, err
			}
		} else {
			return 1, e
		}
	} else {
		// kill PID
		pid, err = strconv.Atoi(pidArg)
		if err != nil {
			return 1, err
		}
	}

	if err := syscall.Kill(pid, sig); err != nil {
		return 1, err
	}
	return 0, nil
}

// builtinPs — вызывает системный ps
func builtinPs(out io.Writer) (int, error) {
	cmd := exec.Command("ps", "-e", "-o", "pid,ppid,comm")
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return 1, err
	}
	return 0, nil
}
