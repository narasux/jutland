package tutorial

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/yosuke-furukawa/json5/encoding/json5"

	"github.com/narasux/jutland/pkg/config"
	"github.com/narasux/jutland/pkg/i18n"
	"github.com/narasux/jutland/pkg/mission/metadata"
	objPos "github.com/narasux/jutland/pkg/mission/object/position"
)

type rawScript struct {
	ID          string    `json:"id"`
	Mission     string    `json:"mission"`
	Title       string    `json:"title"`
	TitleEn     string    `json:"titleEn"`
	TitleRu     string    `json:"titleRu"`
	TitleJa     string    `json:"titleJa"`
	TitleZhHant string    `json:"titleZhHant"`
	Steps       []rawStep `json:"steps"`
}

type rawStep struct {
	ID          string        `json:"id"`
	Mode        string        `json:"mode"`
	Title       string        `json:"title"`
	TitleEn     string        `json:"titleEn"`
	TitleRu     string        `json:"titleRu"`
	TitleJa     string        `json:"titleJa"`
	TitleZhHant string        `json:"titleZhHant"`
	Body        string        `json:"body"`
	BodyEn      string        `json:"bodyEn"`
	BodyRu      string        `json:"bodyRu"`
	BodyJa      string        `json:"bodyJa"`
	BodyZhHant  string        `json:"bodyZhHant"`
	Hint        rawHint       `json:"hint"`
	Setup       []string      `json:"setup"`
	Completion  rawCompletion `json:"completion"`
	AllowSkip   *bool         `json:"allowSkip"`
	Target      []int         `json:"target"`
}

type rawHint struct {
	Actions []string `json:"actions"`
	Hold    bool     `json:"hold"`
}

type rawCompletion struct {
	Kind     string  `json:"kind"`
	Min      int     `json:"min"`
	Distance float64 `json:"distance"`
	Group    int     `json:"group"`
}

var (
	scriptsByMission = map[string]*Script{}
	scriptsByID      = map[string]*Script{}
)

func init() {
	loadScripts()
}

func loadScripts() {
	path := filepath.Join(config.ConfigBaseDir, "tutorials.json5")
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("failed to open tutorials.json5: ", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("failed to read tutorials.json5: ", err)
	}

	var raws []rawScript
	if err = json5.Unmarshal(bytes, &raws); err != nil {
		log.Fatal("failed to unmarshal tutorials.json5: ", err)
	}

	for idx := range raws {
		script := buildScript(raws[idx])
		if _, exists := scriptsByID[script.ID]; exists {
			log.Fatalf("duplicate tutorial script id %q", script.ID)
		}
		if _, exists := scriptsByMission[script.Mission]; exists {
			log.Fatalf("duplicate tutorial script for mission %q", script.Mission)
		}
		scriptsByID[script.ID] = script
		scriptsByMission[script.Mission] = script
	}
	validateScripts()
}

func buildScript(raw rawScript) *Script {
	if raw.ID == "" {
		log.Fatal("tutorial script id must not be empty")
	}
	if raw.Mission == "" {
		log.Fatalf("tutorial script %q mission must not be empty", raw.ID)
	}
	script := &Script{
		ID:      raw.ID,
		Mission: raw.Mission,
		Title:   newLocalized(raw.Title, raw.TitleZhHant, raw.TitleEn, raw.TitleRu, raw.TitleJa),
		Steps:   make([]Step, 0, len(raw.Steps)),
	}
	validateLocalized(script.Title, "tutorial script "+raw.ID+" title")
	stepIDs := make(map[string]struct{}, len(raw.Steps))
	for idx := range raw.Steps {
		step := buildStep(raw.ID, raw.Steps[idx])
		if _, exists := stepIDs[step.ID]; exists {
			log.Fatalf("tutorial script %q has duplicate step id %q", raw.ID, step.ID)
		}
		stepIDs[step.ID] = struct{}{}
		script.Steps = append(script.Steps, step)
	}
	if len(script.Steps) == 0 {
		log.Fatalf("tutorial script %q must have at least one step", raw.ID)
	}
	return script
}

