package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/narasux/jutland/internal/cliutil"
	_ "github.com/narasux/jutland/pkg/mission/object/initialize"
	"github.com/narasux/jutland/pkg/mission/sim"
	"github.com/narasux/jutland/pkg/mission/sim/duel"
	"github.com/narasux/jutland/pkg/mission/sim/macro"
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
	if args[0] != "suite" {
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
	return cmdSuite(args[1:])
}

func cmdSuite(args []string) error {
	fs := flag.NewFlagSet("suite", flag.ContinueOnError)
	out := fs.String("out", "", "override report directory")
	failOnRed := fs.Bool("fail-on-red", false, "exit non-zero on missing runs or smoke red items")
	flagArgs, pos := cliutil.Split(args)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if len(pos) == 0 {
		return fmt.Errorf("suite requires a manifest path")
	}
	paths, err := sim.ResolvePaths(*out)
	if err != nil {
		return err
	}
	if err := sim.ValidateLayout(paths); err != nil {
		return err
	}
	suitePath, err := sim.ResolveSuitePath(paths, strings.TrimSpace(pos[0]))
	if err != nil {
		return err
	}
	suite, err := sim.LoadSuite(suitePath)
	if err != nil {
		return err
	}
	started := time.Now()
	var runs []sim.SuiteRunResult
	for stepIdx, step := range suite.Steps {
		scenarioPath, err := sim.ResolveScenarioPath(paths, step.Scenario)
		if err != nil {
			return err
		}
		sc, err := sim.LoadScenario(scenarioPath)
		if err != nil {
			return err
		}
		if sc.Mode != step.Mode {
			return fmt.Errorf("step %d scenario %s mode %s != %s", stepIdx, step.Scenario, sc.Mode, step.Mode)
		}
		seeds := step.Seeds
		if len(seeds) == 0 {
			seeds = []int64{sc.Seed}
		}
		for _, seed := range seeds {
			runSc := *sc
			runSc.Seed = seed
			if step.Ticks > 0 {
				runSc.MaxTicks = step.Ticks
				if runSc.Duel != nil {
					runSc.Duel.MaxTicks = step.Ticks
				}
				if runSc.Macro != nil {
					runSc.Macro.MaxTicks = step.Ticks
				}
			}
			if err := runSc.Validate(); err != nil {
				return fmt.Errorf("step %d seed %d: %w", stepIdx, seed, err)
			}
			command := fmt.Sprintf("jutland-suite suite %s", suitePath)
			var result *sim.Result
			switch step.Mode {
			case sim.ModeShot:
				runSc.Runs = step.Runs
				result, err = sim.RunShot(&runSc, time.Now())
			case sim.ModePenetration:
				result, err = sim.RunPenetration(&runSc, time.Now())
			case sim.ModeDuel:
				result, err = duel.RunDuel(&runSc, time.Now())
			case sim.ModeMacro:
				result, err = macro.RunMacro(&runSc, time.Now())
			}
			if err != nil {
				return fmt.Errorf("step %d seed %d: %w", stepIdx, seed, err)
			}
			info, err := sim.WriteReport(paths, result, scenarioPath, command)
			if err != nil {
				return err
			}
			runs = append(runs, sim.SuiteRunResult{
				Step: stepIdx, Mode: step.Mode, Scenario: step.Scenario, Seed: seed,
				RunID: info.RunID, Fidelity: result.Fidelity, ReportHTML: info.ReportHTML,
			})
			fmt.Printf("%s %s seed=%d -> %s\n", step.Mode, step.Scenario, seed, info.RunID)
		}
	}
	summary, err := sim.WriteSuiteSummary(paths, suite, runs, "jutland-suite suite "+suitePath, started)
	if err != nil {
		return err
	}
	fmt.Printf("suite: %s\n", summary)
	report, evaluation, err := sim.WriteEvaluation(paths, summary)
	if err != nil {
		return err
	}
	fmt.Printf("evaluation: %s\n", evaluation)
	if *failOnRed {
		if err := sim.EvaluationGateError(report); err != nil {
			return err
		}
		fmt.Println("gate: pass (no missing runs or smoke red items)")
	}
	return nil
}

const usage = `jutland-suite - JSON5 simulation suite runner

Requires JUTLAND_SIM_HOME. F1/F2/F3 steps import game resources/Ebiten.

Commands:
  suite <manifest> [--out DIR] [--fail-on-red]
  help

--fail-on-red turns the suite into a CI gate: it fails on missing runs or smoke
red items (balance reds stay informational).
`
