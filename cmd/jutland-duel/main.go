package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/narasux/jutland/internal/cliutil"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/sim/duel"
	"github.com/narasux/jutland/pkg/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "duel":
		return cmdDuel(args[1:])
	case "version":
		fmt.Print(version.GetVersion())
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func cmdDuel(args []string) error {
	fs := flag.NewFlagSet("duel", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	open := fs.Bool("open", false, "open report in browser")
	seed := fs.Int64("seed", -1, "override scenario seed")
	runs := fs.Int("runs", 0, "override run count")
	baseline := fs.String("baseline", "", "baseline file/name for metric deltas")
	ticks := fs.Int("ticks", 0, "override max ticks")
	flagArgs, pos := cliutil.Split(args, "open")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	arg := ""
	if len(pos) > 0 {
		arg = strings.TrimSpace(pos[0])
	}
	if arg == "" {
		return fmt.Errorf("duel requires a scenario path")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	full, err := sim.ResolveScenarioPath(paths, arg)
	if err != nil {
		return err
	}
	sc, err := sim.LoadScenario(full)
	if err != nil {
		return err
	}
	if sc.Mode != sim.ModeDuel {
		return fmt.Errorf("scenario mode is %q, want duel", sc.Mode)
	}
	if *seed >= 0 {
		sc.Seed = *seed
	}
	if *runs > 0 {
		sc.Runs = *runs
	}
	if *ticks > 0 {
		sc.Duel.MaxTicks = *ticks
	}
	sc.ApplyDefaults()
	if err := sc.Validate(); err != nil {
		return err
	}
	result, err := duel.RunDuel(sc, time.Now())
	if err != nil {
		return err
	}
	if err := applyBaselineIfSet(paths, *baseline, result); err != nil {
		return err
	}
	command := "jutland-duel " + strings.Join(os.Args[1:], " ")
	outInfo, err := sim.WriteReport(paths, result, full, command)
	if err != nil {
		return err
	}
	fmt.Printf("scenario: %s (duel, %s)\n", sc.Name, result.Fidelity)
	fmt.Printf("run:      %s\n", outInfo.RunID)
	fmt.Printf("report:   %s\n", outInfo.ReportHTML)
	fmt.Printf("summary:  %s\n", outInfo.Summary)
	fmt.Printf("ai:       %s\n", outInfo.AIContext)
	if *open {
		return openPath(outInfo.ReportHTML)
	}
	return nil
}

func applyBaselineIfSet(paths sim.Paths, arg string, result *sim.Result) error {
	if strings.TrimSpace(arg) == "" {
		return nil
	}
	path, err := sim.ResolveBaselinePath(paths, arg)
	if err != nil {
		return err
	}
	b, err := sim.LoadBaseline(path)
	if err != nil {
		return err
	}
	_, err = sim.ApplyBaseline(result, b)
	return err
}

func openPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open report: %w (path: %s)", err, path)
	}
	return nil
}

const usage = `jutland-duel - F1 single-ship duel runner

Requires JUTLAND_SIM_HOME. F1 imports the game's unit/config initialization,
so it needs an Ebiten-capable environment (no game window is opened).

Commands:
  duel <scenario> [flags]      run one real-config duel and write a report
  version
  help

Flags:
  --out DIR                    report directory override
  --open                       open report.html
  --baseline FILE              baseline file/name for metric deltas
  --seed N                     override scenario seed
  --runs N                     override run count (seeds seed..seed+N-1)
  --ticks N                    override max ticks
`
