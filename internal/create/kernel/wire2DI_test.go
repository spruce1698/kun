package kernel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMarker = "// ==== db don't edit this line.===="

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// 对 CRLF 文件注入后不得产生混合行尾。
func TestWireProcess_PreservesCRLF(t *testing.T) {
	src := "package db\r\n\r\nvar Set = wire.NewSet(\r\n\t" + testMarker + "\r\n)\r\n"
	p := writeTemp(t, "serverDI.go", src)

	if err := wireProcess(p, testMarker, "\tNewDemoDb,"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)

	if !strings.Contains(text, "NewDemoDb") {
		t.Fatalf("injection did not happen: %q", text)
	}
	// 每个 \n 都必须有配套的 \r,否则就是混合行尾
	lf := strings.Count(text, "\n")
	crlf := strings.Count(text, "\r\n")
	if lf != crlf {
		t.Fatalf("mixed line endings: %d LF vs %d CRLF in %q", lf, crlf, text)
	}
}

// LF 文件保持 LF,不得被改成 CRLF。
func TestWireProcess_PreservesLF(t *testing.T) {
	src := "package db\n\nvar Set = wire.NewSet(\n\t" + testMarker + "\n)\n"
	p := writeTemp(t, "serverDI.go", src)

	if err := wireProcess(p, testMarker, "\tNewDemoDb,"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if strings.Contains(string(got), "\r") {
		t.Fatalf("LF file must not gain CR: %q", string(got))
	}
}

// 重复运行必须幂等,不能重复注入。
func TestWireProcess_Idempotent(t *testing.T) {
	src := "package db\n\nvar Set = wire.NewSet(\n\t" + testMarker + "\n)\n"
	p := writeTemp(t, "serverDI.go", src)

	for i := 0; i < 3; i++ {
		if err := wireProcess(p, testMarker, "\tNewDemoDb,"); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := os.ReadFile(p)
	if n := strings.Count(string(got), "NewDemoDb"); n != 1 {
		t.Fatalf("expected exactly 1 injection, got %d:\n%s", n, string(got))
	}
}

// 关键回归:gofmt 重排缩进后仍须判定为"已注入",否则第二次运行会重复插入。
func TestWireProcess_IdempotentAfterReindent(t *testing.T) {
	// 模拟 gofmt 把缩进从 tab 改成不同宽度、并把多行压紧
	src := "package db\n\nvar Set = wire.NewSet(\n    NewDemoDb,\n\t" + testMarker + "\n)\n"
	p := writeTemp(t, "serverDI.go", src)

	if err := wireProcess(p, testMarker, "\tNewDemoDb,"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if n := strings.Count(string(got), "NewDemoDb"); n != 1 {
		t.Fatalf("re-indented content must be detected as already injected, got %d:\n%s", n, string(got))
	}
}

// 多行内容跨行重排后同样要幂等。
func TestWireProcess_IdempotentMultilineReindented(t *testing.T) {
	appended := "\twire.Struct(new(DemoCtx), \"*\"),\n\tNewDemoSvc,"
	// 文件里已存在同样内容,但缩进/换行风格不同
	src := "package svc\n\nvar Set = wire.NewSet(\n  wire.Struct(new(DemoCtx),   \"*\"),\n      NewDemoSvc,\n\t" + testMarker + "\n)\n"
	p := writeTemp(t, "serverDI.go", src)

	if err := wireProcess(p, testMarker, appended); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if n := strings.Count(string(got), "NewDemoSvc"); n != 1 {
		t.Fatalf("expected idempotent multiline injection, got %d:\n%s", n, string(got))
	}
}

// 找不到 marker 必须报错,不能静默返回 nil。
func TestWireProcess_MissingMarkerErrors(t *testing.T) {
	p := writeTemp(t, "serverDI.go", "package db\n\nvar Set = wire.NewSet()\n")

	err := wireProcess(p, testMarker, "\tNewDemoDb,")
	if err == nil {
		t.Fatal("missing marker must return an error, not silently succeed")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	cases := [][2]string{
		{"  a   b \n\t c ", "a b c"},
		{"a\r\nb", "a b"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeWhitespace(c[0]); got != c[1] {
			t.Fatalf("normalizeWhitespace(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}
