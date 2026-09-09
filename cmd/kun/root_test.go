package kun

import (
	"bytes"
	"errors"
	"testing"
)

func TestIsUsageError(t *testing.T) {
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
		{err: errors.New("create requires a subcommand: hdl, svc, hs, rt, db, cache"), expected: true},
		{err: errors.New("unknown command \"curd\" for \"kun create\""), expected: true},
		{err: errors.New("unknown command \"foo\" for \"kun\""), expected: true},
		{err: errors.New("unknown flag: --unknown-flag"), expected: true},
		{err: errors.New("unknown shorthand flag: 'z' in -z"), expected: true},
		{err: errors.New("connect db server fail: invalid DSN"), expected: false},
	}

	for _, tt := range tests {
		got := IsUsageError(tt.err)
		if got != tt.expected {
			t.Errorf("IsUsageError(%q) = %v; want %v", tt.err, got, tt.expected)
		}
	}
}

func TestExecute_UsageErrorWrapped(t *testing.T) {
	var buf bytes.Buffer
	CmdRoot.SetOut(&buf)
	CmdRoot.SetErr(&buf)
	CmdRoot.SetArgs([]string{"create", "curd"})

	err := Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var usageErr *UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected *UsageError, got %T: %v", err, err)
	}

	if usageErr.CmdPath != "kun create" {
		t.Errorf("expected CmdPath to be 'kun create', got: %s", usageErr.CmdPath)
	}
}

func TestExecute_AllErrorsWrapped(t *testing.T) {
	var buf bytes.Buffer
	CmdRoot.SetOut(&buf)
	CmdRoot.SetErr(&buf)
	CmdRoot.SetArgs([]string{"create", "db", "invalid_dsn", "*"})

	err := Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) {
		t.Fatalf("expected *CommandError, got %T: %v", err, err)
	}

	if cmdErr.CmdPath != "kun create db" {
		t.Errorf("expected CmdPath to be 'kun create db', got: %s", cmdErr.CmdPath)
	}
}
