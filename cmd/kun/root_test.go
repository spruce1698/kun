package kun

import (
	"bytes"
	"errors"
	"testing"
)

func TestIsArgsError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{err: nil, expected: false},
		{err: errors.New("accepts between 1 and 2 arg(s), received 0"), expected: true},
		{err: errors.New("accepts between 1 and 2 arg(s), received 3"), expected: true},
		{err: errors.New("accepts 1 arg(s), received 0"), expected: true},
		{err: errors.New("accepts 1 arg(s), received 2"), expected: true},
		{err: errors.New("requires at least 1 arg(s), received 0"), expected: true},
		{err: errors.New("accepts at most 1 arg(s), received 2"), expected: true},
		{err: errors.New("accepts at most 1 arg, use '--' to pass arguments"), expected: true},
		{err: errors.New("unknown command \"foo\" for \"kun\""), expected: false},
		{err: errors.New("connect db server fail: invalid DSN"), expected: false},
	}

	for _, tt := range tests {
		got := IsArgsError(tt.err)
		if got != tt.expected {
			t.Errorf("IsArgsError(%q) = %v; want %v", tt.err, got, tt.expected)
		}
	}
}

func TestExecute_ArgsErrorPrintsHelp(t *testing.T) {
	var buf bytes.Buffer
	CmdRoot.SetOut(&buf)
	CmdRoot.SetErr(&buf)
	CmdRoot.SetArgs([]string{"create", "db"})

	err := Execute()
	if !errors.Is(err, ErrSilent) {
		t.Fatalf("expected ErrSilent, got %v", err)
	}

	out := buf.String()
	if out == "" {
		t.Fatal("expected help output, got empty")
	}

	// 验证输出中包含 db 命令的帮助信息
	if !bytes.Contains(buf.Bytes(), []byte("Create a new DB repository")) {
		t.Errorf("expected help to contain 'Create a new DB repository', got: %s", out)
	}
}
