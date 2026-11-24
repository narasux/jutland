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
	"github.com/narasux/jutland/pkg/mission/state"
)

type frameGame struct {
	m       *manager.MissionManager
	max     int
	warmup  int
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
		// warmup 帧只负责把缩放缓存/首帧着色器热起来，不计入门禁样本。
		if g.frames >= g.warmup {
			g.times = append(g.times, time.Since(g.started))
		}
		g.started = time.Time{}
	}
	g.frames++
}

func (g *frameGame) Layout(_, _ int) (int, int) { return 1280, 720 }

// recordFrameHistory 把本次门禁结果追加到 lab 的 frame/history.jsonl，供 trend 追踪；
// 未配置 JUTLAND_SIM_HOME 或写入失败都不影响门禁本身的退出码。
func recordFrameHistory(report sim.FrameReport) {
	// 趋势门禁只认默认缩放档位；按档位扫描的 zoom 1/2/8/16 样本不混进 p99 历史。
	if report.Zoom != 0 && report.Zoom != state.DefaultZoom() {
		return
	}
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
	warmup := flag.Int("warmup", 0, "frames to run before recording (cache warmup)")
	budget := flag.Float64("budget-ms", 16.7, "p99 frame budget")
	mission := flag.String("mission", "PearlHarbor1941", "mission name")
	zoom := flag.Int("zoom", state.DefaultZoom(), "map zoom: 1, 2, 4, 8 or 16")
	out := flag.String("out", "", "optional JSON output path")
	flag.Parse()
	if *frames <= 0 {
		fmt.Fprintln(os.Stderr, "frames must be positive")
		os.Exit(1)
	}
	if *warmup < 0 || *warmup >= *frames {
		fmt.Fprintln(os.Stderr, "warmup must satisfy 0 <= warmup < frames")
		os.Exit(1)
	}
	config.G = config.NewDefaultGameSettings()
	m := manager.New(*mission, faction.SideP1)
	m.Resize(1280, 720)
	normalizedZoom := state.NormalizeZoom(*zoom)
	m.State().UI.GameOpts.Zoom = normalizedZoom
	m.State().RefreshCameraSize()
	game := &frameGame{m: m, max: *frames, warmup: *warmup}
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("jutland-frame")
	if err := ebiten.RunGame(game); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report := sim.BuildFrameReport(game.times, *budget)
	report.Zoom = normalizedZoom
	report.Warmup = *warmup
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
