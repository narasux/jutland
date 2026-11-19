package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/mission/faction"
	"github.com/narasux/jutland/pkg/mission/manager"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	"github.com/narasux/jutland/pkg/mission/sim"
)

type frameGame struct {
	m       *manager.MissionManager
	max     int
	frames  int
	times   []time.Duration
	started time.Time
}

func (g *frameGame) Update() error {
	if g.frames >= g.max {
		return ebiten.Termination
	}
	g.started = time.Now()
	_, err := g.m.Update()
	return err
}

func (g *frameGame) Draw(screen *ebiten.Image) {
	g.m.Draw(screen)
	if !g.started.IsZero() {
		g.times = append(g.times, time.Since(g.started))
		g.started = time.Time{}
	}
	g.frames++
}

func (g *frameGame) Layout(_, _ int) (int, int) { return 1280, 720 }

// recordFrameHistory 把本次门禁结果追加到 lab 的 frame/history.jsonl，供 trend 追踪；
// 未配置 JUTLAND_SIM_HOME 或写入失败都不影响门禁本身的退出码。
func recordFrameHistory(report sim.FrameReport) {
	paths, err := sim.ResolvePaths("")
	if err != nil {
		return
	}
	git := sim.CollectGitInfo("")
	entry := sim.NewFrameHistoryEntry(report, git.CommitShort, git.Dirty, time.Now())
	if err := sim.AppendFrameHistory(paths.Reports, entry); err != nil {
		fmt.Fprintf(os.Stderr, "append frame history: %v\n", err)
	}
}

func main() {
	frames := flag.Int("frames", 300, "number of rendered frames")
	budget := flag.Float64("budget-ms", 16.7, "p99 frame budget")
	mission := flag.String("mission", "PearlHarbor1941", "mission name")
	out := flag.String("out", "", "optional JSON output path")
	flag.Parse()
	config.G = config.NewDefaultGameSettings()
	m := manager.New(*mission, faction.SideP1)
	game := &frameGame{m: m, max: *frames}
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("jutland-frame")
	if err := ebiten.RunGame(game); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report := sim.BuildFrameReport(game.times, *budget)
	recordFrameHistory(report)
	if *out != "" {
		if err := sim.WriteFrameReport(*out, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	payload, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(payload))
	if !report.Pass {
		os.Exit(1)
	}
}
