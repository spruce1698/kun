package create

import (
	"testing"
)

func TestCmdCreateDBRepository_Flags(t *testing.T) {
	flag := CmdCreateDBRepository.Flags().Lookup("json-tag")
	if flag == nil {
		t.Fatal("expected flag --json-tag to be registered")
	}
	if flag.DefValue != "false" {
		t.Fatalf("expected default value 'false', got %q", flag.DefValue)
	}
	if flag.Shorthand != "j" {
		t.Fatalf("expected shorthand 'j', got %q", flag.Shorthand)
	}
}
