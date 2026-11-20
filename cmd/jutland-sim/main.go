package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/narasux/jutland/internal/cliutil"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	switch args[0] {
	case "init":
		return cmdInit(args[1:])
	case "demo":
		return cmdDemo(args[1:])
	case "shot":
		return cmdShot(args[1:])
	case "penetration":
		return cmdPenetration(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "validate":
		return cmdValidate(args[1:])
	case "trend":
		return cmdTrend(args[1:])
	case "analyze":
		return cmdAnalyze(args[1:])
	case "baseline":
		return cmdBaseline(args[1:])
	case "perf":
		return cmdPerf(args[1:])
	case "compare":
		return cmdCompare(args[1:])
	case "compare-record":
		return cmdCompareRecord(args[1:])
	case "llm-package":
		return cmdLLMPackage(args[1:])
	case "llm-review":
		return cmdLLMReview(args[1:])
	case "evaluation":
		return cmdEvaluation(args[1:])
	case "watch":
		return cmdWatch(args[1:])
	case "archive":
		return cmdArchive(args[1:])
	case "version":
		fmt.Print(version.GetVersion())
		return nil
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText)
	}
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	paths, err := sim.ResolvePathsForInit()
	if err != nil {
		return err
	}
	if err := sim.EnsureLayout(paths); err != nil {
		return err
	}
	if err := writeIfMissing(filepath.Join(paths.Scenarios, "demo-report.json5"), sampleScenario); err != nil {
		return err
	}
	if err := writeIfMissing(filepath.Join(paths.Home, "README.md"), labReadme); err != nil {
		return err
	}
	fmt.Printf("lab initialized: %s\n", paths.Home)
	fmt.Printf("  scenarios: %s\n", paths.Scenarios)
	fmt.Printf("  golden:    %s\n", paths.Golden)
	fmt.Printf("  reports:   %s\n", paths.Reports)
	fmt.Println("next: jutland-sim demo")
	return nil
}