func buildStep(scriptID string, raw rawStep) Step {
	if raw.ID == "" {
		log.Fatalf("tutorial script %q has a step with empty id", scriptID)
	}
	prefix := "tutorial " + scriptID + " step " + raw.ID
	mode := Mode(raw.Mode)
	if mode == "" {
		mode = ModeGuided
	}
	if !mode.Valid() {
		log.Fatalf("%s has invalid mode %q", prefix, raw.Mode)
	}

	step := Step{
		ID:    raw.ID,
		Mode:  mode,
		Title: newLocalized(raw.Title, raw.TitleZhHant, raw.TitleEn, raw.TitleRu, raw.TitleJa),
		Body:  newLocalized(raw.Body, raw.BodyZhHant, raw.BodyEn, raw.BodyRu, raw.BodyJa),
		Hint: Hint{
			Hold: raw.Hint.Hold,
		},
		Completion: Completion{
			Kind:     CompletionKind(raw.Completion.Kind),
			Min:      raw.Completion.Min,
			Distance: raw.Completion.Distance,
			Group:    raw.Completion.Group,
		},
		AllowSkip: raw.AllowSkip == nil || *raw.AllowSkip,
	}
	if len(raw.Target) != 0 {
		if len(raw.Target) != 2 {
			log.Fatalf("%s target must be [x, y]", prefix)
		}
		point := objPos.New(raw.Target[0], raw.Target[1])
		step.Target = &point
	}
	validateLocalized(step.Title, prefix+" title")
	validateLocalized(step.Body, prefix+" body")
	if step.Completion.Kind == "" {
		step.Completion.Kind = CompletionManual
	}
	if !step.Completion.Kind.Valid() {
		log.Fatalf("%s has invalid completion kind %q", prefix, raw.Completion.Kind)
	}
	if step.Completion.Min < 0 {
		log.Fatalf("%s min must be >= 0", prefix)
	}
	if step.Completion.Distance < 0 {
		log.Fatalf("%s distance must be >= 0", prefix)
	}
	if step.Completion.Kind == CompletionSelectedShips && step.Completion.Min == 0 {
		step.Completion.Min = 1
	}
	if step.Completion.Kind == CompletionShipMoved && step.Completion.Distance == 0 {
		step.Completion.Distance = 2
	}
	if step.Completion.Kind == CompletionGroupAssigned ||
		step.Completion.Kind == CompletionGroupSelected {
		if step.Completion.Group == 0 {
			step.Completion.Group = 1
		}
	}
	if mode == ModeGuided && step.Completion.Kind == CompletionManual {
		log.Fatalf("%s is guided but uses manual completion; use info or a detectable condition", prefix)
	}
	if mode == ModeInfo && step.Completion.Kind != CompletionManual {
		log.Fatalf("%s is info but uses %q completion", prefix, step.Completion.Kind)
	}

	for _, action := range raw.Hint.Actions {
		if !ActionValid(ActionID(action)) {
			log.Fatalf("%s has unknown hint action %q", prefix, action)
		}
		step.Hint.Actions = append(step.Hint.Actions, ActionID(action))
	}
	for _, setup := range raw.Setup {
		action := SetupAction(setup)
		if !action.Valid() {
			log.Fatalf("%s has unknown setup action %q", prefix, setup)
		}
		step.Setup = append(step.Setup, action)
	}
	return step
}

func validateLocalized(value Localized, name string) {
	for _, lang := range []i18n.Language{
		i18n.LanguageZhHans,
		i18n.LanguageZhHant,
		i18n.LanguageEnglish,
		i18n.LanguageRussian,
		i18n.LanguageJapanese,
	} {
		if value.values[lang] == "" {
			log.Fatalf("%s is missing locale %s", name, lang)
		}
	}
}

func validateScripts() {
	for _, mission := range metadata.AllMissions() {
		md := metadata.Get(mission)
		if md.Category != metadata.MissionCategoryTutorial {
			continue
		}
		script, ok := scriptsByMission[mission]
		if !ok {
			log.Fatalf("tutorial mission %q has no tutorial script", mission)
		}
		if md.TutorialScript != "" && md.TutorialScript != script.ID {
			log.Fatalf(
				"tutorial mission %q references script %q but config provides %q",
				mission, md.TutorialScript, script.ID,
			)
		}
	}
	for mission, script := range scriptsByMission {
		md := metadata.Get(mission)
		if md.Name == "" {
			log.Fatalf("tutorial script %q references unknown mission %q", script.ID, mission)
		}
		if md.Category != metadata.MissionCategoryTutorial {
			log.Fatalf(
				"tutorial script %q references mission %q with category %q",
				script.ID, mission, md.Category,
			)
		}
	}
}

// ScriptForMission 返回指定任务的教学脚本。
func ScriptForMission(mission string) (*Script, bool) {
	script, ok := scriptsByMission[mission]
	return script, ok
}

// StepCount 返回教学任务的步骤数；非教学任务返回 0。
func StepCount(mission string) int {
	if script, ok := scriptsByMission[mission]; ok {
		return len(script.Steps)
	}
	return 0
}
