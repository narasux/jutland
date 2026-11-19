package game

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/narasux/jutland/pkg/mission/observer"
)

// attachDebugRecorder enables JSONL combat recording when
// JUTLAND_DEBUG_RECORD_DIR is set. It is intentionally env-gated so the
// production game path keeps the observer nil and pays no recording cost.
func (g *Game) attachDebugRecorder() {
	g.closeDebugRecorder()
	dir := strings.TrimSpace(os.Getenv("JUTLAND_DEBUG_RECORD_DIR"))
	if dir == "" || g.missionMgr == nil {
		return
	}
	rec, path, err := observer.NewFileRecorder(filepath.Join(dir, g.curMission))
	if err != nil {
		log.Printf("[WARN] debug recording disabled: %v", err)
		return
	}
	g.missionMgr.SetObserver(rec)
	g.debugRecord = rec
	log.Printf("[INFO] debug combat recording: %s", path)
}

func (g *Game) closeDebugRecorder() {
	if g.debugRecord == nil {
		return
	}
	if err := g.debugRecord.Close(); err != nil {
		log.Printf("[WARN] close debug recording: %v", err)
	}
	g.debugRecord = nil
}
