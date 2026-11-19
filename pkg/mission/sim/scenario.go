package sim

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yosuke-furukawa/json5/encoding/json5"
)

// Mode selects the runner used by a scenario.
type Mode string

const (
	// ModeDemo is the P1 synthetic report mode.
	ModeDemo Mode = "demo"
	// ModeShot is the single-shot/damage bench.
	ModeShot Mode = "shot"
	// ModeMicro is a controlled multi-unit engagement.
	ModeMicro Mode = "micro"
	// ModeDuel is a platform duel.
	ModeDuel Mode = "duel"
	// ModeStrike is an air strike / air defense scenario.
	ModeStrike Mode = "strike"
	// ModeMacro is a full mission AI-vs-AI run.
	ModeMacro Mode = "macro"
	// ModePenetration is the pure geometric penetration matrix.
	ModePenetration Mode = "penetration"
)

// Environment describes the scenario environment.
type Environment struct {
	Map             string  `json:"map"`
	Fog             bool    `json:"fog"`
	FriendlyFire    bool    `json:"friendlyFire"`
	SpeedMultiplier float64 `json:"speedMultiplier"`
}

// UnitSpec is a single ship or aircraft in a scenario.
type UnitSpec struct {
	ID         string    `json:"id"`
	Ship       string    `json:"ship"`
	Plane      string    `json:"plane"`
	Count      int       `json:"count"`
	Pos        []float64 `json:"pos"`
	Rotation   float64   `json:"rotation"`
	SpeedRatio float64   `json:"speedRatio"`
	Target     string    `json:"target"`
	Weapons    []string  `json:"weapons"`
	Ammo       string    `json:"ammo"`
}

// ShotSpec defines the F0 pure-damage bench: one target profile and one ammo.
type ShotSpec struct {
	Name                string   `json:"name"`
	TargetName          string   `json:"targetName"`
	TargetType          string   `json:"targetType"`
	TargetHP            float64  `json:"targetHP"`
	HorizontalReduction float64  `json:"horizontalReduction"`
	VerticalReduction   float64  `json:"verticalReduction"`
	AmmoName            string   `json:"ammoName"`
	AmmoType            string   `json:"ammoType"`
	Diameter            int      `json:"diameter"`
	BaseDamage          float64  `json:"baseDamage"`
	CriticalRate        float64  `json:"criticalRate"`
	Plunge              float64  `json:"plunge"`
	Count               int      `json:"count"`
	ExpectedDamage      *float64 `json:"expectedDamage,omitempty"`
	Tolerance           float64  `json:"tolerance"`
}

// PenetrationSpec defines a pure geometric penetration matrix.
type PenetrationSpec struct {
	AmmoName       string  `json:"ammoName,omitempty"`
	PenetrationMM  float64 `json:"penetrationMM"`
	ArmorMin       float64 `json:"armorMin"`
	ArmorMax       float64 `json:"armorMax"`
	AngleMin       float64 `json:"angleMin,omitempty"`
	AngleMax       float64 `json:"angleMax,omitempty"`
	Steps          int     `json:"steps,omitempty"`
	ReferenceArmor float64 `json:"referenceArmor,omitempty"`
	ReferenceAngle float64 `json:"referenceAngle,omitempty"`
}

// MacroSpec defines an AI-vs-AI full mission run.
type MacroSpec struct {
	Mission    string `json:"mission"`
	PlayerSide string `json:"playerSide,omitempty"` // P1 | P2
	MaxTicks   int    `json:"maxTicks,omitempty"`
}

// DuelShip is one ship group in a duel fleet.
type DuelShip struct {
	Ship      string  `json:"ship"`
	Count     int     `json:"count,omitempty"`
	InitialHP float64 `json:"initialHP,omitempty"`
}

