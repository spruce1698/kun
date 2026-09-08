package create

import (
	"testing"
)

func TestCmdCreateDBRepository_NoJSONTagFlag(t *testing.T) {
	flag := CmdCreateDBRepository.Flags().Lookup("json-tag")
	if flag != nil {
		t.Fatal("expected flag --json-tag to be removed")
	}
	alias := CmdCreateDBRepository.Flags().Lookup("json")
	if alias != nil {
		t.Fatal("expected flag --json to be removed")
	}
}

func TestCmdCreate_DryRunFlags(t *testing.T) {
	cmds := []*struct {
		name string
		cmd  interface {
			Flags() *interface{ Lookup(string) interface{} }
		}
	}{
		// 校验所有子命令均正确挂载 --dry-run
	}
	_ = cmds

	for _, cmd := range []*struct {
		name string
		flag bool
	}{
		{name: "hdl", flag: CmdCreateHandler.Flags().Lookup("dry-run") != nil},
		{name: "svc", flag: CmdCreateService.Flags().Lookup("dry-run") != nil},
		{name: "hs", flag: CmdCreateHandlerAndService.Flags().Lookup("dry-run") != nil},
		{name: "rt", flag: CmdCreateRouter.Flags().Lookup("dry-run") != nil},
		{name: "db", flag: CmdCreateDBRepository.Flags().Lookup("dry-run") != nil},
		{name: "cache", flag: CmdCreateCacheRepository.Flags().Lookup("dry-run") != nil},
		{name: "crud", flag: CmdCreateCRUD.Flags().Lookup("dry-run") != nil},
	} {
		if !cmd.flag {
			t.Fatalf("expected command %s to have --dry-run flag", cmd.name)
		}
	}
}
