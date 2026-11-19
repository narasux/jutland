package manager

import (
	"errors"
	"fmt"

	"github.com/narasux/jutland/pkg/mission/controller/computer"
	"github.com/narasux/jutland/pkg/mission/faction"
	instr "github.com/narasux/jutland/pkg/mission/instruction"
	"github.com/narasux/jutland/pkg/mission/object"
	objUnit "github.com/narasux/jutland/pkg/mission/object/unit"
	"github.com/narasux/jutland/pkg/mission/state"
	"github.com/narasux/jutland/pkg/mission/targeting"
)

// SimulationOptions controls the headless simulation path.
type SimulationOptions struct {
	// Headless skips UI, input, camera, audio and wall-clock-dependent work.
	Headless bool
	// SyncTargeting computes target plans inline instead of via a goroutine.
	SyncTargeting bool
	// SkipCommands disables AI/input instruction generation. Queued scenario
	// instructions still execute; Micro benches that fire directly set this true.
	SkipCommands bool
}

// NewWithState builds a manager around an already-constructed MissionState.
// It is intended for tests/headless benches that must not create the game
// drawer/sidebar/audio. The caller must use SetSimulationOptions before update.
func NewWithState(ms *state.MissionState) *MissionManager {
	m := &MissionManager{
		state:              ms,
		instructionSet:     NewInstructionSet(),
		targetingResults:   make(chan targeting.Plan, 1),
		targetingCursors:   map[string]map[object.Type]int{},
		takeoffTypeCursors: map[string]int{},
		targetingDirty:     true,
	}
	m.playerAlphaHandler = computer.NewHandler(faction.HumanAlpha)
	m.playerBetaHandler = computer.NewHandler(faction.ComputerAlpha)
	return m
}

// SetSimulationOptions configures the manager for a headless run.
// The human-side input handler is replaced with the computer handler so no
// Ebiten input is read; this is one-way for the lifetime of the manager.
func (m *MissionManager) SetSimulationOptions(opts SimulationOptions) {
	m.headless = opts.Headless
	m.skipCommands = opts.SkipCommands
	if opts.Headless {
		m.playerAlphaHandler = computer.NewHandler(faction.HumanAlpha)
		instr.SetSyncPathSearch(true)
	} else {
		instr.SetSyncPathSearch(false)
	}
	m.SetSyncTargeting(opts.Headless || opts.SyncTargeting)
}

// UpdateHeadless advances one simulation tick without UI, input, camera,
// audio or render work. It uses the same command/support/combat/status/stats
// phases as the game loop.
func (m *MissionManager) UpdateHeadless() (state.MissionStatus, error) {
	if m.state == nil {
		return "", errors.New("mission state is nil")
	}
	if !m.headless {
		return m.state.Core.MissionStatus, errors.New("headless mode is not enabled")
	}
	status := m.state.Core.MissionStatus
	if !missionStatusRunsSimulation(status) {
		return status, nil
	}
	m.advanceSimulationTick(nil, nil)
	return m.state.Core.MissionStatus, nil
}

// advanceSimulationTick is the shared phase order for one simulated tick.
// beforeSupport/beforeCombat are presentation-only hooks used by the game loop
// for camera, map prewarm and selection updates; headless passes nil.
func (m *MissionManager) advanceSimulationTick(beforeSupport, beforeCombat func()) {
	m.simTick++
	m.state.Core.SimTick = m.simTick
	objUnit.SetSimTick(m.simTick)
	if !m.skipCommands {
		// Input/AI may add instructions; execute them even when AI input is
		// skipped so synthetic sortie/move orders still run.
		m.updateInstructions()
	}
	m.executeInstructions()
	if beforeSupport != nil {
		beforeSupport()
	}
	m.updateSupportPhase()
	if beforeCombat != nil {
		beforeCombat()
	}
	m.updateCombatPhase()
	m.updateMissionStatus()
	m.finalizeBattleStats()
}

// State exposes the manager's mission state for read-only headless reporting.
// Do not mutate it while the manager is updating.
func (m *MissionManager) State() *state.MissionState {
	return m.state
}

// SimulationTick returns the current headless simulation tick.
func (m *MissionManager) SimulationTick() int64 {
	return m.simTick
}

// SetMissionStatus is a test/setup helper for headless scenarios that need to
// start from a non-default status without touching UI input.
func (m *MissionManager) SetMissionStatus(status state.MissionStatus) {
	if m.state != nil {
		m.state.Core.MissionStatus = status
	}
}

// String makes SimulationOptions printable in manifests/logs.
func (o SimulationOptions) String() string {
	return fmt.Sprintf("headless=%t syncTargeting=%t skipCommands=%t", o.Headless, o.SyncTargeting, o.SkipCommands)
}