// DuelSpec defines the F1 single- or multi-ship duel using real ship/weapon configs.
type DuelSpec struct {
	ShooterShip      string     `json:"shooterShip,omitempty"`
	TargetShip       string     `json:"targetShip,omitempty"`
	ShooterFleet     []DuelShip `json:"shooterFleet,omitempty"`
	TargetFleet      []DuelShip `json:"targetFleet,omitempty"`
	Distance         float64    `json:"distance"`
	ShooterRotation  float64    `json:"shooterRotation"`
	TargetRotation   float64    `json:"targetRotation"`
	ShooterInitialHP float64    `json:"shooterInitialHP,omitempty"`
	TargetInitialHP  float64    `json:"targetInitialHP,omitempty"`
	Doctrine         string     `json:"doctrine,omitempty"` // auto | focus
	Terrain          []string   `json:"terrain,omitempty"`  // optional ASCII map rows
	MaxTicks         int        `json:"maxTicks,omitempty"`
}

// Assertion is a scenario-level expectation checked by the runner.
type Assertion struct {
	Metric string `json:"metric"`
	Op     string `json:"op"`
	Value  any    `json:"value"`
}

// Scenario is the JSON5 scenario schema.
type Scenario struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Mode          Mode                   `json:"mode"`
	Seed          int64                  `json:"seed"`
	Runs          int                    `json:"runs"`
	MaxTicks      int                    `json:"maxTicks"`
	Environment   Environment            `json:"environment"`
	Forces        map[string][]UnitSpec  `json:"forces"`
	Assertions    []Assertion            `json:"assertions"`
	Sweep         map[string][]any       `json:"sweep"`
	Shot          *ShotSpec              `json:"shot,omitempty"`
	Duel          *DuelSpec              `json:"duel,omitempty"`
	Macro         *MacroSpec             `json:"macro,omitempty"`
	Penetration   *PenetrationSpec       `json:"penetration,omitempty"`
	Notes         []string               `json:"notes"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// LoadScenario reads a JSON5 scenario, applies defaults and validates it.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scenario %s: %w", path, err)
	}
	var scenario Scenario
	if err := json5.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("parse scenario %s: %w", path, err)
	}
	scenario.ApplyDefaults()
	if err := scenario.Validate(); err != nil {
		return nil, fmt.Errorf("scenario %s: %w", path, err)
	}
	return &scenario, nil
}

// ApplyDefaults fills optional fields with safe defaults.
func (s *Scenario) ApplyDefaults() {
	if s.SchemaVersion <= 0 {
		s.SchemaVersion = 1
	}
	if s.Mode == "" {
		s.Mode = ModeDemo
	}
	if s.Runs <= 0 {
		s.Runs = 1
	}
	if s.MaxTicks <= 0 {
		s.MaxTicks = 3600
	}
	if s.Environment.SpeedMultiplier <= 0 {
		s.Environment.SpeedMultiplier = 1.0
	}
	if s.Environment.Map == "" {
		s.Environment.Map = "open_sea"
	}
	if s.Shot != nil && s.Shot.Count <= 0 {
		s.Shot.Count = 1
	}
	if s.Shot != nil && s.Shot.TargetHP <= 0 {
		s.Shot.TargetHP = 1000
	}
	if s.Shot != nil && s.Shot.TargetName == "" {
		s.Shot.TargetName = "target"
	}
	if s.Shot != nil && s.Shot.TargetType == "" {
		s.Shot.TargetType = "battleship"
	}
	if s.Duel != nil {
		if len(s.Duel.ShooterFleet) == 0 && s.Duel.ShooterShip != "" {
			s.Duel.ShooterFleet = []DuelShip{{Ship: s.Duel.ShooterShip, Count: 1}}
		}
		if len(s.Duel.TargetFleet) == 0 && s.Duel.TargetShip != "" {
			s.Duel.TargetFleet = []DuelShip{{Ship: s.Duel.TargetShip, Count: 1}}
		}
		for i := range s.Duel.ShooterFleet {
			if s.Duel.ShooterFleet[i].Count <= 0 {
				s.Duel.ShooterFleet[i].Count = 1
			}
		}
		for i := range s.Duel.TargetFleet {
			if s.Duel.TargetFleet[i].Count <= 0 {
				s.Duel.TargetFleet[i].Count = 1
			}
		}
		if s.Duel.Doctrine == "" {
			s.Duel.Doctrine = "auto"
		}
		if s.Duel.Distance <= 0 {
			s.Duel.Distance = 10
		}
		if s.Duel.ShooterRotation == 0 {
			s.Duel.ShooterRotation = 90
		}
		if s.Duel.TargetRotation == 0 {
			s.Duel.TargetRotation = 270
		}
		if s.Duel.MaxTicks <= 0 {
			s.Duel.MaxTicks = s.MaxTicks
		}
	}
}

// Validate checks the scenario bounds and required fields.
func (s *Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("name is required")
	}
	switch s.Mode {
	case ModeDemo, ModeShot, ModeMicro, ModeDuel, ModeStrike, ModeMacro, ModePenetration:
	default:
		return fmt.Errorf("unknown mode %q", s.Mode)
	}
	if s.Runs < 1 || s.Runs > 10000 {
		return fmt.Errorf("runs must be between 1 and 10000, got %d", s.Runs)
	}
	if s.MaxTicks < 1 || s.MaxTicks > 216000 {
		return fmt.Errorf("maxTicks must be between 1 and 216000, got %d", s.MaxTicks)
	}
	switch s.Environment.SpeedMultiplier {
	case 0.5, 1.0, 2.0:
	default:
		return fmt.Errorf("speedMultiplier must be 0.5, 1.0 or 2.0, got %v", s.Environment.SpeedMultiplier)
	}
	if s.Mode == ModePenetration {
		if s.Penetration == nil {
			return fmt.Errorf("penetration section is required for mode=penetration")
		}
		if s.Penetration.PenetrationMM <= 0 || s.Penetration.ArmorMin < 0 ||
			s.Penetration.ArmorMax <= s.Penetration.ArmorMin {
			return fmt.Errorf("penetration requires penetrationMM>0 and armorMax>armorMin>=0")
		}
		if s.Penetration.AngleMin < 0 || s.Penetration.AngleMax > 80 ||
			s.Penetration.AngleMax < s.Penetration.AngleMin {
			return fmt.Errorf("penetration angles must satisfy 0<=min<=max<=80")
		}
		if s.Penetration.Steps < 2 || s.Penetration.Steps > 100 {
			return fmt.Errorf("penetration.steps must be 2..100")
		}
		if s.Penetration.ReferenceArmor < s.Penetration.ArmorMin ||
			s.Penetration.ReferenceArmor > s.Penetration.ArmorMax {
			return fmt.Errorf("penetration.referenceArmor must be inside armor range")
		}
	}
	if s.Mode == ModeMacro {
		if s.Macro == nil {
			return fmt.Errorf("macro section is required for mode=macro")
		}
		if strings.TrimSpace(s.Macro.Mission) == "" {
			return fmt.Errorf("macro.mission is required")
		}
		if s.Macro.PlayerSide != "P1" && s.Macro.PlayerSide != "P2" {
			return fmt.Errorf("macro.playerSide must be P1 or P2")
		}
		if s.Macro.MaxTicks < 1 || s.Macro.MaxTicks > 216000 {
			return fmt.Errorf("macro.maxTicks must be between 1 and 216000")
		}
	}
	if s.Mode == ModeDuel {
		if s.Duel == nil {
			return fmt.Errorf("duel section is required for mode=duel")
		}
		if len(s.Duel.ShooterFleet) == 0 || len(s.Duel.TargetFleet) == 0 {
			return fmt.Errorf("duel fleets must not be empty")
		}
		shooterCount, targetCount := 0, 0
		for _, group := range s.Duel.ShooterFleet {
			if group.Ship == "" {
				return fmt.Errorf("duel.shooterFleet.ship is required")
			}
			if group.Count < 1 || group.Count > 20 {
				return fmt.Errorf("duel.shooterFleet.count must be 1..20")
			}
			shooterCount += group.Count
		}
		for _, group := range s.Duel.TargetFleet {
			if group.Ship == "" {
				return fmt.Errorf("duel.targetFleet.ship is required")
			}
			if group.Count < 1 || group.Count > 20 {
				return fmt.Errorf("duel.targetFleet.count must be 1..20")
			}
			targetCount += group.Count
		}
		if shooterCount+targetCount > 100 {
			return fmt.Errorf("duel total ships must be <= 100")
		}
		if s.Duel.Doctrine != "auto" && s.Duel.Doctrine != "focus" {
			return fmt.Errorf("duel.doctrine must be auto or focus")
		}
		if len(s.Duel.Terrain) > 0 {
			width := len(s.Duel.Terrain[0])
			if width < 20 || len(s.Duel.Terrain) < 10 {
				return fmt.Errorf("duel.terrain must be at least 20x10")
			}
			for i, row := range s.Duel.Terrain {
				if len(row) != width {
					return fmt.Errorf("duel.terrain row %d has width %d, want %d", i, len(row), width)
				}
			}
		}
		if s.Duel.Distance <= 0 {
			return fmt.Errorf("duel.distance must be > 0")
		}
		if s.Duel.MaxTicks < 1 || s.Duel.MaxTicks > 216000 {
			return fmt.Errorf("duel.maxTicks must be between 1 and 216000")
		}
		if s.Duel.ShooterInitialHP < 0 || s.Duel.TargetInitialHP < 0 {
			return fmt.Errorf("duel initial HP must be >= 0")
		}
	}
	if s.Mode == ModeShot {
		if s.Shot == nil {
			return fmt.Errorf("shot section is required for mode=shot")
		}
		if s.Shot.TargetHP <= 0 {
			return fmt.Errorf("shot.targetHP must be > 0")
		}
		if s.Shot.BaseDamage <= 0 {
			return fmt.Errorf("shot.baseDamage must be > 0")
		}
		if s.Shot.Count < 1 || s.Shot.Count > 10000 {
			return fmt.Errorf("shot.count must be between 1 and 10000")
		}
		if s.Shot.HorizontalReduction < 0 || s.Shot.HorizontalReduction > 1 {
			return fmt.Errorf("shot.horizontalReduction must be 0..1")
		}
		if s.Shot.VerticalReduction < 0 || s.Shot.VerticalReduction > 1 {
			return fmt.Errorf("shot.verticalReduction must be 0..1")
		}
		if s.Shot.Plunge < 0 || s.Shot.Plunge > 1 {
			return fmt.Errorf("shot.plunge must be 0..1")
		}
		if s.Shot.CriticalRate < 0 || s.Shot.CriticalRate > 1 {
			return fmt.Errorf("shot.criticalRate must be 0..1")
		}
		if s.Shot.Tolerance < 0 {
			return fmt.Errorf("shot.tolerance must be >= 0")
		}
		if int64(s.Runs)*int64(s.Shot.Count) > 1_000_000 {
			return fmt.Errorf("shot total samples must be <= 1000000")
		}
	}
	for side, units := range s.Forces {
		for i, unit := range units {
			if unit.Ship == "" && unit.Plane == "" {
				return fmt.Errorf("forces.%s[%d] must set ship or plane", side, i)
			}
			if unit.Ship != "" && unit.Plane != "" {
				return fmt.Errorf("forces.%s[%d] cannot set both ship and plane", side, i)
			}
			if len(unit.Pos) != 0 && len(unit.Pos) != 2 {
				return fmt.Errorf("forces.%s[%d].pos must have 2 numbers", side, i)
			}
			if unit.SpeedRatio < 0 || unit.SpeedRatio > 1 {
				return fmt.Errorf("forces.%s[%d].speedRatio must be 0..1", side, i)
			}
		}
	}
	return nil
}

// Hash returns a stable short hash of the normalized scenario.
func (s *Scenario) Hash() string {
	data, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

// Slug returns a filesystem-safe scenario name.
func (s *Scenario) Slug() string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(s.Name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "scenario"
	}
	return out
}

// ResolveScenarioPath resolves an absolute or lab-relative scenario path.
func ResolveScenarioPath(paths Paths, arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("scenario path is required")
	}
	if filepath.IsAbs(arg) {
		return arg, nil
	}
	candidate := filepath.Join(paths.Scenarios, arg)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	if filepath.Ext(arg) == "" {
		withExt := candidate + ".json5"
		if _, err := os.Stat(withExt); err == nil {
			return withExt, nil
		}
	}
	return candidate, nil
}