func cmdDemo(args []string) error {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	scenarioPath := fs.String("scenario", "", "scenario path (optional; built-in demo if omitted)")
	open := fs.Bool("open", false, "open report in browser")
	seed := fs.Int64("seed", -1, "override scenario seed")
	runs := fs.Int("runs", 0, "override run count")
	if err := fs.Parse(args); err != nil {
		return err
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	var sc sim.Scenario
	if strings.TrimSpace(*scenarioPath) == "" {
		sc = sim.DefaultDemoScenario()
		*scenarioPath = "built-in:demo-report"
	} else {
		full, err := sim.ResolveScenarioPath(paths, *scenarioPath)
		if err != nil {
			return err
		}
		loaded, err := sim.LoadScenario(full)
		if err != nil {
			return err
		}
		sc = *loaded
		*scenarioPath = full
	}
	return runMock(paths, sc, *scenarioPath, *seed, *runs, *open)
}

func cmdShot(args []string) error {
	fs := flag.NewFlagSet("shot", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	open := fs.Bool("open", false, "open report in browser")
	seed := fs.Int64("seed", -1, "override scenario seed")
	runs := fs.Int("runs", 0, "override run count")
	baseline := fs.String("baseline", "", "baseline file/name for metric deltas")
	flagArgs, pos := cliutil.Split(args, "open")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	scenarioArg := ""
	if len(pos) > 0 {
		scenarioArg = strings.TrimSpace(pos[0])
	}
	if scenarioArg == "" {
		return fmt.Errorf("shot requires a scenario path")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	full, err := sim.ResolveScenarioPath(paths, scenarioArg)
	if err != nil {
		return err
	}
	sc, err := sim.LoadScenario(full)
	if err != nil {
		return err
	}
	if sc.Mode != sim.ModeShot {
		return fmt.Errorf("scenario mode is %q, want shot", sc.Mode)
	}
	if *seed >= 0 {
		sc.Seed = *seed
	}
	if *runs > 0 {
		sc.Runs = *runs
	}
	sc.ApplyDefaults()
	if err := sc.Validate(); err != nil {
		return err
	}
	result, err := sim.RunShot(sc, time.Now())
	if err != nil {
		return err
	}
	if err := applyBaselineIfSet(paths, *baseline, result); err != nil {
		return err
	}
	command := "jutland-sim " + strings.Join(os.Args[1:], " ")
	outInfo, err := sim.WriteReport(paths, result, full, command)
	if err != nil {
		return err
	}
	fmt.Printf("scenario: %s (shot, F0)\n", sc.Name)
	fmt.Printf("run:      %s\n", outInfo.RunID)
	fmt.Printf("report:   %s\n", outInfo.ReportHTML)
	fmt.Printf("summary:  %s\n", outInfo.Summary)
	fmt.Printf("ai:       %s\n", outInfo.AIContext)
	if *open {
		return openPath(outInfo.ReportHTML)
	}
	return nil
}

func cmdPenetration(args []string) error {
	fs := flag.NewFlagSet("penetration", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	open := fs.Bool("open", false, "open report in browser")
	flagArgs, pos := cliutil.Split(args, "open")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	scenarioArg := ""
	if len(pos) > 0 {
		scenarioArg = strings.TrimSpace(pos[0])
	}
	if scenarioArg == "" {
		return fmt.Errorf("penetration requires a scenario path")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	full, err := sim.ResolveScenarioPath(paths, scenarioArg)
	if err != nil {
		return err
	}
	sc, err := sim.LoadScenario(full)
	if err != nil {
		return err
	}
	if sc.Mode != sim.ModePenetration {
		return fmt.Errorf("scenario mode is %q, want penetration", sc.Mode)
	}
	result, err := sim.RunPenetration(sc, time.Now())
	if err != nil {
		return err
	}
	command := "jutland-sim " + strings.Join(os.Args[1:], " ")
	outInfo, err := sim.WriteReport(paths, result, full, command)
	if err != nil {
		return err
	}
	fmt.Printf("scenario: %s (penetration, F0)\n", sc.Name)
	fmt.Printf("run:      %s\n", outInfo.RunID)
	fmt.Printf("report:   %s\n", outInfo.ReportHTML)
	fmt.Printf("summary:  %s\n", outInfo.Summary)
	fmt.Printf("ai:       %s\n", outInfo.AIContext)
	if *open {
		return openPath(outInfo.ReportHTML)
	}
	return nil
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	mock := fs.Bool("mock", false, "generate the P1 synthetic report instead of a real simulation")
	out := fs.String("out", "", "override report directory")
	open := fs.Bool("open", false, "open report in browser")
	seed := fs.Int64("seed", -1, "override scenario seed")
	runs := fs.Int("runs", 0, "override run count")
	flagArgs, pos := cliutil.Split(args, "open", "mock")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if !*mock {
		return fmt.Errorf(
			"real simulation is not implemented yet (P2/P3); use `jutland-sim run --mock` or `jutland-sim demo`",
		)
	}
	scenarioArg := ""
	if len(pos) > 0 {
		scenarioArg = strings.TrimSpace(pos[0])
	}
	if scenarioArg == "" {
		return fmt.Errorf("run --mock requires a scenario path")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	full, err := sim.ResolveScenarioPath(paths, scenarioArg)
	if err != nil {
		return err
	}
	sc, err := sim.LoadScenario(full)
	if err != nil {
		return err
	}
	return runMock(paths, *sc, full, *seed, *runs, *open)
}

func cmdTrend(args []string) error {
	fs := flag.NewFlagSet("trend", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	open := fs.Bool("open", false, "open trend.html")
	limit := fs.Int("limit", 50, "history window size")
	flagArgs, pos := cliutil.Split(args, "open")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	_ = pos
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	info, err := sim.WriteTrend(paths, *limit)
	if err != nil {
		return err
	}
	fmt.Printf("trend:  %s\n", info.HTML)
	fmt.Printf("summary: %s\n", info.MD)
	fmt.Printf("ai:     %s\n", info.AIJSON)
	if *open {
		return openPath(info.HTML)
	}
	return nil
}

func cmdAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	limit := fs.Int("limit", 20, "history window size")
	flagArgs, _ := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	info, err := sim.WriteAnalysisBrief(paths, *limit)
	if err != nil {
		return err
	}
	fmt.Printf("brief: %s\n", info.Markdown)
	fmt.Printf("json:  %s\n", info.JSON)
	return nil
}

func cmdBaseline(args []string) error {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	name := fs.String("name", "", "baseline name (default scenario name)")
	tolerance := fs.String("tolerance", "", "metric=value[,metric=value]")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	runArg := ""
	if len(pos) > 0 {
		runArg = strings.TrimSpace(pos[0])
	}
	if runArg == "" {
		return fmt.Errorf("baseline requires a run id or result.json path")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	tolerances, err := sim.ParseTolerances(*tolerance)
	if err != nil {
		return err
	}
	path, err := sim.WriteBaselineFromRun(paths, runArg, *name, tolerances)
	if err != nil {
		return err
	}
	fmt.Printf("baseline: %s\n", path)
	return nil
}

func cmdPerf(args []string) error {
	fs := flag.NewFlagSet("perf", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	budget := fs.Float64("budget-ms", 16.7, "tick p99 budget in milliseconds")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	runArg := ""
	if len(pos) > 0 {
		runArg = strings.TrimSpace(pos[0])
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	report, err := sim.CheckPerf(paths, runArg, *budget)
	if err != nil {
		return err
	}
	fmt.Printf("run:    %s (%s)\n", report.RunID, report.Fidelity)
	fmt.Printf("p99:    %.3f ms (budget %.3f ms)\n", report.P99MS, report.BudgetMS)
	fmt.Printf("result: %v\n", report.Pass)
	if !report.Pass {
		return fmt.Errorf("p99 tick regression: %.3f ms > %.3f ms", report.P99MS, report.BudgetMS)
	}
	return nil
}

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) < 1 {
		return fmt.Errorf("compare requires a run id (or two run ids)")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	var info sim.CompareOutput
	if len(pos) >= 2 {
		info, err = sim.WriteComparison(paths, strings.TrimSpace(pos[0]), strings.TrimSpace(pos[1]))
	} else {
		info, err = sim.WriteComparisonToPrevious(paths, strings.TrimSpace(pos[0]))
	}
	if err != nil {
		return err
	}
	fmt.Printf("compare: %s\n", info.Markdown)
	fmt.Printf("json:    %s\n", info.JSON)
	return nil
}

func cmdWatch(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	interval := fs.Duration("interval", 10*time.Minute, "refresh interval")
	limit := fs.Int("limit", 50, "history window size")
	once := fs.Bool("once", false, "run one refresh and exit")
	flagArgs, _ := cliutil.Split(args, "once")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	return sim.RunWatch(context.Background(), paths, sim.WatchOptions{
		Interval: *interval, Limit: *limit, Once: *once, Writer: os.Stdout,
	})
}

func cmdArchive(args []string) error {
	fs := flag.NewFlagSet("archive", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	keep := fs.Int("keep", 50, "number of newest runs to keep")
	dryRun := fs.Bool("dry-run", false, "print the archive plan without moving files")
	flagArgs, _ := cliutil.Split(args, "dry-run")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	result, err := sim.ArchiveRuns(paths, sim.ArchiveOptions{Keep: *keep, DryRun: *dryRun})
	if err != nil {
		return err
	}
	fmt.Printf("kept:    %d\n", len(result.Kept))
	fmt.Printf("moved:   %d\n", len(result.Moved))
	if result.ArchiveDir != "" {
		fmt.Printf("archive: %s\n", result.ArchiveDir)
	}
	if *dryRun {
		fmt.Println("dry-run: no files moved")
	}
	return nil
}

func cmdCompareRecord(args []string) error {
	fs := flag.NewFlagSet("compare-record", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) < 2 {
		return fmt.Errorf("compare-record requires <recording.jsonl> <run>")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	info, err := sim.WriteRecordingComparison(paths, strings.TrimSpace(pos[0]), strings.TrimSpace(pos[1]))
	if err != nil {
		return err
	}
	fmt.Printf("compare: %s\n", info.Markdown)
	fmt.Printf("json:    %s\n", info.JSON)
	return nil
}

func cmdLLMPackage(args []string) error {
	fs := flag.NewFlagSet("llm-package", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) < 1 {
		return fmt.Errorf("llm-package requires a run id")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	dir, err := sim.WriteLLMPackage(paths, strings.TrimSpace(pos[0]))
	if err != nil {
		return err
	}
	fmt.Printf("package: %s\n", dir)
	return nil
}

func cmdLLMReview(args []string) error {
	fs := flag.NewFlagSet("llm-review", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	dryRun := fs.Bool("dry-run", false, "write request JSON without calling the API")
	flagArgs, pos := cliutil.Split(args, "dry-run")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) < 1 {
		return fmt.Errorf("llm-review requires a run id")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	result, err := sim.ReviewRun(paths, strings.TrimSpace(pos[0]), *dryRun)
	if err != nil {
		return err
	}
	if result == "" {
		fmt.Println("dry-run: request written to ai/llm-request.json")
	} else {
		fmt.Printf("review: %s\n", result)
	}
	return nil
}

func cmdEvaluation(args []string) error {
	fs := flag.NewFlagSet("evaluation", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	failOnRed := fs.Bool("fail-on-red", false, "exit non-zero on missing runs or smoke red items")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) < 1 {
		return fmt.Errorf("evaluation requires a suite summary JSON")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	report, html, err := sim.WriteEvaluation(paths, strings.TrimSpace(pos[0]))
	if err != nil {
		return err
	}
	fmt.Printf("evaluation: %s\n", html)
	if *failOnRed {
		if err := sim.EvaluationGateError(report); err != nil {
			return err
		}
		fmt.Println("gate: pass (no missing runs or smoke red items)")
	}
	return nil
}

func cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	scenarioPath := fs.String("scenario", "", "scenario path")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	arg := strings.TrimSpace(*scenarioPath)
	if arg == "" && len(pos) > 0 {
		arg = strings.TrimSpace(pos[0])
	}
	if arg == "" {
		return fmt.Errorf("validate requires a scenario path")
	}
	paths, err := sim.ResolvePaths("")
	if err != nil {
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
	fmt.Printf("valid: %s\n", full)
	fmt.Printf(
		"  name: %s\n  mode: %s\n  seed: %d\n  runs: %d\n  maxTicks: %d\n",
		sc.Name,
		sc.Mode,
		sc.Seed,
		sc.Runs,
		sc.MaxTicks,
	)
	return nil
}

func runMock(paths sim.Paths, sc sim.Scenario, scenarioPath string, seed int64, runs int, open bool) error {
	if seed >= 0 {
		sc.Seed = seed
	}
	if runs > 0 {
		sc.Runs = runs
	}
	sc.ApplyDefaults()
	if err := sc.Validate(); err != nil {
		return err
	}
	startedAt := time.Now()
	result := sim.GenerateDemoResult(&sc, startedAt)
	command := "jutland-sim " + strings.Join(os.Args[1:], " ")
	out, err := sim.WriteReport(paths, result, scenarioPath, command)
	if err != nil {
		return err
	}
	fmt.Printf("scenario: %s (%s)\n", sc.Name, sc.Mode)
	fmt.Printf("run:      %s\n", out.RunID)
	fmt.Printf("report:   %s\n", out.ReportHTML)
	fmt.Printf("summary:  %s\n", out.Summary)
	fmt.Printf("ai:       %s\n", out.AIContext)
	fmt.Printf("index:    %s\n", filepath.Join(paths.Reports, "index.html"))
	if open {
		return openPath(out.ReportHTML)
	}
	return nil
}

func writeIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
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

func printUsage() {
	fmt.Print(usageText)
}

const usageText = `jutland-sim - headless simulation and balance report tool

Requires JUTLAND_SIM_HOME (private lab root). It has no default and must not be empty.

Commands:
  init                         create the lab skeleton and a demo scenario
  demo [flags]                 generate the P1 synthetic HTML/ECharts report
  shot <scenario>              run the F0 pure-damage bench and write a real report
  run --mock <scenario>        same as demo, but requires a scenario path
  validate <scenario>          parse and validate a scenario
  version                      print version information
  help                         show this help

Common flags:
  --out DIR                    override report directory (default $JUTLAND_SIM_HOME/reports)
  --open                       open report.html in the browser

Errors:
  JUTLAND_SIM_HOME is missing/empty -> exit 1 with an export example; no files are written.
`

const sampleScenario = `{
  schemaVersion: 1,
  name: "demo-report",
  description: "P1 报告版式 mock：合成数据，不代表真实模拟结果。",
  mode: "demo",
  seed: 42,
  runs: 100,
  maxTicks: 3600,
  environment: {
    map: "open_sea",
    fog: false,
    friendlyFire: false,
    speedMultiplier: 1.0,
  },
  notes: ["合成数据，仅用于确认报告版式。"],
}
`

const labReadme = `# Jutland simulation lab

This directory is the private lab for Jutland balance reports.

- ` + "`scenarios/`" + `: experiment scenarios (commit)
- ` + "`golden/`" + `: regression snapshots (commit)
- ` + "`baselines/`" + `: local/formal baselines
- ` + "`notes/`" + `: experiment notes and decisions
- ` + "`reports/`" + `: generated reports (do not commit to the game repo)

Run:

` + "```bash" + `
export JUTLAND_SIM_HOME="$HOME/jutland-sim-lab"
jutland-sim init
jutland-sim demo --open
` + "```" + `
`
