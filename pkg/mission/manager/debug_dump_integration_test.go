package manager

import (
	"strings"
	"testing"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/hacker/cheat"
)

func TestDumpStateOnRealMission(t *testing.T) {
	oldBaseDir := config.BaseDir
	config.BaseDir = t.TempDir()
	t.Cleanup(func() { config.BaseDir = oldBaseDir })

	manager := New("PearlHarbor1941")
	output := (&cheat.DumpMisState{}).Exec(manager.state)
	if strings.Contains(output, "failed") {
		t.Fatalf("dump state failed: %s", output)
	}
}
